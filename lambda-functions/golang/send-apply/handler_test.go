package main

import (
	"testing"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil"
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tracertag"
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

	// Create a test instance of the Lambda function
	testHandler := &SendApplyHandler{
		tfeClient: tfeClient,
		s3Client:  nil,
		region:    "us-west-2",
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
		ProductId:     "id-4-number-1-best-producy",
		Tags:          make([]AWSTag, 0),
		TracerTag: tracertag.TracerTag{
			TracerTagKey:   "test-tracer-tag-key",
			TracerTagValue: "test-trace-tag-value",
		},
	}

	_, err = testHandler.HandleRequest(context.Background(), testRequest)
	if err != nil {
		t.Error(err)
	}

}
