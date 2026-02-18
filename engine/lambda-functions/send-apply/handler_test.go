/*
 * Copyright IBM Corp. 2023, 2025
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/identifiers"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/s3"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/testtfc"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tracertag"
	"github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
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
		t.Fatal(err)
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

			// Region SHOULD be set since default provider exists without region
			assert.Equal(t, "narnia-west-2", providerOverride.Provider.AWS.Region, "region should be overridden when default provider exists without region")

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

	// Check to make sure correct launch role ARN was assumed to download S3 files
	assert.Equal(t, testRequest.LaunchRoleArn, mockDownloader.AssumedRole, "correct launch role arn should have been assumed to download s3 files")

	// The workspace was persisted
	assert.Equal(t, 1, len(tfcServer.Workspaces), "correct number of workspaces was persisted")
	keys := reflect.ValueOf(tfcServer.Workspaces).MapKeys()
	workspaceId := keys[0].String()

	// Check that the metadata headers were set on the workspace in TFC
	serviceCatalogMetadata := tfcServer.WorkspaceServiceCatalogMetadata[workspaceId]
	assert.Equal(t, testRequest.ProductId, serviceCatalogMetadata.ProductId)
	assert.Equal(t, testRequest.ProvisionedProductId, serviceCatalogMetadata.ProvisionedProductId)
	assert.Equal(t, testRequest.ProvisionedArtifactId, serviceCatalogMetadata.ProductVersion)
}

func TestSendApplyHandler_Success_WithAliasedProviders(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock S3 downloader with artifact containing aliased providers
	const MockArtifactPath = "test-artifacts/artifact-with-aliased-and-nonaliased-providers.tar.gz"
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
		t.Fatal(err)
	}

	// Check uploaded artifact contains overrides
	entries := GetArtifactEntryNames(t, tfcServer.UploadedArtifact())

	checkedProviderOverrides := false
	for _, entry := range entries {
		if entry.FileName == "provider_override.tf.json" {
			checkedProviderOverrides = true

			// Parse the override JSON
			var overrideData map[string]interface{}
			err := json.Unmarshal([]byte(entry.FileContents), &overrideData)
			if err != nil {
				t.Error(err)
			}

			// Verify the provider section exists
			providerSection, ok := overrideData["provider"].(map[string]interface{})
			if !ok {
				t.Error("provider section not found in override file")
			}

			// Verify ONLY default AWS provider override exists (no aliased provider overrides)
			assert.Equal(t, 1, len(providerSection), "should only have one provider override (default aws)")
			
			defaultAws, ok := providerSection["aws"].(map[string]interface{})
			if !ok {
				t.Error("default aws provider override not found")
			}
			
			// Region should NOT be overridden since default provider has region defined (us-east-1)
			assert.Nil(t, defaultAws["region"], "region should not be overridden when default provider has region set")

			// Verify aliased provider overrides do NOT exist (they should be excluded)
			_, hasWest := providerSection["aws.west"]
			assert.False(t, hasWest, "aws.west provider override should not exist")
			
			_, hasEu := providerSection["aws.eu"]
			assert.False(t, hasEu, "aws.eu provider override should not exist")

			// Verify default provider has the tracer tag
			defaultTags, ok := defaultAws["default_tags"].(map[string]interface{})
			if !ok {
				t.Error("default_tags not found for default provider")
			}
			tags, ok := defaultTags["tags"].(map[string]interface{})
			if !ok {
				t.Error("tags not found for default provider")
			}
			tracerTag, ok := tags["test-tracer-tag-key"]
			if !ok {
				t.Error("tracer tag not found for default provider")
			}
			assert.Equal(t, "test-trace-tag-value", tracerTag)
		}
	}

	assert.True(t, checkedProviderOverrides, "provider_override.tf.json file should be present in the uploaded artifact")
}

func TestSendApplyHandler_Success_WithVariableRegion(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}
	// Create mock S3 downloader with artifact containing a provider with region to a variable
	const MockArtifactPath = "test-artifacts/artifact-with-variable-region.tar.gz"
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
		t.Fatal(err)
	}

	// Check uploaded artifact contains overrides for all providers
	entries := GetArtifactEntryNames(t, tfcServer.UploadedArtifact())

	checkedProviderOverrides := false
	for _, entry := range entries {
		if entry.FileName == "provider_override.tf.json" {
			checkedProviderOverrides = true

			// Parse the override JSON
			var overrideData map[string]interface{}
			err := json.Unmarshal([]byte(entry.FileContents), &overrideData)
			if err != nil {
				t.Error(err)
			}

			// Verify the provider section exists
			providerSection, ok := overrideData["provider"].(map[string]interface{})
			if !ok {
				t.Error("provider section not found in override file")
			}

			// Verify default AWS provider override exists
			defaultAws, ok := providerSection["aws"].(map[string]interface{})
			if !ok {
				t.Error("default aws provider override not found")
			}
			// Region should NOT be overridden since it's defined in the config (us-east-1)
			assert.Nil(t, defaultAws["region"])

			// Verify all providers have the tracer tag
			for providerKey, providerConfig := range providerSection {
				config := providerConfig.(map[string]interface{})
				defaultTags, ok := config["default_tags"].(map[string]interface{})
				if !ok {
					t.Errorf("default_tags not found for provider %s", providerKey)
					continue
				}
				tags, ok := defaultTags["tags"].(map[string]interface{})
				if !ok {
					t.Errorf("tags not found for provider %s", providerKey)
					continue
				}
				tracerTag, ok := tags["test-tracer-tag-key"]
				if !ok {
					t.Errorf("tracer tag not found for provider %s", providerKey)
					continue
				}
				assert.Equal(t, "test-trace-tag-value", tracerTag)
			}
		}
	}

	assert.True(t, checkedProviderOverrides, "provider_override.tf.json file should be present in the uploaded artifact")
}

func TestSendApplyHandler_Success_WithNoProvider(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock S3 downloader with artifact containing no providers
	const MockArtifactPath = "test-artifacts/artifact-with-no-provider.tar.gz"
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
		t.Fatal(err)
	}

	// Check uploaded artifact contains overrides for all providers
	entries := GetArtifactEntryNames(t, tfcServer.UploadedArtifact())

	checkedProviderOverrides := false
	for _, entry := range entries {
		if entry.FileName == "provider_override.tf.json" {
			checkedProviderOverrides = true

			// Parse the override JSON
			var overrideData map[string]interface{}
			err := json.Unmarshal([]byte(entry.FileContents), &overrideData)
			if err != nil {
				t.Error(err)
			}

			// Verify the provider section exists
			providerSection, ok := overrideData["provider"].(map[string]interface{})
			if !ok {
				t.Error("provider section not found in override file")
			}

			// Verify default AWS provider override exists
			defaultAws, ok := providerSection["aws"].(map[string]interface{})
			if !ok {
				t.Error("default aws provider override not found")
			}
			// Region should be overridden since it's defined in the config (us-east-1)
			assert.Equal(t, "narnia-west-2", defaultAws["region"])

			// Verify all providers have the tracer tag
			for providerKey, providerConfig := range providerSection {
				config := providerConfig.(map[string]interface{})
				defaultTags, ok := config["default_tags"].(map[string]interface{})
				if !ok {
					t.Errorf("default_tags not found for provider %s", providerKey)
					continue
				}
				tags, ok := defaultTags["tags"].(map[string]interface{})
				if !ok {
					t.Errorf("tags not found for provider %s", providerKey)
					continue
				}
				tracerTag, ok := tags["test-tracer-tag-key"]
				if !ok {
					t.Errorf("tracer tag not found for provider %s", providerKey)
					continue
				}
				assert.Equal(t, "test-trace-tag-value", tracerTag)
			}
		}
	}

	assert.True(t, checkedProviderOverrides, "provider_override.tf.json file should be present in the uploaded artifact")
}

func TestSendApplyHandler_Success_WithOnlyAliasedProviderNoRegion(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock S3 downloader with artifact containing only an aliased provider without region
	const MockArtifactPath = "test-artifacts/artifact-with-only-aliased-provider-no-region.tar.gz"
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
		t.Fatal(err)
	}

	// Check uploaded artifact contains overrides
	entries := GetArtifactEntryNames(t, tfcServer.UploadedArtifact())

	checkedProviderOverrides := false
	for _, entry := range entries {
		if entry.FileName == "provider_override.tf.json" {
			checkedProviderOverrides = true

			// Parse the override JSON
			var overrideData map[string]interface{}
			err := json.Unmarshal([]byte(entry.FileContents), &overrideData)
			if err != nil {
				t.Error(err)
			}

			// Verify the provider section exists
			providerSection, ok := overrideData["provider"].(map[string]interface{})
			if !ok {
				t.Error("provider section not found in override file")
			}

			// Verify ONLY default AWS provider override exists
			assert.Equal(t, 1, len(providerSection), "should only have one provider override (default aws)")
			
			defaultAws, ok := providerSection["aws"].(map[string]interface{})
			if !ok {
				t.Error("default aws provider override not found")
			}
			
			// Region SHOULD be overridden since only aliased provider exists without region
			assert.Equal(t, "narnia-west-2", defaultAws["region"], "region should be overridden when only aliased provider without region exists")

			// Verify aliased provider override does NOT exist
			_, hasWest := providerSection["aws.west"]
			assert.False(t, hasWest, "aws.west provider override should not exist")

			// Verify default provider has the tracer tag
			defaultTags, ok := defaultAws["default_tags"].(map[string]interface{})
			if !ok {
				t.Error("default_tags not found for default provider")
			}
			tags, ok := defaultTags["tags"].(map[string]interface{})
			if !ok {
				t.Error("tags not found for default provider")
			}
			tracerTag, ok := tags["test-tracer-tag-key"]
			if !ok {
				t.Error("tracer tag not found for default provider")
			}
			assert.Equal(t, "test-trace-tag-value", tracerTag)
		}
	}

	assert.True(t, checkedProviderOverrides, "provider_override.tf.json file should be present in the uploaded artifact")
}

func TestSendApplyHandler_Success_WithDefaultProviderNoRegionSet(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create mock S3 downloader with artifact containing no providers
	const MockArtifactPath = "test-artifacts/artifact-with-default-provider-no-region-set.tar.gz"
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
		t.Fatal(err)
	}

	// Check uploaded artifact contains overrides for all providers
	entries := GetArtifactEntryNames(t, tfcServer.UploadedArtifact())

	checkedProviderOverrides := false
	for _, entry := range entries {
		if entry.FileName == "provider_override.tf.json" {
			checkedProviderOverrides = true

			// Parse the override JSON
			var overrideData map[string]interface{}
			err := json.Unmarshal([]byte(entry.FileContents), &overrideData)
			if err != nil {
				t.Error(err)
			}

			// Verify the provider section exists
			providerSection, ok := overrideData["provider"].(map[string]interface{})
			if !ok {
				t.Error("provider section not found in override file")
			}

			// Verify default AWS provider override exists
			defaultAws, ok := providerSection["aws"].(map[string]interface{})
			if !ok {
				t.Error("default aws provider override not found")
			}
			// Region should be overridden since it's defined in the config (us-east-1)
			assert.Equal(t, "narnia-west-2", defaultAws["region"])

			// Verify all providers have the tracer tag
			for providerKey, providerConfig := range providerSection {
				config := providerConfig.(map[string]interface{})
				defaultTags, ok := config["default_tags"].(map[string]interface{})
				if !ok {
					t.Errorf("default_tags not found for provider %s", providerKey)
					continue
				}
				tags, ok := defaultTags["tags"].(map[string]interface{})
				if !ok {
					t.Errorf("tags not found for provider %s", providerKey)
					continue
				}
				tracerTag, ok := tags["test-tracer-tag-key"]
				if !ok {
					t.Errorf("tracer tag not found for provider %s", providerKey)
					continue
				}
				assert.Equal(t, "test-trace-tag-value", tracerTag)
			}
		}
	}

	assert.True(t, checkedProviderOverrides, "provider_override.tf.json file should be present in the uploaded artifact")
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

	// Add the actual variables the handler needs to update
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

func TestSendApplyHandler_Success_PurgesUnknownVariables(t *testing.T) {
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

	// Add a large amount of variables to the workspace that should be removed (we use a large number to forcefully test pagination)
	numberOfVarsToCreate := 150
	for varNumber := 0; varNumber < numberOfVarsToCreate; varNumber++ {
		// Make half the variables ENV variables and the other terraform variables
		category := tfe.CategoryEnv
		if varNumber%2 == 0 {
			category = tfe.CategoryTerraform
		}

		tfcServer.AddVar(&tfe.Variable{
			Key:       fmt.Sprintf("VAR_%d", varNumber),
			Value:     "yo",
			Category:  category,
			HCL:       false,
			Sensitive: false,
			Workspace: testWorkspace,
		})
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
		Parameters: []Parameter{
			{Key: "keep_me", Value: "i want to live!"},
			{Key: "keep_me_too", Value: "i also want to live!!"},
		},
		TracerTag: tracertag.TracerTag{
			TracerTagKey:   "test-tracer-tag-key",
			TracerTagValue: "test-trace-tag-value",
		},
	}

	// Send the test request
	_, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Fatal(err)
	}

	// Check Variables were updated
	assert.Equal(t, 4, len(tfcServer.Vars[testWorkspace.ID]), "Only the 2 parameters and OIDC variables should exist after all other variables were purged")
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
	// Verify that no errors were returned
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

	// Verify that an error was returned
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

	// Unzip the file
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

		if header == nil {
			t.Error("artifact had nil header")
		}

		entryContents, err := io.ReadAll(tr)
		if err != nil {
			t.Error(err)
		}

		if entryContents == nil {
			t.Error("entry was empty")
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
