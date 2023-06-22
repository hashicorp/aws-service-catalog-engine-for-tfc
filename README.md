# AWS Service Catalog Engine for Terraform Cloud

This repository contains everything you need to install an AWS Service Catalog Engine for Terraform Cloud into your AWS account. It provides you with:
1. Pre-configured AWS Resources that enable AWS Service Catalog to manage products in AWS via Terraform Cloud
2. An AWS IAM OIDC Provider that Terraform Cloud can use to authenticate securely and automatically with AWS using [Dynamic Credentials](https://developer.hashicorp.com/terraform/tutorials/cloud/dynamic-credentials)
3. [An example AWS Service Catalog Product](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/tree/main/example-product) that you can use as a template for your own Terraform configurations

## Getting Started

#### Launch the Engine

You'll need the Terraform CLI installed, and you'll need to set the `TFE_TOKEN` environment variable to a Terraform Cloud user token with permission to create workspaces within your organization.

You'll also need to authenticate the AWS provider as you would normally, using one of the methods mentioned in the AWS provider documentation [here](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration).

1. Copy `terraform.tfvars.example` to `terraform.tfvars` and set the organization name to your TFC organization's name.
1. Run `terraform plan` to verify your setup, and then run `terraform apply`.

#### Test the Engine

Once you've applied the configuration, you should see a newly created AWS Service Catalog portfolio in [your AWS Service Catalog dashboard](https://console.aws.amazon.com/servicecatalog/home). 

To test your newly provisioned Service Catalog Engine for Terraform Cloud, follow [the guide to granting access to portfolios](https://docs.aws.amazon.com/servicecatalog/latest/adminguide/catalogs_portfolios_users.html). Navigate to the newly provisioned `"TFC Example Portfolio"` and grant access to a user of your choosing. Instruct the newly assigned "test user" to attempt to provision the included example product that this engine creates (it is already assigned to the `"TFC Example Portfolio"`).    
