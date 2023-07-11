package lambda

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"log"
)

type Lambda interface {
	GetEventSourceMappingUuidTuples(ctx context.Context, functionNames []string) ([]FunctionNameUuidTuple, error)
	EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error
	DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error
}

type L struct {
	Client *lambda.Client
}

type FunctionNameUuidTuple struct {
	FunctionName       string
	EventSourceMapping string
}

// NewFromConfig creates a new aws lambda client
func NewFromConfig(sdkConfig aws.Config) *L {
	innerClient := lambda.NewFromConfig(sdkConfig)

	return &L{
		Client: innerClient,
	}
}

func (l *L) GetEventSourceMappingUuidTuples(ctx context.Context, functionNames []string) ([]FunctionNameUuidTuple, error) {
	var functionNameUuidTuples []FunctionNameUuidTuple

	for _, functionName := range functionNames {
		log.Default().Printf("getting event source mappings for function %s", functionName)
		// Get the event source mapping UUIDs and disable the SQS queues
		eventSourceMappings, err := l.Client.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
			FunctionName: aws.String(functionName),
		})

		if err != nil {
			return nil, err
		}

		for _, eventSourceMapping := range eventSourceMappings.EventSourceMappings {
			functionNameUuidTuples = append(functionNameUuidTuples, FunctionNameUuidTuple{
				FunctionName:       functionName,
				EventSourceMapping: *eventSourceMapping.UUID,
			})
		}
	}

	return functionNameUuidTuples, nil
}

func (h *L) EnableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	log.Default().Printf("Enabling event source mapping of %s:%s", functionName, uuid)

	_, err := h.Client.UpdateEventSourceMapping(ctx, &lambda.UpdateEventSourceMappingInput{
		FunctionName: aws.String(functionName),
		UUID:         aws.String(uuid),
		Enabled:      aws.Bool(true),
	})

	return err
}

func (h *L) DisableEventSourceMapping(ctx context.Context, functionName string, uuid string) error {
	log.Default().Printf("Disabled event source mapping of %s:%s", functionName, uuid)

	_, err := h.Client.UpdateEventSourceMapping(ctx, &lambda.UpdateEventSourceMappingInput{
		FunctionName: aws.String(functionName),
		UUID:         aws.String(uuid),
		Enabled:      aws.Bool(false),
	})

	return err
}
