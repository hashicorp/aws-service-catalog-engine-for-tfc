module github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang

go 1.14

require (
	github.com/aws/aws-lambda-go v1.15.0
	github.com/aws/aws-sdk-go-v2 v1.18.0
	github.com/aws/aws-sdk-go-v2/config v1.18.22
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.11.64
	github.com/aws/aws-sdk-go-v2/service/s3 v1.33.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.19.6
	github.com/aws/aws-sdk-go-v2/service/servicecatalog v1.18.3
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-tfe v1.22.0
)
