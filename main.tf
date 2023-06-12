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
      source = "hashicorp/tfe"
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

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

resource "aws_servicecatalog_portfolio" "portfolio" {
  name          = "TFC Example Portfolio"
  description   = "Example Portfolio created via AWS Service Catalog Engine for TFC"
  provider_name = "Hashicorp Examples"
}

# Products
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
  product_name = "service-catalog-example-product-${random_string.random.result}"
  artifact_bucket_name = aws_s3_object.object.bucket
  artifact_object_key = aws_s3_object.object.id

  # Arn of the parameter parser
  parameter_parser_role_arn = aws_iam_role.parameter_parser.arn

  # AWS Service Catalog portfolio you would like to add this product to
  service_catalog_portfolio_ids = [aws_servicecatalog_portfolio.portfolio.id]

  # Variables for authentication to AWS via Dynamic Credentials
  tfc_hostname = var.tfc_hostname
  tfc_organization = var.tfc_organization
  tfc_provider_arn = aws_iam_openid_connect_provider.tfc_provider.arn
}