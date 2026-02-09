# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.12.0"
    }
  }
}


variable "region" {
  description = "AWS region for this deployment"
  type        = string
}

variable "tfc_organization" {
  type        = string
  description = "Name of the organization to manage infrastructure with in TFC"
}

variable "tfc_team" {
  type        = string
  description = "Name of the TFC team to use to provision infrastructure with in TFC"
}

variable "tfc_hostname" {
  type        = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
}

variable "tfc_aws_audience" {
  type        = string
  description = "The audience value to use in run identity tokens"
}

variable "cloudwatch_log_retention_in_days" {
  type        = number
  description = "Number of days you wish retain Cloudwatch logs for all the AWS resources in this configuration"
}

variable "enable_xray_tracing" {
  type        = bool
  description = "When set to true, AWS X-Ray tracing is enabled"
}

variable "token_rotation_interval_in_days" {
  type        = number
  description = "Interval for automatic rotation of the Terraform Cloud API Token"
}

variable "terraform_version" {
  type        = string
  description = "Version of Terraform Core to use in Terraform Cloud for all Service Catalog products"
}

variable "portfolio_name" {
  type        = string
  description = "Name for the Service Catalog portfolio. If not provided, defaults to 'TFC Example Portfolio - {region}'"
  default     = null
}

variable "name_suffix" {
  type        = string
  description = "Optional suffix to append to resource names for uniqueness across regions (e.g., '-us-east-1')"
  default     = ""
}

variable "create_oidc_provider" {
  type        = bool
  description = "Whether to create the OIDC provider. Set to false for additional regions to avoid duplicates."
  default     = true
}

variable "create_tfc_team" {
  type        = bool
  description = "Whether to create the TFC team and token. Set to false for additional regions to avoid duplicates."
  default     = true
}

variable "oidc_provider_arn" {
  type        = string
  description = "ARN of existing OIDC provider to use when create_oidc_provider is false"
  default     = ""
}

variable "create_tag_options" {
  type        = bool
  description = "Whether to create Service Catalog tag options. Set to false for additional regions to avoid duplicates."
  default     = true
}

# This module provisions the Terraform Cloud Reference Engine
module "terraform_cloud_reference_engine" {
  source = "../engine"

  tfc_organization                 = var.tfc_organization
  tfc_team                         = var.tfc_team
  tfc_aws_audience                 = var.tfc_aws_audience
  tfc_hostname                     = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing              = var.enable_xray_tracing
  token_rotation_interval_in_days  = var.token_rotation_interval_in_days
  terraform_version                = var.terraform_version

  # Multi-region support
  name_suffix          = var.name_suffix
  create_oidc_provider = var.create_oidc_provider
  create_tfc_team      = var.create_tfc_team
  oidc_provider_arn    = var.oidc_provider_arn
}

# Creates an AWS Service Catalog Portfolio
resource "aws_servicecatalog_portfolio" "portfolio" {
  name          = var.portfolio_name != null ? var.portfolio_name : "TFC Example Portfolio - ${var.region}"
  description   = "Example Portfolio created via AWS Service Catalog Engine for TFC"
  provider_name = "HashiCorp Examples"
}

# Example product
module "example_product" {
  source = "../example-product"

  # ARNs of Lambda functions that need to be able to assume the IAM Launch Role
  parameter_parser_role_arn  = module.terraform_cloud_reference_engine.parameter_parser_role_arn
  send_apply_lambda_role_arn = module.terraform_cloud_reference_engine.send_apply_lambda_role_arn

  # AWS Service Catalog portfolio you would like to add this product to
  service_catalog_portfolio_ids = [aws_servicecatalog_portfolio.portfolio.id]

  # Variables for authentication to AWS via Dynamic Credentials
  tfc_hostname     = module.terraform_cloud_reference_engine.tfc_hostname
  tfc_organization = module.terraform_cloud_reference_engine.tfc_organization
  tfc_provider_arn = module.terraform_cloud_reference_engine.oidc_provider_arn

  # Multi-region support
  create_tag_options = var.create_tag_options
}

# Output key values that might be useful
output "oidc_provider_arn" {
  description = "ARN of the OIDC provider for this region"
  value       = module.terraform_cloud_reference_engine.oidc_provider_arn
}

output "portfolio_id" {
  description = "ID of the Service Catalog portfolio for this region"
  value       = aws_servicecatalog_portfolio.portfolio.id
}

output "tfc_hostname" {
  description = "TFC hostname"
  value       = module.terraform_cloud_reference_engine.tfc_hostname
}

output "tfc_organization" {
  description = "TFC organization"
  value       = module.terraform_cloud_reference_engine.tfc_organization
}

output "parameter_parser_role_arn" {
  description = "ARN of the parameter parser role"
  value       = module.terraform_cloud_reference_engine.parameter_parser_role_arn
}

output "send_apply_lambda_role_arn" {
  description = "ARN of the send apply lambda role"
  value       = module.terraform_cloud_reference_engine.send_apply_lambda_role_arn
}

