/*
 * Copyright (c) HashiCorp, Inc.
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
const TestS3BucketArtifactFileContent = "\"bucket_name\" {\n  type = string\n}\nprovider \"aws\" {\n}\nresource \"aws_s3_bucket\" \"bucket\" {\n  bucket = var.bucket_name\n}\noutput regional_domain_name {\n  value = aws_s3_bucket.bucket.bucket_regional_domain_name\n}"

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

	if reflect.DeepEqual(fileContent, TestS3BucketArtifactFileContent) {
		t.Errorf("File content for %s is not as expected", TestS3BucketArtifactFileName)
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

	if reflect.DeepEqual(fileContent, TestS3BucketArtifactFileContent) {
		t.Errorf("File content for %s is not as expected", TestS3BucketArtifactFileName)
	}
}
