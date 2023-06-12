# AWS Service Catalog Engine for Terraform Cloud

This repository contains everything you need to install a AWS Service Catalog Engine for Terraform Cloud into your AWS account. It provides you with:
1. Pre-configured AWS Resources that Service Catalog can use to manage Products in AWS via Terraform Cloud
2. An AWS IAM OIDC Provider that Terraform Cloud can use to authenticate securely and automatically with AWS using Dynamic Credentials
3. [An example AWS Service Catalog Product](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/tree/main/example-product) that you can use as a template for your own Terraform configurations

## Getting Started

#### Launch the Engine

You'll need the Terraform CLI installed, and you'll need to set the following environment variables in your local shell:

1. `TFE_TOKEN`: a Terraform Cloud user token with permission to create workspaces within your organization.

You'll also need to authenticate the AWS provider as you would normally using one of the methods mentioned in the AWS provider documentation [here](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#authentication-and-configuration).

Copy `terraform.tfvars.example` to `terraform.tfvars` and set the organization name to your TFC organization's name.

Run `terraform plan` to verify your setup, and then run `terraform apply`.

#### Configure the Portfolio

Once you've applied the configuration, you should see an Example 