package main

import (
	"certmaster/models"
	"fmt"
)

func HandleLambdaEvent(event *models.CertRequest) (*models.CertResponse, error) {
	if event == nil {
		return nil, fmt.Errorf("received nil event")
	}

	err := createCert(event)

	status := "Success"
	if err != nil {
		status = "Failed"
	}

	return &models.CertResponse{Status: status, Message: err.Error()}, err
}
