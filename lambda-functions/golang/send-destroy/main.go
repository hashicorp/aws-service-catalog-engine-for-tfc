package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"context"
)

type SendDestroyRequest struct {
	AwsAccountId          string `json:"awsAccountId"`
	TerraformOrganization string `json:"terraformOrganization"`
	ProvisionedProductId  string `json:"provisionedProductId"`
}

type SendDestroyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	sdkConfig := awsconfig.GetSdkConfig(initContext)

	client, err := tfc.GetTFEClient(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	handler := SendDestroyHandler{
		tfeClient: client,
	}

	lambda.Start(handler.HandleRequest)
}

// Get the workspace name, which is `${accountId} - ${provisionedProductId}`
func getWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
