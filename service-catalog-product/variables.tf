variable "service_catalog_portfolio_ids" {
  type        = list(string)
  description = "ID of the AWS Service Catalog Portfolios to assign the product to"
}

locals {
  unique_portfolio_ids = { for index, portfolio_id in var.service_catalog_portfolio_ids : index => portfolio_id }
}

variable "service_catalog_product_owner" {
  type        = string
  description = "Name of the owner of the AWS Service Catalog Product"
  default     = "Service Catalog Admin"
}

variable "tfc_provider_arn" {
  type        = string
  description = "Arn of the AWS IAM OpenID Connect Provider that establishes trust with TFC"
}

variable "tfc_hostname" {
  type        = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
  default     = "app.terraform.io"
}

variable "tfc_organization" {
  type        = string
  description = "Name of the organization to manage infrastructure with in TFC"
}

variable "product_name" {
  type        = string
  description = "Name of the Service Catalog product"
}

locals {
  _product_name_convert_snake_case_to_class_case = join("", [for word in split("_", var.product_name) : title(word)])
  _product_name_convert_kebab_case_to_class_case = join("", [for word in split("-", local._product_name_convert_snake_case_to_class_case) : title(word)])
  class_case_product_name                        = local._product_name_convert_kebab_case_to_class_case
}

variable "artifact_bucket_name" {
  type        = string
  description = "Name of bucket where product Terraform configuration is stored"
}

variable "artifact_object_key" {
  type        = string
  description = "Path to Terraform configuration within S3 bucket"
}

variable "parameter_parser_role_arn" {
  type        = string
  description = "ARN of the IAM Role that the Terraform Parameter Parser Lambda Function uses to parse parameters"
}

variable "send_apply_lambda_role_arn" {
  type        = string
  description = "ARN of the IAM Role that the Send Apply Lambda Function uses to trigger applies in Terraform Cloud"
}