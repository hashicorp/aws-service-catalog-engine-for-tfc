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
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tracertag"
	"log"
	"os"
)

type SendApplyRequest struct {
	AwsAccountId          string              `json:"awsAccountId"`
	TerraformOrganization string              `json:"terraformOrganization"`
	ProvisionedProductId  string              `json:"provisionedProductId"`
	ProvisionedArtifactId string              `json:"provisioningArtifactId"`
	Artifact              Artifact            `json:"artifact"`
	LaunchRoleArn         string              `json:"launchRoleArn"`
	ProductId             string              `json:"productId"`
	Parameters            []Parameter         `json:"parameters"`
	Tags                  []AWSTag            `json:"tags"`
	TracerTag             tracertag.TracerTag `json:"tracerTag"`
}

type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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

	// Get Terraform Version
	terraformVersion := os.Getenv("TERRAFORM_VERSION")

	// Create the handler
	handler := &SendApplyHandler{
		s3Downloader:     s3Downloader,
		secretsManager:   secretsManager,
		region:           sdkConfig.Region,
		terraformVersion: terraformVersion,
	}

	// Start the lambda using the handler
	lambda.Start(handler.HandleRequest)
}
