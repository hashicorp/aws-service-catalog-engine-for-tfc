package main

import (
	"testing"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/servicecatalog"
	"github.com/stretchr/testify/assert"
	"github.com/hashicorp/go-tfe"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
)

func TestNotifyRunResultHandler_Terminating_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testutil.WorkspaceFactoryParameters{Name: "yolo"})
	assert.Equal(t, 1, len(tfcServer.Workspaces), "Make sure the TFC instance has only 1 workspace")

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Error(err)
	}

	// Create mock ServiceCatalog
	mockServiceCatalog := servicecatalog.MockServiceCatalog{}

	// Create a test instance of the Lambda function
	testHandler := &NotifyRunResultHandler{
		serviceCatalog: &mockServiceCatalog,
		tfeClient:      tfeClient,
	}

	// Create test request
	testRequest := NotifyRunResultRequest{
		TerraformRunId: "run-forrest-run",
		WorkflowToken:  "whistle-while-you-work",
		RecordId:       "record-this-id",
		TracerTag: tracertag.TracerTag{
			TracerTagKey:   "test-tracer-tag-key",
			TracerTagValue: "test-trace-tag-value",
		},
		ServiceCatalogOperation: Terminating,
		AwsAccountId:            "123456789042",
		TerraformOrganization:   tfcServer.OrganizationName,
		ProvisionedProductId:    "amazingly-great-product-instance",
		Error:                   "",
		ErrorMessage:            "",
	}

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Verify the TFC workspace was deleted
	assert.Equal(t, 0, len(tfcServer.Workspaces), "The TFC workspace should have been deleted")
}

func TestNotifyRunResultHandler_Provisioning_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	testWorkspace := tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testutil.WorkspaceFactoryParameters{Name: "yolo"})

	// Add a Run
	tfcServer.AddRun("run-forrest-run", testutil.RunFactoryParameters{
		RunStatus: tfe.RunApplied,
		Apply: &tfe.Apply{
			ID:                   "apply-ran-ed",
			LogReadURL:           "some-log-read-url",
			ResourceAdditions:    1337,
			ResourceChanges:      42,
			ResourceDestructions: 21,
			Status:               tfe.ApplyFinished,
			StatusTimestamps:     nil,
		},
	})

	// Add an Apply
	tfcServer.AddApply("apply-ran-ed", &tfe.Apply{
		ID:                   "apply-ran-ed",
		LogReadURL:           "some-log-read-url",
		ResourceAdditions:    1337,
		ResourceChanges:      42,
		ResourceDestructions: 21,
		Status:               tfe.ApplyFinished,
		StatusTimestamps:     nil,
	})

	testStateVersionOutput := tfe.StateVersionOutput{
		Name:      "super_valuable_information_about_your_infra",
		Sensitive: true,
		Type:      "string",
		Value:     "yourmomsayshi...JINX",
	}

	tfcServer.AddStateVersion(testWorkspace.ID, &tfe.StateVersion{
		Outputs: []*tfe.StateVersionOutput{&testStateVersionOutput},
	})

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Error(err)
	}

	// Create mock ServiceCatalog
	mockServiceCatalog := servicecatalog.MockServiceCatalog{}

	// Create a test instance of the Lambda function
	testHandler := &NotifyRunResultHandler{
		serviceCatalog: &mockServiceCatalog,
		tfeClient:      tfeClient,
	}

	// Create test request
	testRequest := NotifyRunResultRequest{
		TerraformRunId: "run-forrest-run",
		WorkflowToken:  "whistle-while-you-work",
		RecordId:       "record-this-id",
		TracerTag: tracertag.TracerTag{
			TracerTagKey:   "test-tracer-tag-key",
			TracerTagValue: "test-trace-tag-value",
		},
		ServiceCatalogOperation: Provisioning,
		AwsAccountId:            "123456789042",
		TerraformOrganization:   tfcServer.OrganizationName,
		ProvisionedProductId:    "amazingly-great-product-instance",
		Error:                   "",
		ErrorMessage:            "",
	}

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Verify the TFC workspace was not deleted (like in the terminating workflow)
	assert.Equal(t, 1, len(tfcServer.Workspaces), "The TFC workspace should have been deleted")

	// Verify the workflow was successfully reported as a success
	assert.Equal(t, types.EngineWorkflowStatusSucceeded, mockServiceCatalog.NotifyProvisionProductEngineWorkflowResultInput.Status)

	// Verify the outputs were published correctly
	actualOutputs := mockServiceCatalog.NotifyProvisionProductEngineWorkflowResultInput.Outputs
	assert.Equal(t, 1, len(actualOutputs))
	for _, actualOutput := range actualOutputs {
		assert.Equal(t, testStateVersionOutput.Name, *actualOutput.OutputKey)
		assert.Equal(t, testStateVersionOutput.Value, *actualOutput.OutputValue)
		assert.Nil(t, actualOutput.Description)
	}
}
