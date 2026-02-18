/*
 * Copyright IBM Corp. 2023, 2025
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"reflect"
	"testing"

	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/s3"
)

const TestArtifactPath = "s3://terraform-configurations-cross-account-demo/product_with_override_var.tar.gz"
const TestArtifactType = "AWS_S3"
const TestLaunchRoleArn = "arn:aws:iam::829064435212:role/SCLaunchRole"
const TestS3BucketArtifactPath = "../../../example-product/product.tar.gz"
const TestS3BucketArtifactFileName = "main.tf"
const TestS3BucketArtifactFileContent = "# Copyright (c) HashiCorp, Inc.\n# SPDX-License-Identifier: MPL-2.0\n\nterraform {\n  required_providers {\n    aws = {\n      source  = \"hashicorp/aws\"\n      version = \"4.63.0\"\n    }\n\n    random = {\n      source  = \"hashicorp/random\"\n      version = \"3.5.1\"\n    }\n  }\n}\n\nprovider \"aws\" {}\n\nresource \"random_string\" \"random\" {\n  length  = var.random_string_length\n  special = false\n  upper   = false\n}\n\nresource \"aws_s3_bucket\" \"my-bucket\" {\n  bucket = \"aws-tfc-service-catalog-example-${random_string.random.result}\"\n}\n\nvariable \"random_string_length\" {\n  type        = number\n  description = \"Length of the random string to append to the bucket name\"\n  default     = 16\n}\n\noutput \"bucket_name\" {\n  value = aws_s3_bucket.my-bucket.bucket\n}"

func TestConfigFetcherFetchHappy(t *testing.T) {
	// setup
	// Create mock S3 downloader
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: TestS3BucketArtifactPath,
	}

	testHandler := &TerraformParameterParserHandler{s3Downloader: mockDownloader}

	input := TerraformParameterParserInput{
		Artifact: Artifact{
			Path: TestArtifactPath,
			Type: TestArtifactType,
		},
		LaunchRoleArn: TestLaunchRoleArn,
	}

	// act
	fileMap, err := testHandler.fetchArtifact(context.Background(), input)

	// assert
	if err != nil {
		t.Errorf("Unexpected error occured. cause: %v", err.Error())
	}

	fileContent, ok := fileMap[TestS3BucketArtifactFileName]
	if !ok {
		t.Errorf("Expected file %s was not parsed", TestS3BucketArtifactFileName)
	}

	if !reflect.DeepEqual(fileContent, TestS3BucketArtifactFileContent) {
		t.Errorf("File content for %s is not as expected.\nExpected:\n%q\nGot:\n%q", TestS3BucketArtifactFileName, TestS3BucketArtifactFileContent, fileContent)
	}
}

func TestConfigFetcherFetchWithEmptyLaunchRoleHappy(t *testing.T) {
	// setup
	// Create mock S3 downloader
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: TestS3BucketArtifactPath,
	}

	testHandler := &TerraformParameterParserHandler{s3Downloader: mockDownloader}

	input := TerraformParameterParserInput{
		Artifact: Artifact{
			Path: TestArtifactPath,
			Type: TestArtifactType,
		},
		LaunchRoleArn: "",
	}

	// act
	fileMap, err := testHandler.fetchArtifact(context.Background(), input)

	// assert
	if err != nil {
		t.Errorf("Unexpected error occured. cause: %v", err.Error())
	}

	fileContent, ok := fileMap[TestS3BucketArtifactFileName]
	if !ok {
		t.Errorf("Expected file %s was not parsed", TestS3BucketArtifactFileName)
	}

	if !reflect.DeepEqual(fileContent, TestS3BucketArtifactFileContent) {
		t.Errorf("File content for %s is not as expected.\nExpected:\n%q\nGot:\n%q", TestS3BucketArtifactFileName, TestS3BucketArtifactFileContent, fileContent)
	}
}
