package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/identifiers"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/s3"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/testtfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestSendApplyHandler_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock S3 downloader
	const MockArtifactPath = "../../../example-product/product.tar.gz"
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: MockArtifactPath,
	}

	// Create a test instance of the Lambda function
	testHandler := &SendApplyHandler{
		secretsManager: mockSecretsManager,
		s3Downloader:   mockDownloader,
		region:         "narnia-west-2",
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
	_, err := testHandler.HandleRequest(context.Background(), testRequest)
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

	// Check to make sure correct launch role arn was assumed to download s3 files
	assert.Equal(t, testRequest.LaunchRoleArn, mockDownloader.AssumedRole, "correct launch role arn should have been assumed to download s3 files")
}

func TestSendApplyHandler_Success_UpdatingExistingWorkspace(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	tfcServer.AddProject("id-4-number-1-best-product", testtfc.ProjectFactoryParameters{
		Name: "id-4-number-1-best-product",
	})

	workspaceName := identifiers.GetWorkspaceName("123456789042", "amazingly-great-product-instance")
	testWorkspace := tfcServer.AddWorkspace("ws-4329432942", testtfc.WorkspaceFactoryParameters{
		Name: workspaceName,
	})

	// Add a large amount of variables to the workspace to force the handler to have to paginate through them to find
	// the ones it needs to update
	numberOfVarsToCreate := 250
	for varNumber := 0; varNumber < numberOfVarsToCreate; varNumber++ {
		tfcServer.AddVar(&tfe.Variable{
			Key:       fmt.Sprintf("VAR_%d", varNumber),
			Value:     "yo",
			Category:  tfe.CategoryEnv,
			HCL:       false,
			Sensitive: false,
			Workspace: testWorkspace,
		})
	}
	// add the actual variables the handler needs to update
	providerAuthVar := tfcServer.AddVar(&tfe.Variable{
		Key:       "TFC_AWS_PROVIDER_AUTH",
		Value:     "false",
		Category:  tfe.CategoryEnv,
		HCL:       false,
		Sensitive: false,
		Workspace: testWorkspace,
	})
	runRoleArnVar := tfcServer.AddVar(&tfe.Variable{
		Key:       "TFC_AWS_RUN_ROLE_ARN",
		Category:  tfe.CategoryEnv,
		HCL:       false,
		Sensitive: false,
		Workspace: testWorkspace,
	})

	// Create mock S3 downloader
	const MockArtifactPath = "../../../example-product/product.tar.gz"
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: MockArtifactPath,
	}

	// Create a test instance of the Lambda function
	testHandler := &SendApplyHandler{
		secretsManager: mockSecretsManager,
		s3Downloader:   mockDownloader,
		region:         "narnia-west-2",
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
	_, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Check Variables were updated
	assert.Equal(t, "true", providerAuthVar.Value)
	assert.Equal(t, "arn:::some/fake/role/arn", runRoleArnVar.Value)
}

func TestSendApplyHandler_Success_ProjectAlreadyExists(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	tfcServer.AddProject("id-4-number-1-best-product", testtfc.ProjectFactoryParameters{
		Name: "id-4-number-1-best-product",
	})

	// Create mock S3 downloader
	const MockArtifactPath = "../../../example-product/product.tar.gz"
	mockDownloader := &s3.MockDownloader{
		MockArtifactPath: MockArtifactPath,
	}

	// Create a test instance of the Lambda function
	testHandler := &SendApplyHandler{
		secretsManager: mockSecretsManager,
		s3Downloader:   mockDownloader,
		region:         "narnia-west-2",
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
	_, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}
}

func TestSendApplyHandler_ErrorFetchingArtifactFromS3(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	tfcServer.AddProject("id-4-number-1-best-product", testtfc.ProjectFactoryParameters{
		Name: "id-4-number-1-best-product",
	})

	// Create mock S3 downloader
	mockDownloader := s3.MockErrorDownloader{}

	// Create a test instance of the Lambda function
	testHandler := &SendApplyHandler{
		secretsManager: mockSecretsManager,
		s3Downloader:   mockDownloader,
		region:         "narnia-west-2",
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
	_, err := testHandler.HandleRequest(context.Background(), testRequest)

	// Verify an errors was returned
	assert.Error(t, err, "Verify handler failed")
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
