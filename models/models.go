package models

type CertRequest struct {
	RequesterEmail   string            `json:"email"`
	Domain           string            `json:"domain"`
	DNSCredentials   map[string]string `json:"dns"`
	Destinations     []map[string]any  `json:"destinations"`
	UseDummyCert     bool              `json:"dummy"`
	StageEnvironment bool              `json:"stage"`
}

type CertResponse struct {
	Status string `json:"status"`
}

type DestinationDescription struct {
	Name        string
	Description string
}

type DestinationConfig struct {
	Field        string
	Description  string
	IsCredential bool
}
