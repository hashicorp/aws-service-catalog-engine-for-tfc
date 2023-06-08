package main

import (
	"testing"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/servicecatalog"
)

func TestNotifyRunResultHandler_Terminating_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testutil.NewMockTFC()
	defer tfcServer.Stop()

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
}
