/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package servicecatalog

import (
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"context"
)

type MockServiceCatalog struct {
	NotifyProvisionProductEngineWorkflowResultInput            *servicecatalog.NotifyProvisionProductEngineWorkflowResultInput
	NotifyTerminateProvisionedProductEngineWorkflowResultInput *servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput
	NotifyUpdateProvisionedProductEngineWorkflowResultInput    *servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput
}

func (serviceCatalog *MockServiceCatalog) NotifyProvisionProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyProvisionProductEngineWorkflowResultInput) (*servicecatalog.NotifyProvisionProductEngineWorkflowResultOutput, error) {
	serviceCatalog.NotifyProvisionProductEngineWorkflowResultInput = input
	return nil, nil
}

func (serviceCatalog *MockServiceCatalog) NotifyTerminateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultOutput, error) {
	serviceCatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput = input
	return nil, nil
}

func (serviceCatalog *MockServiceCatalog) NotifyUpdateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultOutput, error) {
	serviceCatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput = input
	return nil, nil
}
