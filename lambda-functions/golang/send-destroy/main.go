package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/awsconfig"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfeauth"
	"github.com/hashicorp/go-tfe"
	"log"
)

type SendDestroyRequest struct {
	AwsAccountId          string `json:"awsAccountId"`
	TerraformOrganization string `json:"terraformOrganization"`
	ProvisionedProductId  string `json:"provisionedProductId"`
}

type SendDestroyResponse struct {
	TerraformRunId string `json:"terraformRunId"`
}

func HandleRequest(ctx context.Context, request SendDestroyRequest) (*SendDestroyResponse, error) {
	sdkConfig := awsconfig.GetSdkConfig(ctx)

	client, err := tfeauth.GetTFEClient(ctx, sdkConfig)
	if err != nil {
		log.Printf("Failed to initialize TFE client: %s", err)
		return nil, err
	}

	workspaceId := getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Get the workspace
	workspace, err := client.Workspaces.Read(ctx, request.TerraformOrganization, workspaceId)
	if err != nil {
		log.Printf("Workspace does not exist or couldn't be found: %s", err)
		return nil, err
	}

	// Queue "Terraform destroy"
	run, err := client.Runs.Create(ctx, tfe.RunCreateOptions{
		IsDestroy: tfe.Bool(true),
		Message:   tfe.String("Terminating example-product via AWS Service Catalog"),
		Workspace: workspace,
		AutoApply: tfe.Bool(true),
	})
	if err != nil {
		log.Printf("Failed to queue destroy run: %s", err)
		return nil, err
	}

	return &SendDestroyResponse{TerraformRunId: run.ID}, err
}

func main() {
	lambda.Start(HandleRequest)
}

// Get the workspace name, which is `${accountId} - ${provisionedProductId}`
func getWorkspaceName(awsAccountId string, provisionedProductId string) string {
	return fmt.Sprintf("%s-%s", awsAccountId, provisionedProductId)
}
