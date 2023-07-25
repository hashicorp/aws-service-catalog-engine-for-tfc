/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"fmt"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/testutil/testtfc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSendDestroyHandler_Success(t *testing.T) {
	// Create mock TFC instance
	tfcServer := testtfc.NewMockTFC()
	defer tfcServer.Stop()

	tfcServer.AddWorkspace("123456789042-amazingly-great-product-instance", testtfc.WorkspaceFactoryParameters{
		Name: "123456789042-amazingly-great-product-instance",
	})

	// Create tfe client that will send requests to the mock TFC instance
	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create a test instance of the Lambda function
	testHandler := &SendDestroyHandler{
		secretsManager: mockSecretsManager,
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

	// Create the TFE client that will send requests to the mock TFC instance
	mockSecretsManager := &secretsmanager.MockSecretsManager{
		Hostname: tfcServer.Address,
		TeamId:   "team-4123nlol",
		Token:    "supers3cret",
	}

	// Create a test instance of the Lambda function
	testHandler := &SendDestroyHandler{
		secretsManager: mockSecretsManager,
	}

	// Create test request
	testRequest := SendDestroyRequest{
		AwsAccountId:          "123456789042",
		TerraformOrganization: tfcServer.OrganizationName,
		ProvisionedProductId:  "amazingly-great-product-instance",
	}

	// Send the test request
	_, err := testHandler.HandleRequest(context.Background(), testRequest)
	// Verify the handler returned an error
	assert.Error(t, err, "Handler should have responded with an error")
}
