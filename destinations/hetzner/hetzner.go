package hetzner

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/poundifdef/certmaster/models"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type Destination struct {
	APIToken         string `credential:"true" mapstructure:"api_token" description:"Hetzner read/write API token"`
	LoadBalancerName string `mapstructure:"load_balancer_name" description:"Name of load balancer to install certificate"`
	Port             int    `mapstructure:"port" description:"Port number where we will listen for certificate"`
}

func (d Destination) Description() string {
	return "Creates/updates certificate in a Hetzner load balancer"
}

func (d Destination) Upload(request models.CertRequest, cert *certificate.Resource) error {
	ctx := context.TODO()

	client := hcloud.NewClient(hcloud.WithToken(d.APIToken))

	// Add new cert to Hetzner
	certName := request.Domain + " " + time.Now().Format("(2006-01-02 15:04)")
	opts := hcloud.CertificateCreateOpts{
		Name:        certName,
		Certificate: string(cert.Certificate),
		PrivateKey:  string(cert.PrivateKey),
	}
	hetznerCert, _, err := client.Certificate.Create(ctx, opts)
	if err != nil {
		return err
	}

	// Get LB details
	lb, _, err := client.LoadBalancer.GetByName(ctx, d.LoadBalancerName)
	if lb == nil {
		return errors.New("LB does not exist")
	}
	if err != nil {
		return err
	}

	var service *hcloud.LoadBalancerService

	// Find the LB service with a matching port
	for i, lbService := range lb.Services {
		if lbService.ListenPort == d.Port {
			service = &lb.Services[i]
			break
		}
	}

	// Service doesn't exist, so we will create it and add the cert
	if service == nil {
		log.Println("Service on port", d.Port, "does not exist on LB, creating it")
		serviceOpts := hcloud.LoadBalancerAddServiceOpts{
			Protocol:        hcloud.LoadBalancerServiceProtocolHTTPS,
			ListenPort:      hcloud.Ptr[int](d.Port),
			DestinationPort: hcloud.Ptr[int](80),
			HTTP: &hcloud.LoadBalancerAddServiceOptsHTTP{
				Certificates: []*hcloud.Certificate{
					{
						ID: hetznerCert.ID,
					},
				},
			},
		}
		_, _, err := client.LoadBalancer.AddService(ctx, lb, serviceOpts)
		if err != nil {
			log.Println("Unable to create service on LB")
		}
		return err
	} else {
		// Service exists, so we update the cert

		found := false
		for i, certShell := range service.HTTP.Certificates {
			cert, _, err := client.Certificate.GetByID(ctx, certShell.ID)
			if err != nil {
				log.Println("Unable to get info about cert", certShell.ID)
				continue
			}
			for _, domainName := range cert.DomainNames {

				// Does the domain name match the certificate?
				log.Println(domainName, request.Domain)
				if strings.EqualFold(domainName, request.Domain) {

					// Make sure there are not multiple domains associated with the cert.
					// Right now we don't support certs with multiple domains.
					if len(cert.DomainNames) > 1 {
						return errors.New("Cannot replace certificte that has multiple domain names")
					}

					// Replace certificate
					log.Println("Replacing cert ", cert.ID, "with cert", hetznerCert.ID)
					service.HTTP.Certificates[i] = &hcloud.Certificate{ID: hetznerCert.ID}
					found = true
					break
				}

			}

			if found {
				break
			}
		}

		if !found {
			// Couldn't find a matching certificate to replace, so add new one to LB
			log.Println("Adding cert", hetznerCert.ID)
			service.HTTP.Certificates = append(service.HTTP.Certificates, &hcloud.Certificate{ID: hetznerCert.ID})
		}

		// Update LB service with updated cert
		_, _, err = client.LoadBalancer.UpdateService(ctx, lb, service.ListenPort, hcloud.LoadBalancerUpdateServiceOpts{
			HTTP: &hcloud.LoadBalancerUpdateServiceOptsHTTP{
				Certificates: service.HTTP.Certificates,
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}
