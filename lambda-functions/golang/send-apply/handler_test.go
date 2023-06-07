package main

import (
	"testing"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/s3"
	"os"
	"io"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
	"archive/tar"
	"github.com/stretchr/testify/assert"
	"encoding/json"
)

func TestSendApplyHandler_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()
	defer tfcServer.Stop()

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Error(err)
	}

	// Create mock S3 downlaoder
	const MockArtifactPath = "../../../example-product/product.tar.gz"
	mockDownloader := s3.MockDownloader{
		MockArtifactPath: MockArtifactPath,
	}

	// Create a test instance of the Lambda function
	testHandler := &SendApplyHandler{
		tfeClient:    tfeClient,
		s3Downloader: mockDownloader,
		region:       "narnia-west-2",
	}

	// Create test request
	testRequest := SendApplyRequest{
		AwsAccountId:          "123456789042",
		TerraformOrganization: tfcServer.OrganizationName,
		ProvisionedProductId:  "amazingly-great-product-instance",
		Artifact: Artifact{
			Path: "s3://wowzers-this-is-some/fake/artifact/path",
			Type: "beeg-test",
		},
		LaunchRoleArn: "arn:::some/fake/role/arn",
		ProductId:     "id-4-number-1-best-product",
		Tags:          make([]AWSTag, 0),
		TracerTag: tracertag.TracerTag{
			TracerTagKey:   "test-tracer-tag-key",
			TracerTagValue: "test-trace-tag-value",
		},
	}

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Check uploaded artifact contains overrides
	entries := GetArtifactEntryNames(t, tfcServer.UploadedArtifact())

	checkedProviderOverrides := false
	for _, entry := range entries {
		if entry.FileName == "provider_override.tf.json" {
			checkedProviderOverrides = true

			providerOverride := &ProviderOverride{}
			err := json.Unmarshal([]byte(entry.FileContents), providerOverride)
			if err != nil {
				t.Error(err)
			}

			// Verify region was set
			assert.Equal(t, "narnia-west-2", providerOverride.Provider.AWS.Region)

			// Verify billing tags were set
			tags := providerOverride.Provider.AWS.DefaultTags.Tags
			tracerTag := tags["test-tracer-tag-key"]
			if tracerTag == "" {
				t.Error("tracer tag was missing")
			}
			assert.Equal(t, "test-trace-tag-value", tracerTag)
		}
	}

	assert.True(t, checkedProviderOverrides, "provider_override.tf.json file should be present in the uploaded artifact")
}

type UploadedArtifactEntry struct {
	FileName     string
	FileContents string
}

func GetArtifactEntryNames(t *testing.T, uploadedArtifact []byte) []UploadedArtifactEntry {
	// Write uploaded artifact to file
	tmp, err := os.CreateTemp("", "uploaded_artifact")
	if err != nil {
		t.Error(err)
	}
	_, err = tmp.Write(uploadedArtifact)
	if err != nil {
		t.Error(err)
	}

	_, err = tmp.Seek(0, io.SeekStart)
	if err != nil {
		t.Error(err)
	}

	// unzip the file
	unzippedArchive, err := fileutils.UnzipFile(tmp)
	if err != nil {
		t.Error(err)
	}

	// Check the entries
	tr := tar.NewReader(unzippedArchive)

	entryNames := make([]UploadedArtifactEntry, 0)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Error(err)
		}

		entryContents, err := io.ReadAll(tr)
		if err != nil {
			t.Error(err)
		}

		entryNames = append(entryNames, UploadedArtifactEntry{
			FileName:     header.Name,
			FileContents: string(entryContents),
		})

	}

	return entryNames
}

type ProviderOverride struct {
	Provider struct {
		AWS struct {
			DefaultTags struct {
				Tags map[string]string `json:"tags"`
			} `json:"default_tags"`
			Region string `json:"region"`
		} `json:"aws"`
	} `json:"provider"`
}
