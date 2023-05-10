variable "tfc_provider_arn" {
  description = "Arn of the AWS IAM OpenID Connect Provider that establishes trust with TFC"
}

variable "tfc_hostname" {
  type = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
  default = "app.terraform.io"
}

variable "tfc_organization" {
  type = string
  description = "Name of the organization to manage infrastructure with in TFC"
}

variable "product_name" {
  type = string
  description = "Name of the Service Catalog product"
}

variable "artifact_bucket_name" {
  type = string
  description = "Name of bucket where product Terraform configuration is stored"
}

variable "artifact_object_key" {
  type = string
  description = "Path to Terraform configuration within S3 bucket"
}

variable "parameter_parser_role_arn" {
  type = string
  description = "ARN of the IAM Role that the Terraform Parameter Parser Lambda Function uses to parse parameters"
}