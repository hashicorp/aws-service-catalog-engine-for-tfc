# Contributing

## Pull Requests
All pull requests require at least one approval from the [CODEOWNERS](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/blob/main/.github/CODEOWNERS). Before merge, all pull request checks must pass, including Go tests.

## Bug Reports
To file a bug report or to provide feedback on the TFC-RE, please [open a GitHub issue](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues). Please try to provide as much detail as possible so that we are able to help you as quickly as possible.

## Build the Code
To build the Go code and Lambda Functions, do the following:
1. `cd` into `engine/lambda-functions/golang`.
2. Run `make bin` to build the Lambda functions and install the necessary dependencies.

Note: Any time you update a Lambda Function, you will need to build the code, as outlined above, to apply those changes.
