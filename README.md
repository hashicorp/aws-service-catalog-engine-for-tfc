# AWS Service Catalog Engine for Terraform Cloud
The AWS Service Catalog Terraform Cloud Reference Engine (TFC-RE) is a configurable Terraform Cloud Reference Engine that can be installed using AWS Service Catalog. This integration gives administrators governance and visibility into their Terraform workloads, and allows Service Catalog administrators to delegate cloud resource provisioning responsibilities to users within their organizations. For more information about using Terraform Cloud with AWS Service Catalog, see **<insert AWS docs on how to get started here>**.

# Prerequisites
The installation can be done from any Linux or Mac machine.

## Install the Tools
1. Install the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) tools.
2. Install [Go](https://go.dev/doc/install).
3. Install the [Terraform CLI](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli) tools.

# Install the Terraform Cloud Reference Engine

## Getting Started

### Set Up Your Environment
1. `git clone` the project.
2. Export the following environment variables:

   `AWS_ACCOUNT_ID=<YOUR AWS ACCOUNT ID>
   AWS_REGION=<YOUR REGION OF CHOICE>`

For further information regarding credentials, please follow the steps outlined in their AWS [developer guide](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html).

### Build the Code
To build the Go code and lambda functions, do the following:
1. `cd` into `lambda-functions/golang`.
2. Run `make bin` to build the Lambda functions and install the necessary dependencies.

### Launch the Engine
To launch the engine, you'll need to set the `TFE_TOKEN` environment variable to a Terraform Cloud user token. It is important to note that this user token must have permission to create workspaces within your organization in order for them to provision products.

You'll also need to authenticate the AWS provider as you would normally, using one of the methods mentioned in the AWS provider documentation [here](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration).

Additionally, you’ll need to do the following:

1. Copy `terraform.tfvars.example` to `terraform.tfvars` and set the organization name to your Terraform Cloud organization name.
2. Run `terraform plan` to verify your setup, and then run `terraform apply` to apply your changes.

### Test the Engine
Once you've applied the configuration, you should see a newly created AWS Service Catalog portfolio in [your AWS Service Catalog dashboard](https://console.aws.amazon.com/servicecatalog/home).

To test your newly provisioned Service Catalog Engine for Terraform Cloud, follow [the guide to granting access to portfolios](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/catalogs_portfolios_users.html). Navigate to the newly provisioned `"TFC Example Portfolio"` and grant access to a user of your choosing. Instruct the newly assigned "test user" to attempt to provision the included example product that this engine creates (it is already assigned to the `"TFC Example Portfolio"`).

The example product mentioned above can be found [here](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/tree/main/example-product).

# Troubleshooting

## Terraform Authentication
If you run into TFC workspace issues, it may mean that the user that has been granted access to launch products within your AWS Service Catalog account may not have the correct set of permissions on Terraform Cloud.

**Solution:** Grant users permission to create workspaces within your organization.

# Creating and Provisioning a Product in Service Catalog
The TFC-RE creates an example product upon launch, however, if you’d prefer to create a new product using the Service Catalog UI, please refer to the steps outlined in this section. For more information on how to provision a product using Terraform Cloud and AWS Service Catalog, please refer to this **<insert documentation on how to do this, here>** documentation.

## Create a Product

1. To create a product in Service Catalog, do the following:
    - Select the “Terraform Cloud” option.
    - Set the product type to `TERRAFORM_CLOUD_SOURCE`.
    - Set the provisioning artifact type as `TERRAFORM_CLOUD_SOURCE`.
    - Upon initial setup, a `.tar.gz` file containing your Terraform Cloud configuration should have been created. Use this `tar` file as the provisioning artifact file. Alternatively, you can use an S3 bucket URL in place of the `tar` file.

## Add a Launch Role
Each and every product requires a launch constraint that indicates the IAM role that will be used to provision the product's resources. This role is known as the "launch role." For more information regarding launch roles, please refer to [this documentation](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/constraints-launch.html) on launch roles and launch constraints.

## Grant Access to the Product
In order to grant users access to the product(s) of choice and to allow those users to perform actions on that product, you’ll need to ensure that the product is within a portfolio that has the correct permissions. To do this, ensure that the user is a part of an IAM group, has the IAM role, or is an IAM user. For more information regarding IAM roles, please refer to [this documentation](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/getstarted-iamenduser.html) on Service Catalog End Users.

Additionally, once this portfolio is set up, it can be shared with other accounts.

## Provision the Product
Once the user has been granted the necessary permissions on a portfolio that contains a product, the user should now be able to provision a product. Additionally, a user should also be able to:

1. Update a provisioned product via Service Catalog, triggering a run in TFC.
2. Terminate a provisioned product via Service Catalog, triggering a destroy run in TFC.
3. Update the team token rotation frequency via AWS EventBridge, altering the frequency in which the TFC team token rotates.

# Troubleshooting

### Exceptions
There are three common exceptions that can be thrown during the provisioning of a product. The three exceptions that can be expected are:
1. `DuplicateResourceException`: This exception is thrown when a duplicate resource has been specified.
2. `InvalidParametersException`: This exception is thrown when the provided parameters are invalid.
3. `ResourceNotFoundException`: This exception is thrown when the resource cannot be found.

For more information on provisioned products, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/servicecatalog/latest/dg/API_ProvisionProduct.html).

# Token Rotation

## Updating Token Rotation Frequency
To enhance security, the Terraform Cloud team token associated with your account is automatically rotated every 30 days. However, the frequency in which the token rotation occurs is customizable. There are two ways in which the token rotation frequency can be updated:
1. Update the token rotation frequency within the Terraform configuration itself. This can be done by updating the `[aws_cloudwatch_event_rule` resource](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/blob/main/token_rotation.tf#L198) within the TFC-RE and running `terraform apply` to apply these changes.
2. Update the token rotation frequency within the AWS EventBridge UI. To do this, navigate to “rules” and select [`TerraformEngineRotateToken`](https://us-west-2.console.aws.amazon.com/events/home?region=us-west-2#/eventbus/default/rules/TerraformEngineRotateToken). From here, click “[Edit](https://us-west-2.console.aws.amazon.com/events/home?region=us-west-2#/eventbus/default/rules/TerraformEngineRotateToken/edit),” and then navigate to “Define schedule.” It is within “Define schedule” where you can manually update the token rotation frequency. While we recommend that tokens are rotated every 30 days, Admins can update this interval to whatever frequency they’d prefer.

## Token Rotation Monitoring
To monitor token rotation, an AWS Admin can navigate to AWS EventBridge and view the [`TerraformEngineRotateToken`](https://us-west-2.console.aws.amazon.com/events/home?region=us-west-2#/eventbus/default/rules/TerraformEngineRotateToken) rules through the “Monitoring” tab.

# Troubleshooting

### Exceptions
There are three types of exceptions that can be thrown by the TFC-RE. The three exceptions that can be expected are:
1. `ParserInvalidParameterException`: This exception is thrown when an invalid input is provided to the parser.
2. `ParserAccessDeniedException`: This exception is thrown when the parser is passed a launch role that it cannot assume. Alternatively, this exception can be thrown when the launch role cannot access the artifact that is passed to the parser.
3. `NoFilesToParseExceptionMessage`: This exception is thrown when the parser is unable to find a `.tf` file to parse, or when the Terraform configuration does not contain a `.tf` file for the `root` module.

# Monitoring

## Monitoring with AWS

### AWS CloudWatch
For more insight on provisioned products and general Service Catalog metrics, CloudWatch can provide further insights through the use of dashboards and monitors. For more information on CloudWatch, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/WhatIsCloudWatch.html).

### AWS X-Ray
AWS X-Ray provides insight and traces that can help identify and debug issues. For more insight on things like Lambda Functions and any alerts associated with them, X-Ray can be a helpful resource. For more information on X-Ray, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/xray/latest/devguide/aws-xray.html).

### AWS Step Functions
The AWS Step Functions service provides further insight into each of the four Step Functions that the TFC-RE uses:`ServiceCatalogTFCTokenRotationStateMachine`, `ServiceCatalogTFCProvisionOperationStateMachine`, `ServiceCatalogTFCUpdateOperationStateMachine`, and `ServiceCatalogTFCTerminateOperationStateMachine`. Clicking into a particular Step Function can provide insight into all State Machine executions and allows you to click into particular executions for more insight into the State Machine’s polling mechanisms. The polling view can also provide more informative errors and has a direct link to the CloudWatch logs associated with the execution. For more information on AWS Step Functions, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/step-functions/latest/dg/welcome.html).

### AWS Lambda
For more insight into a particular Lambda function, leveraging the AWS Lambda service can be helpful. This service can provide information regarding the Lambda’s code source, code properties, and runtime settings, for example. For more information on AWS Lambda Functions, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/lambda/latest/dg/welcome.html).

### Amazon SQS
The Amazon SQS contains information regarding Service Catalog workloads. The Amazon SQS service can provide information regarding Lambda triggers, tagging, access policies, and more for a given queue. It also contains the dead-letter queue. Each queue has its own “message retention period,” which is the duration that the messages will be kept. The message retention period is configurable, and is set to 4 days by default for the TFC-RE. For more information on Amazon SQS, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html), and for more information on setting queue attributes, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_SetQueueAttributes.html).

## Monitoring with TFC

### Run Outputs
There are a few places where you can monitor your organization’s TFC runs and workspaces, but one of the easiest places to monitor them is under the “Runs” tab for a particular workspace. The “Runs” tab will contain each run for a particular workspace. Additionally, you can click into a workspace run from this view, allowing you to gain further insight into the state of the run and where and why it errored. This view also contains the raw log for a given run and its sentinel mocks. For more information on TFC runs, please refer to this [documentation](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run).

# Uninstalling the Integration
To uninstall the integration, you should first destroy any necessary information in AWS and then run the `terraform destroy` command. This will remove the integration.

# Limitations

## Artifact File Types
Provisioning artifacts must be a tar file, in the `.tar.gz` format, with a filename extension of `.tar.gz`. Additionally, all Terraform configuration files must be a Terraform file, in the `.tf` format, with a filename extension of `.tf`. Currently, these two file types are the only ones excepted, and providing anything different as a provisioning artifact can result in failures to complete the provisioning process.

### Troubleshooting Artifact Files
If you run into issues with the artifact files, ensure that the filename extensions are correct—tar files must be in the `.tar.gz` format and Terraform files must be in the `.tf` format. If the filename extensions are not the root cause, next ensure that the `.tar.gz` file is located in the `root` directory.

## Parameter Parser

### Parsing Large Artifacts
AWS Lambdas have a memory size constraint. This limitation can lead to issues when attempting to parse large provisioning artifacts, namely artifacts that are over 500 KB.

## Resource Timeouts
If the provisioning step takes too long, the AWS Service Catalog will timeout. This can also cause the Terraform to timeout, as it has a 30-minute timeout limit. To resolve this timeout issue, try to rerun the provisioning step, or try re-`apply`ing the Terraform.

# Common Error States and How to Resolve Them

## State Machine Timeout
**Error:** `A lambda function invoked by the state machine has timed out`

**Cause:** This error occurs when a Lambda function times out.

**Solution:** To resolve this error, try rerunning the operation.

## `403` Forbidden
**Error:** `403`

**Cause:** This error typically occurs when a role has an issue assuming another role.

**Solution:** To resolve this error, ensure that the appropriate roles have been given and re-`apply` the Terraform.

## Issues with the Service Catalog Product Version
**Error:**

**Cause:** This error occurs when the AWS Service Catalog Product version is out of date.

**Solution:** To resolve this error, create a new version of the product and re-provision.

It is important to create a new product version anytime the configuration has been modified. In doing this, you should be able to avoid this error.

## Error Creating Team
**Error:** `Error: Error creating team aws-service-catalog for organization <org-name>: resource not found`

**Cause:** This error occurs when a `TFE_TOKEN` has not been set.

**Solution:** To resolve this error, set the `TFE_TOKEN` environment variable.

For more information on authentication tokens, please refer to this [documentation](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens).

# Contributing

## Pull Requests
All pull requests require at least one approval from the [CODEOWNERS](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/blob/main/.github/CODEOWNERS). Before merge, all pull request checks must pass, including Go tests.

## Bug Reports
To file a bug report or to provide feedback on the TFC-RE, please [open a GitHub issue](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues). Please try to provide as much detail as possible so that we are able to help you as quickly as possible.

# Attributions
1. [Terraform](https://github.com/hashicorp/terraform)
2. [Terraform Registry](https://registry.terraform.io/providers/hashicorp/aws/latest)
3. [AWS Service Catalog Engine for Terraform Open Source](https://github.com/aws-samples/service-catalog-engine-for-terraform-os)
4. [AWS Service Catalog Documentation](https://docs.aws.amazon.com/servicecatalog/index.html)

--------------------
## License
[Mozilla Public License v2.0](https://github.com/hashicorp/terraform/blob/main/LICENSE)
