package main

import (
	"log"
	"github.com/hashicorp/go-tfe"
	"context"
)

type SendDestroyHandler struct {
	tfeClient *tfe.Client
}

func (h *SendDestroyHandler) HandleRequest(ctx context.Context, request SendDestroyRequest) (*SendDestroyResponse, error) {
	workspaceId := getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Get the workspace
	workspace, err := h.tfeClient.Workspaces.Read(ctx, request.TerraformOrganization, workspaceId)
	if err != nil {
		log.Printf("Workspace does not exist or couldn't be found: %s", err)
		return nil, err
	}

	// Queue "Terraform destroy"
	run, err := h.tfeClient.Runs.Create(ctx, tfe.RunCreateOptions{
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
