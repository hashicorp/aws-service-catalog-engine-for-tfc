# AWS Service Catalog Engine for Terraform Cloud
The AWS Service Catalog Engine for Terraform Cloud (TFC-RE) is an integration between AWS Service Catalog and Terraform Cloud that allows users to provision Service Catalog products using TFC. This integration gives administrators governance and visibility into their Terraform workloads, and allows Service Catalog administrators to delegate cloud resource provisioning responsibilities to users within their organizations.

## Getting Started

### Prerequisites
- A Terraform Cloud organization that supports [Team Management](https://www.hashicorp.com/products/terraform/pricing).

### Provision the Engine
Everything you need to get started using the Terraform Cloud engine is included in this project's terraform configuration.

1. Authenticate with both AWS and Terraform Cloud:
   - Authenticate the AWS provider using one of the methods listed in the [AWS provider documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration).
   - Authenticate the TFE Terraform Provider using one of the methods listed in the [TFE Terraform documentation](https://registry.terraform.io/providers/hashicorp/tfe/0.11.2/docs#authentication). It is important to note that the user/token you use will need permissions to create Teams and other authentication tokens.
     For more information on TFC permissions, please refer to this [documentation](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/permissions).
2. Copy `terraform.tfvars.example` to `terraform.tfvars` and set the following values:
   - `tfc_organization` to the name of your Terraform Cloud organization.
   - `tfc_team` the name of the team that this configuration will create to manage this integration. This team's API Token will be used by Service Catalog to authenticate API calls to Terraform Cloud.
3. Run `terraform plan` to verify your setup, and then run `terraform apply` to apply your changes.

### Test the Engine
Once you've applied the configuration, you should see a newly created AWS Service Catalog portfolio in [your AWS Service Catalog dashboard](https://console.aws.amazon.com/servicecatalog/home).

To test your newly provisioned Service Catalog Engine for Terraform Cloud, follow [the guide to granting access to portfolios](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/catalogs_portfolios_users.html). Navigate to the newly provisioned `"TFC Example Portfolio"` and grant access to a user of your choosing. Instruct the newly assigned "test user" to attempt to provision the included example product that this engine creates (it is already assigned to the `"TFC Example Portfolio"`).

The example product mentioned above can be found [here](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/tree/main/example-product).

## Creating and Provisioning a Product in Service Catalog
The TFC-RE creates an example product upon launch, however, if you’d prefer to create a new product using the AWS Service Catalog UI, please refer to AWS's developer documentation, which can be found [here](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/getstarted-terraform-engine-cloud.html).

## Token Rotation

### Updating Token Rotation Frequency
The Terraform Cloud team token associated with your account is automatically rotated every 30 days. However, the frequency in which the token rotation occurs can be overridden via the `token_rotation_interval_in_days` variable, which can be found [here](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/blob/main/variables.tf#L39).

## Terraform Version

### Updating the Terraform Version
The Terraform version can be set to a version of your choice by updating the `terraform_version` variable, which can be found [here](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/blob/main/engine/variables.tf#L45).
We recommend that you use version 1.5.4 or higher.

### Reset Terraform Cloud Token
If the API token in Secrets Manager becomes invalid for any reason, you can forcefully regenerate and reinstall a fresh API Token into Secrets Manager using the following script:

```bash
# Destroy the current team token
terraform destroy --target=module.terraform_cloud_reference_engine.tfe_team_token.test_team_token

# Re-apply the terraform to re-create the team token (the Secrets Manager secret will be updated as well)
terraform apply
```

## Troubleshooting

### Terraform Authentication
If you run into TFC workspace issues, such as issues when creating TFC workspaces, it may mean that the [Team](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/teams) that has been created to launch products for your AWS Service Catalog account may not have the correct set of permissions on Terraform Cloud.

**Solution:** Re-apply the engine's Terraform to reset the Team's permissions (thus re-granting it permissions to create and manage workspaces within your organization).

### Service Catalog Product Version
If you run into AWS Service Catalog product issues, such as issues when provisioning a new product, it may mean that the product version needs to be updated.

**Solution:** Create a new product version. It is important to note that anytime the configuration has been modified, the product version will need to be updated.

### Hub and Spoke Permission Requirements
If you see that provisioning is failing in a specific spoke account, it may mean that the engine in the hub account hasn't been allowed to assume the launch role assigned to that product in the spoke account.

**Solution:** Allow the `SendApplyRole` and `ParameterParser` IAM roles to assume the launch role of the product in the spoke account by modifying the launch role's IAM trust relationship policy, as shown below:

```hcl
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "GivePermissionsToServiceCatalog",
            "Effect": "Allow",
            "Principal": {
                "Service": "servicecatalog.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        },
        {
            "Sid": "GivePermissionsToTerraformCloudReferenceEngine",
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::012345678901:root"
            },
            "Action": "sts:AssumeRole",
            "Condition": {
                "StringLike": {
                    "aws:PrincipalArn": [
                        "arn:aws:iam::012345678901:role/ServiceCatalogEngineForTerraformCloudSendApplyRole",
                        "arn:aws:iam::012345678901:role/ServiceCatalogTerraformCloudParameterParserRole"
                    ]
                }
            }
        },
        {
            "Sid": "AllowDynamicProviderCredentials",
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::012345678901:oidc-provider/app.terraform.io"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "app.terraform.io:aud": "aws.workload.identity"
                },
                "StringLike": {
                    "app.terraform.io:sub": "organization:organization-name:project:*:workspace:*:run_phase:*"
                }
            }
        }
    ]
}
```

### Exceptions
**Error:** `NoFilesToParseExceptionMessage`

**Solution:** This exception is thrown when the parser is unable to find a `.tf` file to parse, or when the Terraform configuration does not contain a `.tf` file for the `root` module. To ensure that your file contains `.tf` files at the root level, try recreating the file using the commands in the Terraform Cloud API-driven workflow guide [here](https://developer.hashicorp.com/terraform/cloud-docs/run/api#2-create-the-file-for-upload).

### State Machine Timeout
**Error:** `A lambda function invoked by the state machine has timed out`

**Cause:** This error occurs when a Lambda function times out.

**Solution:** To resolve this error, try rerunning the operation. Additionally, please file an issue in the [repository](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues), or contact HashiCorp support.

### Error Creating Team
**Error:** `Error: Error creating team aws-service-catalog for organization <org-name>: resource not found`

**Cause:** This error occurs when a `TFE_TOKEN` has not been set, or the `tfc_organization` variable wasn't provided correctly.

**Solution:** Check that the `tfc_organization` value you provided exactly matches the name of your TFC organization. Also make sure you have set the `TFE_TOKEN` environment variable to a valid [API Token](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens).

For more information on authentication tokens, please refer to this [documentation](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens).

## Monitoring

### Monitoring with AWS

#### AWS CloudWatch
For more insight on provisioned products and general Service Catalog metrics, CloudWatch can provide further insights through the use of dashboards and monitors. For more information on CloudWatch, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/WhatIsCloudWatch.html).

#### AWS X-Ray
AWS X-Ray provides insight and traces that can help identify and debug issues. For more insight on things like Lambda Functions and any alerts associated with them, X-Ray can be a helpful resource. For more information on X-Ray, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/xray/latest/devguide/aws-xray.html).

#### AWS Step Functions
The AWS Step Functions service provides further insight into each of the four Step Functions that the TFC-RE uses:`ServiceCatalogTerraformCloudTokenRotationStateMachine`, `ServiceCatalogTerraformCloudProvisionOperationStateMachine`, `ServiceCatalogTerraformCloudUpdateOperationStateMachine`, and `ServiceCatalogTerraformCloudTerminateOperationStateMachine`. Clicking into a particular Step Function can provide insight into all State Machine executions and allows you to click into particular executions for more insight into the State Machine’s polling mechanisms. The polling view can also provide more informative errors and has a direct link to the CloudWatch logs associated with the execution. For more information on AWS Step Functions, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/step-functions/latest/dg/welcome.html).

#### AWS Lambda
For more insight into a particular Lambda function, leveraging the AWS Lambda service can be helpful. This service can provide information regarding the Lambda’s code source, code properties, and runtime settings, for example. For more information on AWS Lambda Functions, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/lambda/latest/dg/welcome.html).

#### Amazon SQS
The Amazon SQS contains information regarding Service Catalog workloads. The Amazon SQS service can provide information regarding Lambda triggers, tagging, access policies, and more for a given queue. It also contains the dead-letter queue. Each queue has its own “message retention period,” which is the duration that the messages will be kept. The message retention period is configurable, and is set to 4 days by default for the TFC-RE. For more information on Amazon SQS, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html), and for more information on setting queue attributes, please refer to this AWS developer [documentation](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_SetQueueAttributes.html).

### Monitoring with TFC

#### Run Outputs
There are a few places where you can monitor your organization’s TFC runs and workspaces, but one of the easiest places to monitor them is under the “Runs” tab for a particular workspace. The “Runs” tab will contain each run for a particular workspace. Additionally, you can click into a workspace run from this view, allowing you to gain further insight into the state of the run and where and why it errored. This view also contains the raw log for a given run and its sentinel mocks. For more information on TFC runs, please refer to this [documentation](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/run).

### Monitoring TFE Token Rotation
To monitor token rotation, an AWS Admin can search for metrics related to the `TerraformEngineRotateToken` event rule in AWS CloudWatch. We recommend that you set up an AWS CloudWatch alarm for token rotation in the event that an error occurs. For more information on CloudWatch alarms, please refer to this  AWS developer [documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/AlarmThatSendsEmail.html).

## Limitations

### Maximum Configuration Version Size
The maximum configuration version size supports files up to 950KB. Files larger than 950KB will result in a failure within AWS Service Catalog.

### Parsing Large Artifacts
AWS Lambdas have a memory size constraint. This limitation can lead to issues when attempting to parse large provisioning artifacts, namely artifacts that are over 500 KB.

### Resource Timeouts
If the provisioning step takes too long, the AWS Service Catalog will timeout. This can also cause the Terraform to timeout, as it has a 30-minute timeout limit. To resolve this timeout issue, try to rerun the provisioning step, or try re-`apply`ing the Terraform.

### Renaming Workspaces
Workspaces created by the engine should not be renamed within TFC. When a provisioned product's workspace is renamed and then updated within AWS Service Catalog, a new workspace will be created for that provisioned product. To avoid conflicts, it is recommended that you do not rename workspaces created by the engine.

### Variable Sets
Unlike variables, variable sets are not automatically purged. This may lead to an issue where a workspace's run will not apply properly because it contains an extraneous variable set. to resolve this, remove the variable set and update the provisioned product within AWS Service Catalog.

## Uninstalling the Integration
To uninstall the integration, you should first destroy any necessary information in AWS. Next, run the `terraform destroy` command. This will remove the integration.

## Contributing
For more information on how to contribute, please refer to the [Contributing Guide](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/blob/main/CONTRIBUTING.md).

## Attributions
1. [Terraform](https://github.com/hashicorp/terraform)
2. [AWS Service Catalog Engine for Terraform Open Source](https://github.com/aws-samples/service-catalog-engine-for-terraform-os)
3. [AWS Service Catalog Documentation](https://docs.aws.amazon.com/servicecatalog/index.html)
