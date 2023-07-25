# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "4.63.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "3.5.1"
    }
    tfe = {
      source  = "hashicorp/tfe"
      version = "0.45.0"
    }
  }
}

provider "aws" {
  # Configuration options
  region = "us-west-2"

  default_tags {
    tags = {
      "Projects" = "aws-service-catalog-engine"
    }
  }
}

provider "tfe" {
  hostname = var.tfc_hostname
}

# This module provisions the Terraform Cloud Reference Engine
module "terraform_cloud_reference_engine" {
  source = "./engine"

  tfc_organization = var.tfc_organization
  tfc_team = var.tfc_team
  tfc_aws_audience = var.tfc_aws_audience
  tfc_hostname = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing = var.enable_xray_tracing
}

# Creates an AWS Service Catalog Portfolio to house the example product
resource "aws_servicecatalog_portfolio" "portfolio" {
  name          = "TFC Example Portfolio"
  description   = "Example Portfolio created via AWS Service Catalog Engine for TFC"
  provider_name = "HashiCorp Examples"
}

resource "random_string" "random" {
  length  = 16
  special = false
  lower   = true
  upper   = false
}

resource "aws_s3_bucket" "my_bucket" {
  bucket = "service-catalog-example-product-${random_string.random.result}"
}

resource "aws_s3_object" "object" {
  bucket = aws_s3_bucket.my_bucket.id
  key    = "product.tar.gz"
  source = "${path.module}/example-product/product.tar.gz"
  etag   = filemd5("${path.module}/example-product/product.tar.gz")
}

module "example_product" {
  source = "./service-catalog-product"

  # Variables for the product name and configuration files (most likely, you'll want to modify these after you've tested things out)
  product_name         = "service-catalog-example-product-${random_string.random.result}"
  artifact_bucket_name = aws_s3_object.object.bucket
  artifact_object_key  = aws_s3_object.object.id

  # ARNs of Lambda functions that need to be able to assume the IAM Launch Role
  parameter_parser_role_arn  = module.terraform_cloud_reference_engine.parameter_parser_role_arn
  send_apply_lambda_role_arn = module.terraform_cloud_reference_engine.send_apply_lambda_role_arn

  # AWS Service Catalog portfolio you would like to add this product to
  service_catalog_portfolio_ids = [aws_servicecatalog_portfolio.portfolio.id]

  # Variables for authentication to AWS via Dynamic Credentials
  tfc_hostname     = module.terraform_cloud_reference_engine.tfc_hostname
  tfc_organization = module.terraform_cloud_reference_engine.tfc_organization
  tfc_provider_arn = module.terraform_cloud_reference_engine.oidc_provider_arn

}