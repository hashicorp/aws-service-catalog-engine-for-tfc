/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	sc "github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/google/uuid"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/identifiers"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/secretsmanager"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/servicecatalog"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
	"github.com/hashicorp/go-tfe"
	"log"
	"fmt"
)

type NotifyRunResultHandler struct {
	serviceCatalog servicecatalog.ServiceCatalog
	secretsManager secretsmanager.SecretsManager
}

func (h NotifyRunResultHandler) HandleRequest(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {

	tfeClient, err := tfc.GetTFEClient(ctx, h.secretsManager)
	if err != nil {
		log.Fatalf("failed to initialize TFE client: %s", err)
	}

	switch {
	case request.ServiceCatalogOperation == Terminating:
		return h.NotifyTerminateResult(ctx, tfeClient, request)
	case request.ServiceCatalogOperation == Provisioning:
		return h.NotifyProvisioningResult(ctx, tfeClient, request)
	case request.ServiceCatalogOperation == Updating:
		return h.NotifyUpdatingResult(ctx, tfeClient, request)
	default:
		log.Printf("Unknown serviceCatalogOperation: %s\n", request.ServiceCatalogOperation)
		return nil, errors.New("unknown serviceCatalogOperation")
	}
}

func (h NotifyRunResultHandler) NotifyTerminateResult(ctx context.Context, tfeClient *tfe.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	// If the termination was successful, delete the workspace
	if request.ErrorMessage == "" {
		err := DeleteWorkspace(ctx, tfeClient, request)
		if err != nil {
			log.Default().Printf("failed to delete workspace: %v", err)
			request.ErrorMessage = err.Error()
		}
	}

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	}

	log.Printf("Notifying terminate result %s\n", status)

	_, err := h.serviceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResult(
		ctx,
		&sc.NotifyTerminateProvisionedProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           status,
			FailureReason:    failureReason,
			IdempotencyToken: tfe.String(uuid.New().String()),
		},
	)
	if err != nil {
		log.Default().Printf("failed to notify service catalog: %v", err)
	}

	return nil, err
}

func (h NotifyRunResultHandler) NotifyProvisioningResult(ctx context.Context, tfeClient *tfe.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	var outputs []types.RecordOutput
	var err error

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	} else {
		outputs, err = FetchRunOutputs(ctx, tfeClient, request)
		if err != nil {
			log.Default().Printf("failed to fetch run outputs, Cause: %v", err)
			status = types.EngineWorkflowStatusFailed
			failureReason = aws.String(fmt.Sprintf("Failed to fetch run outputs. If re-provisioning/updating the product fails, please file an issue in the repository: https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues or contact HashiCorp support. Cause: %v", err))
		}
	}

	log.Printf("Notifying provision result %s\n", status)

	_, err = h.serviceCatalog.NotifyProvisionProductEngineWorkflowResult(
		ctx,
		&sc.NotifyProvisionProductEngineWorkflowResultInput{
			WorkflowToken:    &request.WorkflowToken,
			RecordId:         &request.RecordId,
			Status:           status,
			FailureReason:    failureReason,
			IdempotencyToken: tfe.String(uuid.New().String()),
			Outputs:          outputs,
			ResourceIdentifier: &types.EngineWorkflowResourceIdentifier{
				UniqueTag: &types.UniqueTagResourceIdentifier{
					Key:   tfe.String(request.TracerTag.TracerTagKey),
					Value: tfe.String(request.TracerTag.TracerTagValue),
				},
			},
		},
	)
	if err != nil {
		log.Default().Printf("failed to notify service catalog: %v", err)
	}

	return nil, err
}

func (h NotifyRunResultHandler) NotifyUpdatingResult(ctx context.Context, tfeClient *tfe.Client, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	var outputs []types.RecordOutput
	var err error

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	} else {
		outputs, err = FetchRunOutputs(ctx, tfeClient, request)
		if err != nil {
			log.Default().Printf("failed to fetch run outputs, Cause: %v", err)
			status = types.EngineWorkflowStatusFailed
			failureReason = aws.String(fmt.Sprintf("Failed to fetch run outputs. If updating the product fails, please file an issue in the repository: https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues or contact HashiCorp support. Cause: %v", err))
		}
	}

	log.Printf("Notifying update result %s\n", status)
	_, err = h.serviceCatalog.NotifyUpdateProvisionedProductEngineWorkflowResult(
		ctx,
		&sc.NotifyUpdateProvisionedProductEngineWorkflowResultInput{
			IdempotencyToken: tfe.String(uuid.New().String()),
			RecordId:         &request.RecordId,
			Status:           status,
			WorkflowToken:    &request.WorkflowToken,
			FailureReason:    failureReason,
			Outputs:          outputs,
		},
	)
	if err != nil {
		log.Default().Printf("failed to notify service catalog: %v", err)
	}

	return nil, err
}

func FormatError(err string, errorMessage string) *string {
	// Simplify the error message (if possible)
	simplifiedErrorString := SimplifyError(errorMessage)

	// Check if error was due to lambda timeout
	if err == "States.Timeout" {
		return aws.String("A lambda function invoked by the state machine has timed out")
	}

	// Max error message length is 2048
	if len(simplifiedErrorString) <= (2048) {
		return aws.String(simplifiedErrorString)
	}

	// Truncate error message to fit maximum failure reason length allowed by Service Catalog.
	// We use 2045 to make room for the ellipsis.
	return aws.String(simplifiedErrorString[:2045] + "...")
}

func DeleteWorkspace(ctx context.Context, client *tfe.Client, request NotifyRunResultRequest) error {
	// Get workspace name
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Make a call to delete workspace
	err := client.Workspaces.Delete(ctx, request.TerraformOrganization, workspaceName)

	return err
}
