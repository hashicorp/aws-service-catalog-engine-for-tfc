package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/tfc"
	"github.com/hashicorp/go-tfe"
	"log"
)

type SendDestroyHandler struct {
	secretsManager secretsmanager.SecretsManager
}

func (h *SendDestroyHandler) HandleRequest(ctx context.Context, request SendDestroyRequest) (*SendDestroyResponse, error) {
	// Get TFE Client
	tfeClient, err := tfc.GetTFEClient(ctx, h.secretsManager)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	workspaceId := getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Get the workspace
	workspace, err := tfeClient.Workspaces.Read(ctx, request.TerraformOrganization, workspaceId)
	if err != nil {
		log.Printf("Workspace does not exist or couldn't be found: %s", err)
		return nil, err
	}

	// Queue "Terraform destroy"
	run, err := tfeClient.Runs.Create(ctx, tfe.RunCreateOptions{
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
