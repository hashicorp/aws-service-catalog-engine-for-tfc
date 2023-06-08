package servicecatalog

import (
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"context"
)

type MockServiceCatalog struct {
}

func (serviceCatalog MockServiceCatalog) NotifyProvisionProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyProvisionProductEngineWorkflowResultInput) (*servicecatalog.NotifyProvisionProductEngineWorkflowResultOutput, error) {
	return nil, nil
}

func (serviceCatalog MockServiceCatalog) NotifyTerminateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyTerminateProvisionedProductEngineWorkflowResultOutput, error) {
	return nil, nil
}

func (serviceCatalog MockServiceCatalog) NotifyUpdateProvisionedProductEngineWorkflowResult(ctx context.Context, input *servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultInput) (*servicecatalog.NotifyUpdateProvisionedProductEngineWorkflowResultOutput, error) {
	return nil, nil
}
