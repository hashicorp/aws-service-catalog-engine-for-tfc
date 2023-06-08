package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	sc "github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"log"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/servicecatalog"
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

	tfeClient, err := tfc.GetTFEClient(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	handler := NotifyRunResultHandler{
		serviceCatalog: serviceCatalog,
		tfeClient:      tfeClient,
	}

	lambda.Start(handler.HandleRequest)
}
