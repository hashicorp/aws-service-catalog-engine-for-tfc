package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/identifiers"
	"github.com/hashicorp/go-tfe"
	"github.com/aws/aws-sdk-go-v2/aws"
	"log"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/google/uuid"
	"errors"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/servicecatalog"
	sc "github.com/aws/aws-sdk-go-v2/service/servicecatalog"
)

type NotifyRunResultHandler struct {
	serviceCatalog servicecatalog.ServiceCatalog
	tfeClient      *tfe.Client
}

func (h NotifyRunResultHandler) HandleRequest(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	switch {
	case request.ServiceCatalogOperation == Terminating:
		return h.NotifyTerminateResult(ctx, request)
	case request.ServiceCatalogOperation == Provisioning:
		return h.NotifyProvisioningResult(ctx, request)
	case request.ServiceCatalogOperation == Updating:
		return h.NotifyUpdatingResult(ctx, request)
	default:
		log.Printf("Unknown serviceCatalogOperation: %s\n", request.ServiceCatalogOperation)
		return nil, errors.New("unknown serviceCatalogOperation")
	}
}

func (h NotifyRunResultHandler) NotifyTerminateResult(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	// Delete the workspace
	err := DeleteWorkspace(ctx, h.tfeClient, request)
	if err != nil {
		request.ErrorMessage = err.Error()
	}

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = aws.String(request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	}

	log.Printf("Notifying terminate result %s\n", status)

	_, err = h.serviceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResult(
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
		log.Fatal(err)
	}

	return nil, err
}

func (h NotifyRunResultHandler) NotifyProvisioningResult(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	var outputs []types.RecordOutput
	var err error

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	} else {
		outputs, err = FetchRunOutputs(ctx, h.tfeClient, request)
		if err != nil {
			return nil, err
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
		log.Fatal(err)
	}

	return nil, err
}

func (h NotifyRunResultHandler) NotifyUpdatingResult(ctx context.Context, request NotifyRunResultRequest) (*NotifyRunResultResponse, error) {
	var outputs []types.RecordOutput
	var err error

	var status = types.EngineWorkflowStatusSucceeded
	var failureReason *string = nil
	if request.ErrorMessage != "" {
		failureReason = FormatError(request.Error, request.ErrorMessage)
		status = types.EngineWorkflowStatusFailed
	} else {
		outputs, err = FetchRunOutputs(ctx, h.tfeClient, request)
		if err != nil {
			return nil, err
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
		log.Fatal(err)
	}

	return nil, err
}

func FormatError(err string, errorMessage string) *string {
	// Check if error was due to lambda timeout
	if err == "States.Timeout" {
		return aws.String("A lambda function invoked by the state machine has timed out")
	}

	// Max error message length is 2048
	if len(errorMessage) <= (2048) {
		return aws.String(errorMessage)
	}

	// Truncate error message to fit maximum failure reason length allowed by Service Catalog.
	// We use 2045 to make room for the ellipsis.
	return aws.String(errorMessage[:2045] + "...")
}

func DeleteWorkspace(ctx context.Context, client *tfe.Client, request NotifyRunResultRequest) error {
	// Get workspace name
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)

	// Make a call to delete workspace
	err := client.Workspaces.Delete(ctx, request.TerraformOrganization, workspaceName)

	return err
}
