package lambdafunction

import (
	"context"
	"errors"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/lambda-functions/golang/token-rotation/lambda"
)

type MockLambdaFunction struct {
	Provisioning bool
	Updating     bool
	Terminating  bool
}

type MockLambdaFunctionWithErrorResponse struct {
	Provisioning bool
	Updating     bool
	Terminating  bool
}

func (l *MockLambdaFunction) GetEventSourceMappingUuidTuples(ctx context.Context, functionNames []string) ([]lambda.FunctionNameUuidTuple, error) {
	provisioningUuidTuple := lambda.FunctionNameUuidTuple{"provisioningFunctionName", "provisioningUuid"}
	updatingUuidTuple := lambda.FunctionNameUuidTuple{"updatingFunctionName", "updatingUuid"}
	terminatingUuidTuple := lambda.FunctionNameUuidTuple{"terminatingFunctionName", "terminatingUuid"}
	return []lambda.FunctionNameUuidTuple{provisioningUuidTuple, updatingUuidTuple, terminatingUuidTuple}, nil
}

func (l *MockLambdaFunction) EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	if functionName == "provisioningFunctionName" && uuid == "provisioningUuid" {
		l.Provisioning = true
		return nil
	}

	if functionName == "updatingFunctionName" && uuid == "updatingUuid" {
		l.Updating = true
		return nil
	}

	if functionName == "terminatingFunctionName" && uuid == "terminatingUuid" {
		l.Terminating = true
		return nil
	}

	return errors.New("function name or uuid not found")
}

func (l *MockLambdaFunction) DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	if functionName == "provisioningFunctionName" && uuid == "provisioningUuid" {
		l.Provisioning = false
		return nil
	}

	if functionName == "updatingFunctionName" && uuid == "updatingUuid" {
		l.Updating = false
		return nil
	}

	if functionName == "terminatingFunctionName" && uuid == "terminatingUuid" {
		l.Terminating = false
		return nil
	}

	return errors.New("function name or uuid not found")
}

func (l *MockLambdaFunctionWithErrorResponse) GetEventSourceMappingUuidTuples(ctx context.Context, functionNames []string) ([]lambda.FunctionNameUuidTuple, error) {
	return nil, errors.New("function name or uuid not found")
}

func (l *MockLambdaFunctionWithErrorResponse) EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	return errors.New("function name or uuid not found")
}

func (l *MockLambdaFunctionWithErrorResponse) DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	return errors.New("function name or uuid not found")
}
