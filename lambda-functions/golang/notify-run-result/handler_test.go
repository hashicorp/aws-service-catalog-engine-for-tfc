package main

import (
	"testing"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/servicecatalog"
	"github.com/stretchr/testify/assert"
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
		serviceCatalog: mockServiceCatalog,
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
