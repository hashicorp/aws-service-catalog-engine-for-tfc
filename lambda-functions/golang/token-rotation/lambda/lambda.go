/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package lambda

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"log"
	"os"
)

type Lambda interface {
	GetEventSourceMappingUuidTuples(ctx context.Context) (*FunctionNameUuidTuples, error)
	EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error
	DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error
}

type L struct {
	Client                   *lambda.Client
	provisioningFunctionName string
	updatingFunctionName     string
	terminatingFunctionName  string
}

type FunctionNameUuidTuples struct {
	ProvisioningLambdaEventSourceMapping *FunctionNameUuidTuple
	UpdatingLambdaEventSourceMapping     *FunctionNameUuidTuple
	TerminatingLambdaEventSourceMapping  *FunctionNameUuidTuple
}

type EventSourceMappingStatus string

// Enum values for Event Source Mapping Statuses:
// Creating, Enabling , Enabled , Disabling , Disabled , Updating , or Deleting.
const (
	EventSourceCreating  EventSourceMappingStatus = "Creating"
	EventSourceEnabling  EventSourceMappingStatus = "Enabling"
	EventSourceEnabled   EventSourceMappingStatus = "Enabled"
	EventSourceDisabling EventSourceMappingStatus = "Disabling"
	EventSourceDisabled  EventSourceMappingStatus = "Disabled"
	EventSourceUpdating  EventSourceMappingStatus = "Updating"
	EventSourceDeleting  EventSourceMappingStatus = "Deleting"
)

type FunctionNameUuidTuple struct {
	FunctionName             string
	EventSourceMapping       string
	EventSourceMappingStatus EventSourceMappingStatus
}

// NewFromConfig creates a new aws lambda client
func NewFromConfig(sdkConfig aws.Config) *L {
	innerClient := lambda.NewFromConfig(sdkConfig)
	// Get provisioning function name
	provisioningFunctionName := os.Getenv("PROVISIONING_FUNCTION_NAME")

	// Get updating function name
	updatingFunctionName := os.Getenv("UPDATING_FUNCTION_NAME")

	// Get terminating function name
	terminatingFunctionName := os.Getenv("TERMINATING_FUNCTION_NAME")
	return &L{
		Client:                   innerClient,
		provisioningFunctionName: provisioningFunctionName,
		updatingFunctionName:     updatingFunctionName,
		terminatingFunctionName:  terminatingFunctionName,
	}
}

func (l *L) getEventSourceMapping(ctx context.Context, functionName string) (*FunctionNameUuidTuple, error) {
	log.Default().Printf("getting event source mappings for function %s", functionName)
	// Get the event source mappings for each function: provisioning, updating, and terminating
	eventSourceMappings, err := l.Client.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		return nil, err
	}

	for _, eventSourceMapping := range eventSourceMappings.EventSourceMappings {
		if status, ok := ParseEventSourceMappingStatus(*eventSourceMapping.State); ok {
			return &FunctionNameUuidTuple{
				FunctionName:             functionName,
				EventSourceMapping:       *eventSourceMapping.UUID,
				EventSourceMappingStatus: status,
			}, nil
		} else {
			return nil, errors.New(fmt.Sprintf("unknown event source mapping status: %s, please file an issue in the repository: https://github.com/hashicorp/aws-service-catalog-engine-for-tfc", status))
		}
	}
	return nil, errors.New(fmt.Sprintf("event source mapping for %s not found, please re-apply the aws-service-catalog-engine-for-tfc terraform to regenerate the lambda function's event source mappings", functionName))
}

func (l *L) GetEventSourceMappingUuidTuples(ctx context.Context) (*FunctionNameUuidTuples, error) {
	functionNameUuidTuples := &FunctionNameUuidTuples{}

	provisioningMapping, err := l.getEventSourceMapping(ctx, l.provisioningFunctionName)
	if err != nil {
		return functionNameUuidTuples, err
	}
	functionNameUuidTuples.ProvisioningLambdaEventSourceMapping = provisioningMapping

	updatingMapping, err := l.getEventSourceMapping(ctx, l.updatingFunctionName)
	if err != nil {
		return functionNameUuidTuples, err
	}
	functionNameUuidTuples.UpdatingLambdaEventSourceMapping = updatingMapping

	terminatingMapping, err := l.getEventSourceMapping(ctx, l.terminatingFunctionName)
	if err != nil {
		return functionNameUuidTuples, err
	}
	functionNameUuidTuples.TerminatingLambdaEventSourceMapping = terminatingMapping

	return functionNameUuidTuples, nil
}

func (l *L) EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	log.Default().Printf("Enabling event source mapping of %s:%s", functionName, uuid)

	_, err := l.Client.UpdateEventSourceMapping(ctx, &lambda.UpdateEventSourceMappingInput{
		FunctionName: aws.String(functionName),
		UUID:         aws.String(uuid),
		Enabled:      aws.Bool(true),
	})

	return err
}

func (l *L) DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	log.Default().Printf("Disabled event source mapping of %s:%s", functionName, uuid)

	_, err := l.Client.UpdateEventSourceMapping(ctx, &lambda.UpdateEventSourceMappingInput{
		FunctionName: aws.String(functionName),
		UUID:         aws.String(uuid),
		Enabled:      aws.Bool(false),
	})

	return err
}

var (
	eventSourceMap = map[string]EventSourceMappingStatus{
		"Creating":  EventSourceCreating,
		"Enabling":  EventSourceEnabling,
		"Enabled":   EventSourceEnabled,
		"Disabling": EventSourceDisabling,
		"Disabled":  EventSourceDisabled,
		"Updating":  EventSourceUpdating,
		"Deleting":  EventSourceDeleting,
	}
)

func ParseEventSourceMappingStatus(statusString string) (EventSourceMappingStatus, bool) {
	status, ok := eventSourceMap[statusString]
	return status, ok
}
