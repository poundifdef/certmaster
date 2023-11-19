package hetzner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/poundifdef/certmaster/models"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Destination struct {
	APIToken       string `credential:"true" mapstructure:"api_token" description:"Hetzner read/write API token"`
	LoadBalancerID string `mapstructure:"load_balancer_id" description:"ID of load balancer to replace certificate"`
}

func (d Destination) Upload(request models.CertRequest, cert *certificate.Resource) error {

	client := hcloud.NewClient(hcloud.WithToken(d.APIToken))

	// Add new cert to Hetzner
	certName := request.Domain + "-" + time.Now().Format("2006-01-02")
	opts := hcloud.CertificateCreateOpts{
		Name:        certName,
		Certificate: string(cert.Certificate),
		PrivateKey:  string(cert.PrivateKey),
	}
	hetznerCert, _, err := client.Certificate.Create(context.TODO(), opts)
	if err != nil {
		return err
	}

	// Get all services from LB
	lb, lbResp, err := client.LoadBalancer.Get(context.TODO(), d.LoadBalancerID)
	if lb == nil {
		return errors.New("LB does not exist")
	}
	if err != nil {
		return err
	}

	// Get LB info as JSON string
	lbRespBody, err := io.ReadAll(lbResp.Body)
	if err != nil {
		return err
	}

	// Find the service within the LB to update. This is just looking for a
	// service with a cert matching the domain that we're renewing
	// TODO: What if there are multiple services with a matching domain? We should update all of them.
	serviceIndexToUpdate, oldCertID, err := findServiceToUpdate(client, lb, request.Domain)

	// TODO: if we don't find any service, then create a new one with default values
	if err != nil {
		return err
	}
	if serviceIndexToUpdate < 0 {
		return errors.New("Unable to find service to update")
	}

	// Get the JSON payload for the service we need to update
	serviceJSON := gjson.Get(string(lbRespBody), fmt.Sprintf("load_balancer.services.%d", serviceIndexToUpdate)).Raw
	log.Println(serviceJSON)

	// Find which certificate ID we need to replace
	certJSON := gjson.Get(serviceJSON, "http.certificates")
	certIndex := -1
	for i, v := range certJSON.Array() {
		log.Println(v.Int())
		if v.Int() == int64(oldCertID) {
			certIndex = i
			break
		}
	}

	// TODO: If it doesn't exist, then just add the new cert
	if certIndex < 0 {
		return errors.New("Unable to find old cert to replace for LB")
	}

	newServiceJSON, err := sjson.Set(serviceJSON, fmt.Sprintf("http.certificates.%d", certIndex), hetznerCert.ID)
	if err != nil {
		return err
	}

	// Update the LB service to replace the old certificate ID with new one
	err = updateLBService(d.APIToken, d.LoadBalancerID, newServiceJSON)
	if err != nil {
		return err
	}

	return nil
}

func findServiceToUpdate(client *hcloud.Client, lb *hcloud.LoadBalancer, domain string) (int, int, error) {
	// Hetzner LBs expose "services" to the outside world.
	// Loop through each service in the LB and find the one that matches our domain.

	// For each service
	for serviceIndex, service := range lb.Services {
		log.Println(service.Protocol)
		serviceCerts := service.HTTP.Certificates
		if serviceCerts == nil || len(serviceCerts) == 0 {
			continue
		}

		// For each certificate in the service
		for _, serviceCertShell := range serviceCerts {
			serviceCert, _, err := client.Certificate.GetByID(context.TODO(), serviceCertShell.ID)
			if err != nil {
				return -1, 0, err
			}
			log.Println(serviceCert.ID, serviceCert.DomainNames)

			// For each domain name assiciated with the certificate
			for _, domainName := range serviceCert.DomainNames {

				// Does the domain name match the certificate?
				if strings.EqualFold(domainName, domain) {

					// Make sure there are not multiple domains associated with the cert.
					// Right now we don't support certs with multiple domains.
					if len(serviceCert.DomainNames) > 1 {
						return -1, 0, errors.New("Cannot replace certificte that has multiple domain names")
					}

					// TODO: what if there are multiple services associated with
					// this certificate on a LB? We should instead rturn a list of
					// services to update.
					return serviceIndex, serviceCert.ID, nil
				}
			}
		}
	}

	return -1, 0, errors.New("Could not find service with matching domain")
}

func updateLBService(bearerToken string, lbID string, jsonPayload string) error {
	url := fmt.Sprintf("https://api.hetzner.cloud/v1/load_balancers/%s/actions/update_service", lbID)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonPayload)))
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Add("Authorization", "Bearer "+bearerToken)
	req.Header.Add("Content-Type", "application/json")

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		return errors.New(string(body))
	}

	return nil
}
