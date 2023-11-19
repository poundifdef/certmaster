package destinations

import (
	"errors"
	"reflect"

	"github.com/poundifdef/certmaster/destinations/email"
	"github.com/poundifdef/certmaster/destinations/hetzner"
	"github.com/poundifdef/certmaster/destinations/sftp"
	"github.com/poundifdef/certmaster/models"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/mitchellh/mapstructure"
)

// Reflection magic: https://stackoverflow.com/a/23031445/3788

var typeRegistry = map[string]reflect.Type{
	"email":   reflect.TypeOf(email.Destination{}),
	"hetzner": reflect.TypeOf(hetzner.Destination{}),
	"sftp":    reflect.TypeOf(sftp.Destination{}),
}

func ListDestinations() []models.DestinationDescription {
	rc := make([]models.DestinationDescription, len(typeRegistry))

	i := 0
	for k := range typeRegistry {
		destination := GetDestination(k)
		rc[i] = models.DestinationDescription{
			Name:        k,
			Description: destination.Description(),
		}
		i += 1
	}

	return rc
}

func GetDestination(dest string) Destination {
	destinationType, ok := typeRegistry[dest]
	if !ok {
		return nil
	}
	v := reflect.New(destinationType).Elem()
	return v.Interface().(Destination)
}

func ListConfigFields(providerName string) []models.DestinationConfig {
	rc := []models.DestinationConfig{}

	provider := GetDestination(providerName)
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

	var d Destination = GetDestination(providerName)
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
	Description() string
	Upload(request models.CertRequest, cert *certificate.Resource) error
}
