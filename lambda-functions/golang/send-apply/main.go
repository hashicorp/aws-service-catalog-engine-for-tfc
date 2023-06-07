package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"log"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
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

	// Initialize the TFE client
	sdkConfig := awsconfig.GetSdkConfig(initContext)
	client, err := tfc.GetTFEClient(initContext, sdkConfig)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	// Initialize the s3 downloader
	s3Client := s3.NewFromConfig(sdkConfig)
	s3Downloader := fileutils.S3ManagerDownloader{
		S3Client: s3Client,
	}

	// Create the handler
	handler := &SendApplyHandler{tfeClient: client, s3Downloader: s3Downloader, region: sdkConfig.Region}

	// Start the lambda using the handler
	lambda.Start(handler.HandleRequest)
}
