/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
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
		log.Default().Printf("failed to initialize TFE client: %s", err)
		return nil, err
	}

	workspaceId := getWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Get the workspace
	workspace, err := tfeClient.Workspaces.Read(ctx, request.TerraformOrganization, workspaceId)
	if err != nil {
		log.Default().Printf("Workspace does not exist or couldn't be found: %s", err)
		return nil, tfc.Error(err)
	}

	// Queue "Terraform destroy"
	run, err := tfeClient.Runs.Create(ctx, tfe.RunCreateOptions{
		IsDestroy: tfe.Bool(true),
		Message:   tfe.String("Terminating example-product via AWS Service Catalog"),
		Workspace: workspace,
		AutoApply: tfe.Bool(true),
	})
	if err != nil {
		log.Default().Printf("Failed to queue destroy run: %s", err)
		return nil, tfc.Error(err)
	}

	return &SendDestroyResponse{TerraformRunId: run.ID}, err
}
