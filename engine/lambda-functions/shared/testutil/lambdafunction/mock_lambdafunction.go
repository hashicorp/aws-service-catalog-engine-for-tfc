/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package lambdafunction

import (
	"context"
	"errors"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/token-rotation/lambda"
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

func boolToEventSourceMappingStatus(bool bool) lambda.EventSourceMappingStatus {
	if bool {
		return lambda.EventSourceEnabled
	} else {
		return lambda.EventSourceDisabled
	}
}

func (l *MockLambdaFunction) GetEventSourceMappingUuidTuples(ctx context.Context) (*lambda.FunctionNameUuidTuples, error) {
	provisioningUuidTuple := &lambda.FunctionNameUuidTuple{"provisioningFunctionName", "provisioningUuid", boolToEventSourceMappingStatus(l.Provisioning)}
	updatingUuidTuple := &lambda.FunctionNameUuidTuple{"updatingFunctionName", "updatingUuid", boolToEventSourceMappingStatus(l.Updating)}
	terminatingUuidTuple := &lambda.FunctionNameUuidTuple{"terminatingFunctionName", "terminatingUuid", boolToEventSourceMappingStatus(l.Terminating)}
	return &lambda.FunctionNameUuidTuples{
		ProvisioningLambdaEventSourceMapping: provisioningUuidTuple,
		UpdatingLambdaEventSourceMapping:     updatingUuidTuple,
		TerminatingLambdaEventSourceMapping:  terminatingUuidTuple,
	}, nil
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

func (l *MockLambdaFunctionWithErrorResponse) GetEventSourceMappingUuidTuples(ctx context.Context) (*lambda.FunctionNameUuidTuples, error) {
	return nil, errors.New("function name or uuid not found")
}

func (l *MockLambdaFunctionWithErrorResponse) EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	return errors.New("function name or uuid not found")
}

func (l *MockLambdaFunctionWithErrorResponse) DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	return errors.New("function name or uuid not found")
}
