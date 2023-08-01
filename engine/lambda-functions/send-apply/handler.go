/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/fileutils"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/identifiers"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/go-tfe"
	"time"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
)

type SendApplyHandler struct {
	secretsManager   secretsmanager.SecretsManager
	s3Downloader     fileutils.S3Downloader
	region           string
	terraformVersion string
}

func (h *SendApplyHandler) HandleRequest(ctx context.Context, request SendApplyRequest) (*SendApplyResponse, error) {
	// Create TFC Applier to ensure that metadata headers are supplied in requests
	applier, err := h.NewTFCApplier(ctx, request)
	if err != nil {
		return nil, err
	}

	// Find or create the Project
	projectName := request.ProductId
	p, err := applier.FindOrCreateProject(ctx, request.TerraformOrganization, projectName)
	if err != nil {
		return nil, err
	}

	// Create or find the Workspace
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)
	w, err := applier.FindOrCreateWorkspace(ctx, request.TerraformOrganization, p, workspaceName)
	if err != nil {
		return nil, err
	}

	// Update Terraform Version
	err = applier.UpdateWorkspaceTerraformVersion(ctx, w.ID)
	if err != nil {
		return nil, err
	}

	// Remove all non-recognized variables from the workspace. This helps ensure parity between Service Catalog and TFC
	err = applier.PurgeVariables(ctx, w, request.Parameters)
	if err != nil {
		return nil, err
	}

	// Configure ENV variables for OIDC
	err = applier.UpdateWorkspaceOIDCVariables(ctx, w, request.LaunchRoleArn)
	if err != nil {
		return nil, err
	}

	// Configure Terraform variables provided via the product parameters from Service Catalog
	err = applier.UpdateWorkspaceParameterVariables(ctx, w, request.Parameters)
	if err != nil {
		return nil, err
	}

	// Create configuration version to acquire upload link for configuration files to be sent to
	cv, err := applier.CreateConfigurationVersion(ctx, w.ID)
	if err != nil {
		return nil, err
	}

	// Download product configuration files
	sourceProductConfig, err := fileutils.DownloadS3File(ctx, h.s3Downloader, request.LaunchRoleArn, request.Artifact.Path)
	if err != nil {
		return nil, err
	}

	// Create override files for injecting AWS default tags
	providerOverrides, _ := CreateAWSProviderOverrides(h.region, request.Tags, request.TracerTag)

	// Inject AWS default tags, via the override file, into the tar file
	modifiedProductConfig, err := InjectOverrides(sourceProductConfig, []ConfigurationOverride{*providerOverrides})
	if err != nil {
		return nil, err
	}

	// Upload newly modified configuration to TFE
	err = applier.tfeClient.ConfigurationVersions.UploadTarGzip(ctx, cv.UploadURL, modifiedProductConfig)
	if err != nil {
		return nil, err
	}

	uploadTimeoutInSeconds := 120
	for i := 0; ; i++ {
		refreshed, err := applier.tfeClient.ConfigurationVersions.Read(ctx, cv.ID)
		if err != nil {
			return nil, tfc.Error(err)
		}

		if refreshed.Status == tfe.ConfigurationUploaded {
			break
		}

		if i > uploadTimeoutInSeconds {
			return nil, err
		}

		time.Sleep(1 * time.Second)
	}

	run, err := applier.tfeClient.Runs.Create(ctx, tfe.RunCreateOptions{
		Workspace:            w,
		ConfigurationVersion: cv,
		AutoApply:            tfe.Bool(true),
	})
	if err != nil {
		return nil, tfc.Error(err)
	}

	return &SendApplyResponse{TerraformRunId: run.ID}, err
}
