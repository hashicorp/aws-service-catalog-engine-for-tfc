package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"log"
)

type SendApplyRequest struct {
	AwsAccountId          string              `json:"awsAccountId"`
	TerraformOrganization string              `json:"terraformOrganization"`
	ProvisionedProductId  string              `json:"provisionedProductId"`
	Artifact              Artifact            `json:"artifact"`
	LaunchRoleArn         string              `json:"launchRoleArn"`
	ProductId             string              `json:"productId"`
	Tags                  []AWSTag            `json:"tags"`
	TracerTag             tracertag.TracerTag `json:"tracerTag"`
}

type AWSTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Artifact struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type SendApplyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	sdkConfig := awsconfig.GetSdkConfig(initContext)

	// Create secrets client SDK to fetch TFE credentials
	secretsManager, err := secretsmanager.NewWithConfig(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize secrets manager client: %s", err)
	}

	// Initialize the s3 downloader
	s3Downloader := fileutils.NewS3DownloaderWithAssumedRole(initContext, sdkConfig)

	// Create the handler
	handler := &SendApplyHandler{
		s3Downloader:   s3Downloader,
		secretsManager: secretsManager,
		region:         sdkConfig.Region,
	}

	// Start the lambda using the handler
	lambda.Start(handler.HandleRequest)
}
