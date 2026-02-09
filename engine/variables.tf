# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

variable "tfc_organization" {
  type        = string
  description = "Name of the organization to manage infrastructure with in TFC"
}

variable "tfc_team" {
  type        = string
  description = "Name of the TFC team to use to provision infrastructure with in TFC"
  default     = "aws-service-catalog"
}

variable "enable_xray_tracing" {
  type        = bool
  description = "When set to true, AWS X-Ray tracing is enabled"
  default     = true
}

variable "tfc_hostname" {
  type        = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
  default     = "app.terraform.io"
}

variable "tfc_aws_audience" {
  type        = string
  default     = "aws.workload.identity"
  description = "The audience value to use in run identity tokens"
}

variable "cloudwatch_log_retention_in_days" {
  type        = number
  default     = 90
  description = "Number of days you wish retain Cloudwatch logs for all the AWS resources in this configuration. These logs are invaluable for Terraform Cloud support staff in helping to diagnose any issues you may run into!"
}

variable "token_rotation_interval_in_days" {
  type        = number
  default     = 30
  description = "Interval for automatic rotation of the Terraform Cloud API Token that Service Catalog uses to authenticate with Terraform Cloud. Default is 30 days"
}

variable "terraform_version" {
  type        = string
  default     = "1.5.4"
  description = "Version of Terraform Core to use in Terraform Cloud for all Service Catalog products"
}

variable "name_suffix" {
  type        = string
  default     = ""
  description = "Optional suffix to append to resource names for uniqueness across regions (e.g., '-us-east-1')"
}

variable "create_oidc_provider" {
  type        = bool
  default     = true
  description = "Whether to create the OIDC provider. Set to false for additional regions to avoid duplicates."
}

variable "create_tfc_team" {
  type        = bool
  default     = true
  description = "Whether to create the TFC team and token. Set to false for additional regions to avoid duplicates."
}

variable "oidc_provider_arn" {
  type        = string
  default     = ""
  description = "ARN of existing OIDC provider to use when create_oidc_provider is false"
}
