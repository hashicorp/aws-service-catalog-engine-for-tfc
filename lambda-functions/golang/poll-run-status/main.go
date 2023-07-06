package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/secretsmanager"
	"log"
)

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	// Initialize the TFE client
	sdkConfig := awsconfig.GetSdkConfig(initContext)

	// Create secrets client SDK to fetch TFE credentials
	secretsManager, err := secretsmanager.NewWithConfig(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize secrets manager client: %s", err)
	}

	// Create the handler
	handler := &PollRunStatusHandler{secretsManager: secretsManager}

	// Start the lambda using the handler
	lambda.Start(handler.HandleRequest)
}
