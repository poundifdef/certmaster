package destinations

import (
	"certmaster/destinations/email"
	"certmaster/destinations/hetzner"
	"certmaster/destinations/sftp"
	"certmaster/models"
	"errors"
	"reflect"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/mitchellh/mapstructure"
)

func getDestination(dest string) Destination {
	switch dest {
	case "email":
		return email.Destination{}
	case "hetzner":
		return hetzner.Destination{}
	case "sftp":
		return sftp.Destination{}
	default:
		return nil
	}
}

func ListConfigFields(providerName string) []models.DestinationConfig {
	rc := []models.DestinationConfig{}

	provider := getDestination(providerName)
	if provider != nil {
		st := reflect.TypeOf(provider)
		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			destConfig := models.DestinationConfig{
				Field:       field.Tag.Get("mapstructure"),
				Description: field.Tag.Get("description"),
			}
			if field.Tag.Get("credential") == "true" {
				destConfig.IsCredential = true
			}

			rc = append(rc, destConfig)
		}
	}

	return rc
}

func Upload(request models.CertRequest, cert *certificate.Resource, config map[string]string) error {
	providerName := config["provider"]

	var d Destination = getDestination(providerName)
	if d == nil {
		return errors.New("Destination does not exist")
	}

	err := mapstructure.Decode(config, &d)
	if err != nil {
		return err
	}

	return d.Upload(request, cert)
}

type Destination interface {
	Upload(request models.CertRequest, cert *certificate.Resource) error
}
