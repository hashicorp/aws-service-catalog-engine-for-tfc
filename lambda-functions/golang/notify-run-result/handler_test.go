package main

import (
	"testing"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/servicecatalog"
	"github.com/stretchr/testify/assert"
	"github.com/hashicorp/go-tfe"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/testtfc"
)

func TestNotifyRunResultHandler_Terminating_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{Name: "yolo"})
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

	// Verify the workflow was successfully reported as a success
	assert.Equal(t, types.EngineWorkflowStatusSucceeded, mockServiceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput.Status)

	// Verify workflow token
	assert.Equal(t, testRequest.WorkflowToken, *mockServiceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput.WorkflowToken)
}

func TestNotifyRunResultHandler_Terminating_WithError(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{Name: "yolo"})
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
		Error:                   "My.Bad",
		ErrorMessage:            "you win some, you lose some",
	}

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Verify the TFC workspace was deleted
	assert.Equal(t, 0, len(tfcServer.Workspaces), "The TFC workspace should have been deleted")

	// Verify the workflow was successfully reported as a failure
	assert.Equal(t, types.EngineWorkflowStatusFailed, mockServiceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput.Status)

	// Verify Error was successfully returned
	assert.Equal(t, testRequest.ErrorMessage, *mockServiceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput.FailureReason)

	// Verify workflow token
	assert.Equal(t, testRequest.WorkflowToken, *mockServiceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput.WorkflowToken)
}

func TestNotifyRunResultHandler_Provisioning_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	testWorkspace := tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{Name: "yolo"})

	// Add a Run
	tfcServer.AddRun("run-forrest-run", testtfc.RunFactoryParameters{
		RunStatus: tfe.RunApplied,
		Apply: &tfe.Apply{
			ID: "apply-ran-ed",
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

	// Verify workflow token
	assert.Equal(t, testRequest.WorkflowToken, *mockServiceCatalog.NotifyProvisionProductEngineWorkflowResultInput.WorkflowToken)
}

func TestNotifyRunResultHandler_Provisioning_MissingApply(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{Name: "yolo"})

	// Add a Run
	tfcServer.AddRun("run-forrest-run", testtfc.RunFactoryParameters{
		RunStatus: tfe.RunApplied,
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
	// Verify that an error was returned so that the lambda can be retried
	assert.NotNil(t, err, "Error should have been returned by the Lambda")

	// Verify the TFC workspace was not deleted (like in the terminating workflow)
	assert.Equal(t, 1, len(tfcServer.Workspaces), "The TFC workspace should have been deleted")

	// Verify the workflow was not reported as a failure, so the lambda can be retried
	assert.Nil(t, mockServiceCatalog.NotifyProvisionProductEngineWorkflowResultInput)
}

func TestNotifyRunResultHandler_Updating_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Add a workspace to the TFC instance
	testWorkspace := tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{Name: "yolo"})

	// Add a Run
	tfcServer.AddRun("run-forrest-run", testtfc.RunFactoryParameters{
		RunStatus: tfe.RunApplied,
		Apply: &tfe.Apply{
			ID: "apply-ran-ed",
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
		ServiceCatalogOperation: Updating,
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
	assert.Equal(t, types.EngineWorkflowStatusSucceeded, mockServiceCatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput.Status)

	// Verify the outputs were published correctly
	actualOutputs := mockServiceCatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput.Outputs
	assert.Equal(t, 1, len(actualOutputs))
	for _, actualOutput := range actualOutputs {
		assert.Equal(t, testStateVersionOutput.Name, *actualOutput.OutputKey)
		assert.Equal(t, testStateVersionOutput.Value, *actualOutput.OutputValue)
		assert.Nil(t, actualOutput.Description)
	}

	// Verify workflow token
	assert.Equal(t, testRequest.WorkflowToken, *mockServiceCatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput.WorkflowToken)
}
