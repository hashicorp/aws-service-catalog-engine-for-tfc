package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"context"
)

type TerraformOpenSourceParameterParserInput struct {
	Artifact      Artifact `json:"artifact"`
	LaunchRoleArn string   `json:"launchRoleArn"`
}

type TerraformOpenSourceParameterParserResponse struct {
	Parameters []*Parameter `json:"parameters"`
}

func main() {
	// Create temporary context to initialize the handler with
	initContext := context.TODO()

	// Initialize the TFE client
	sdkConfig := awsconfig.GetSdkConfig(initContext)

	// Initialize the s3 downloader
	s3Downloader := fileutils.NewS3DownloaderWithAssumedRole(initContext, sdkConfig)

	h := &TerraformParameterParserHandler{
		s3Downloader: s3Downloader,
	}

	lambda.Start(h.HandleRequest)
}

func (h *TerraformParameterParserHandler) HandleRequest(ctx context.Context, event TerraformOpenSourceParameterParserInput) (TerraformOpenSourceParameterParserResponse, error) {
	if err := ValidateInput(event); err != nil {
		return TerraformOpenSourceParameterParserResponse{}, err
	}

	fileMap, fileMapErr := h.fetchArtifact(ctx, event)
	if fileMapErr != nil {
		return TerraformOpenSourceParameterParserResponse{}, fileMapErr
	}

	parameters, parseParametersErr := ParseParametersFromConfiguration(fileMap)
	return TerraformOpenSourceParameterParserResponse{Parameters: parameters}, parseParametersErr
}
