package main

import (
	"testing"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/testtfc"
)

func TestSendDestroyHandler_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{
		Name: "123456789042-amazingly-great-product-instance",
	})

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Error(err)
	}

	// Create a test instance of the Lambda function
	testHandler := &SendDestroyHandler{
		tfeClient: tfeClient,
	}

	// Create test request
	testRequest := SendDestroyRequest{
		AwsAccountId:          "123456789042",
		TerraformOrganization: tfcServer.OrganizationName,
		ProvisionedProductId:  "amazingly-great-product-instance",
	}

	// Send the test request
	response, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify no errors were returned
	if err != nil {
		t.Error(err)
	}

	// Verify a new run was added
	runPath := fmt.Sprintf("/api/v2/runs/%s", response.TerraformRunId)
	destroyRun := tfcServer.Runs[runPath]

	assert.NotNil(t, destroyRun, "A run should have been created")
	assert.True(t, destroyRun.IsDestroy, "The new run should be a destroy run")
}

func TestSendDestroyHandler_WorkspaceMissing(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	// Create tfe client that will send requests to the mock TFC instance
	tfeClient, err := tfc.ClientWithDefaultConfig(tfcServer.Address, "supers3cret")
	if err != nil {
		t.Error(err)
	}

	// Create a test instance of the Lambda function
	testHandler := &SendDestroyHandler{
		tfeClient: tfeClient,
	}

	// Create test request
	testRequest := SendDestroyRequest{
		AwsAccountId:          "123456789042",
		TerraformOrganization: tfcServer.OrganizationName,
		ProvisionedProductId:  "amazingly-great-product-instance",
	}

	// Send the test request
	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	// Verify the handler returned an error
	assert.Error(t, err, "Handler should have responded with an error")
}
