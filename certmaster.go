package main

import (
	"crypto"
	"errors"
	"log"
	"log/slog"
	"os"

	"github.com/poundifdef/certmaster/destinations"
	"github.com/poundifdef/certmaster/models"

	"github.com/go-acme/lego/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
)

// ACME User type
type User struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func newUser(email string) *User {
	// Generate a private key for the user
	privateKey, err := certcrypto.GeneratePrivateKey(certcrypto.RSA2048)
	if err != nil {
		log.Fatal(err)
	}

	return &User{
		Email: email,
		key:   privateKey,
	}
}

func setEnvVars(vars map[string]string) error {
	for k, v := range vars {
		err := os.Setenv(k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func uploadCertToDestination(request models.CertRequest, cert *certificate.Resource, destination map[string]any) error {
	return destinations.Upload(request, cert, destination)
}

func requestCertificates(domain string, email string, dnsProvider string, stage bool) (*certificate.Resource, error) {
	// Create a user for ACME
	user := newUser(email)

	// Create a new ACME client
	config := lego.NewConfig(user)
	if stage == true {
		config.CADirURL = lego.LEDirectoryStaging
	}

	client, err := lego.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Set up DNS provider (example: Cloudflare)
	provider, err := dns.NewDNSChallengeProviderByName(dnsProvider)
	if err != nil {
		log.Fatal(err)
	}

	// Use the DNS challenge
	client.Challenge.SetDNS01Provider(provider)

	// Register the user
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		log.Fatal(err)
	}
	user.Registration = reg

	// Request a certificate
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	return certificates, err
}

func createCert(request *models.CertRequest) error {
	log.Println("Creating certificate for", request.Domain)

	err := setEnvVars(request.DNSCredentials)
	if err != nil {
		return err
	}

	var cert *certificate.Resource

	if request.UseDummyCert {
		cert = GetDummyCert(request.Domain)
	} else {
		cert, err = requestCertificates(request.Domain, request.RequesterEmail, request.DNSCredentials["provider"], request.StageEnvironment)
		if err != nil {
			return err
		}
	}

	uploadErrors := false
	for i, destination := range request.Destinations {
		slog.Info("Uploading certificate", "destination", destination["provider"], "index", i)
		err := uploadCertToDestination(*request, cert, destination)
		if err != nil {
			slog.Info("Error uploading certificate", "destination", destination["provider"], "index", i, "error", err.Error())
			uploadErrors = true
		} else {
			slog.Info("Completed uploading certificate", "destination", destination["provider"], "index", i)

		}
	}

	if uploadErrors {
		return errors.New("Error uploading some certificates")
	}

	return nil
}
