/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/fileutils"
)

type TerraformParameterParserInput struct {
	Artifact      Artifact `json:"artifact"`
	LaunchRoleArn string   `json:"launchRoleArn"`
}

type TerraformParameterParserResponse struct {
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

func (h *TerraformParameterParserHandler) HandleRequest(ctx context.Context, event TerraformParameterParserInput) (TerraformParameterParserResponse, error) {
	if err := ValidateInput(event); err != nil {
		return TerraformParameterParserResponse{}, err
	}

	fileMap, fileMapErr := h.fetchArtifact(ctx, event)
	if fileMapErr != nil {
		return TerraformParameterParserResponse{}, fileMapErr
	}

	parameters, parseParametersErr := ParseParametersFromConfiguration(fileMap)
	return TerraformParameterParserResponse{Parameters: parameters}, parseParametersErr
}
