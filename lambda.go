package main

import (
	"certmaster/models"
	"fmt"
	"os"
)

func HandleLambdaEvent(event *models.CertRequest) (*models.CertResponse, error) {
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}

	// Delete this token and use whatever token is passed in
	os.Unsetenv("AWS_SESSION_TOKEN")

	err := createCert(event)

	status := "Success"
	if err != nil {
		status = "Failed"
	}

	return &models.CertResponse{Status: status, Message: err.Error()}, err
}
