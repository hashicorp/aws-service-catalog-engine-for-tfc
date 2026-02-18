/*
 * Copyright IBM Corp. 2023, 2025
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/s3"
)

const TestArtifactPath = "s3://terraform-configurations-cross-account-demo/product_with_override_var.tar.gz"
const TestArtifactType = "AWS_S3"
const TestLaunchRoleArn = "arn:aws:iam::829064435212:role/SCLaunchRole"
const TestS3BucketArtifactFileName = "main.tf"

// createTestTarGz creates a temporary tar.gz file with random content for testing.
// Returns the path to the tar.gz file and the random content that was written to it.
func createTestTarGz(t *testing.T, filename string) (tarGzPath string, content string) {
	t.Helper()

	// Generate random content for the test file
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		t.Fatalf("Failed to generate random content: %v", err)
	}
	randomContent := "# Test content: " + hex.EncodeToString(randomBytes)

	// Create a temporary tar.gz file
	tmpDir := t.TempDir()
	tarGzPath = filepath.Join(tmpDir, "test-product.tar.gz")

	// Create the tar.gz file with the random content
	tarGzFile, err := os.Create(tarGzPath)
	if err != nil {
		t.Fatalf("Failed to create temporary tar.gz file: %v", err)
	}
	defer tarGzFile.Close()

	gzWriter := gzip.NewWriter(tarGzFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add the test file to the tar archive
	header := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(randomContent)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tarWriter.Write([]byte(randomContent)); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	// Close writers to flush content
	tarWriter.Close()
	gzWriter.Close()
	tarGzFile.Close()

	return tarGzPath, randomContent
}

func TestConfigFetcherFetchHappy(t *testing.T) {
	// setup
	tarGzPath, randomContent := createTestTarGz(t, TestS3BucketArtifactFileName)

	// Create mock S3 downloader with the temporary tar.gz file
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: tarGzPath,
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

	if !reflect.DeepEqual(fileContent, randomContent) {
		t.Errorf("File content for %s is not as expected.\nExpected:\n%q\nGot:\n%q", TestS3BucketArtifactFileName, randomContent, fileContent)
	}
}

func TestConfigFetcherFetchWithEmptyLaunchRoleHappy(t *testing.T) {
	// setup
	tarGzPath, randomContent := createTestTarGz(t, TestS3BucketArtifactFileName)

	// Create mock S3 downloader with the temporary tar.gz file
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: tarGzPath,
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

	if !reflect.DeepEqual(fileContent, randomContent) {
		t.Errorf("File content for %s is not as expected.\nExpected:\n%q\nGot:\n%q", TestS3BucketArtifactFileName, randomContent, fileContent)
	}
}
