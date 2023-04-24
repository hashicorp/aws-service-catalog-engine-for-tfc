package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/go-tfe"
	"log"
)

type MyEvent struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
	client, err := tfe.NewClient(tfe.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	orgs, err := client.Organizations.List(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, org := range orgs.Items {
		fmt.Printf("Hello %s!", org.Name)
	}

	return fmt.Sprintf("Hello %s!", name.Name), nil
}

func main() {
	lambda.Start(HandleRequest)
}
