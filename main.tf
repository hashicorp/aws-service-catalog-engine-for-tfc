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
  default_tags {
    tags = {
      "Projects" = "aws-service-catalog-engine"
    }
  }
}

provider "tfe" {
  hostname = var.tfc_hostname
}

# This module provisions the Terraform Cloud Reference Engine.
module "terraform_cloud_reference_engine" {
  source = "./engine"

  tfc_organization                 = var.tfc_organization
  tfc_team                         = var.tfc_team
  tfc_aws_audience                 = var.tfc_aws_audience
  tfc_hostname                     = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing              = var.enable_xray_tracing
  token_rotation_interval_in_days  = var.token_rotation_interval_in_days
  terraform_version                = var.terraform_version
}