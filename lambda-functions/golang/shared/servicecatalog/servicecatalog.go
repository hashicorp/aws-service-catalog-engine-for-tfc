/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package servicecatalog

import (
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"context"
)

type ServiceCatalog interface {
	NotifyProvisionProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyProvisionProductEngineWorkflowResultInput) (*servicecatalog.NotifyProvisionProductEngineWorkflowResultOutput, error)
	NotifyTerminateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultOutput, error)
	NotifyUpdateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultOutput, error)
}

type SC struct {
	Client *servicecatalog.Client
}

func (serviceCatalog SC) NotifyProvisionProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyProvisionProductEngineWorkflowResultInput) (*servicecatalog.NotifyProvisionProductEngineWorkflowResultOutput, error) {
	return serviceCatalog.Client.NotifyProvisionProductEngineWorkflowResult(ctx, input)
}

func (serviceCatalog SC) NotifyTerminateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultOutput, error) {
	return serviceCatalog.Client.NotifyTerminateProvisionedProductEngineWorkflowResult(ctx, input)
}

func (serviceCatalog SC) NotifyUpdateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultOutput, error) {
	return serviceCatalog.Client.NotifyUpdateProvisionedProductEngineWorkflowResult(ctx, input)
}
