/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	sc "github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/servicecatalog"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tracertag"
	"log"
)

type NotifyRunResultRequest struct {
	TerraformRunId          string                  `json:"terraformRunId"`
	WorkflowToken           string                  `json:"workflowToken"`
	RecordId                string                  `json:"recordId"`
	TracerTag               tracertag.TracerTag     `json:"tracerTag"`
	ServiceCatalogOperation ServiceCatalogOperation `json:"serviceCatalogOperation"`
	AwsAccountId            string                  `json:"awsAccountId"`
	TerraformOrganization   string                  `json:"terraformOrganization"`
	ProvisionedProductId    string                  `json:"provisionedProductId"`
	Error                   string                  `json:"error"`
	ErrorMessage            string                  `json:"errorMessage"`
}

type ServiceCatalogOperation string

// Enum values for ServiceCatalogOperation
const (
	Terminating  ServiceCatalogOperation = "TERMINATING"
	Provisioning ServiceCatalogOperation = "PROVISIONING"
	Updating     ServiceCatalogOperation = "UPDATING"
)

type NotifyRunResultResponse struct{}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	sdkConfig := awsconfig.GetSdkConfig(initContext)
	serviceCatalogClient := sc.NewFromConfig(sdkConfig)
	serviceCatalog := servicecatalog.SC{
		Client: serviceCatalogClient,
	}

	// Create secrets client SDK to fetch TFE credentials
	secretsManager, err := secretsmanager.NewWithConfig(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize secrets manager client: %s", err)
	}

	handler := NotifyRunResultHandler{
		serviceCatalog: serviceCatalog,
		secretsManager: secretsManager,
	}

	lambda.Start(handler.HandleRequest)
}
