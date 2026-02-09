# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.12.0"
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

provider "aws" {
  alias  = "us-east-1"
  region = "us-east-1"
  default_tags {
    tags = {
      "Projects" = "aws-service-catalog-engine"
    }
  }
}

provider "aws" {
  alias  = "us-east-2"
  region = "us-east-2"
  default_tags {
    tags = {
      "Projects" = "aws-service-catalog-engine"
    }
  }
}

provider "aws" {
  alias  = "us-west-1"
  region = "us-west-1"
  default_tags {
    tags = {
      "Projects" = "aws-service-catalog-engine"
    }
  }
}

provider "aws" {
  alias  = "us-west-2"
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

# Deploy to us-east-1 using the terraform-cloud-reference-engine wrapper module
module "terraform_cloud_reference_engine_us_east_1" {
  source = "./terraform-cloud-reference-engine"

  providers = {
    aws = aws.us-east-1
  }

  region                           = "us-east-1"
  portfolio_name                   = "TFC Example Portfolio - us-east-1"
  tfc_organization                 = var.tfc_organization
  tfc_team                         = var.tfc_team
  tfc_aws_audience                 = var.tfc_aws_audience
  tfc_hostname                     = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing              = var.enable_xray_tracing
  token_rotation_interval_in_days  = var.token_rotation_interval_in_days
  terraform_version                = var.terraform_version

  # Multi-region: us-east-1 creates global resources
  name_suffix          = "-us-east-1"
  create_oidc_provider = true
  create_tfc_team      = true
  create_tag_options   = true
  oidc_provider_arn    = ""
}

# Deploy to us-east-2 using the terraform-cloud-reference-engine wrapper module
module "terraform_cloud_reference_engine_us_east_2" {
  source = "./terraform-cloud-reference-engine"

  providers = {
    aws = aws.us-east-2
  }

  region                           = "us-east-2"
  portfolio_name                   = "TFC Example Portfolio - us-east-2"
  tfc_organization                 = var.tfc_organization
  tfc_team                         = var.tfc_team
  tfc_aws_audience                 = var.tfc_aws_audience
  tfc_hostname                     = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing              = var.enable_xray_tracing
  token_rotation_interval_in_days  = var.token_rotation_interval_in_days
  terraform_version                = var.terraform_version

  # Multi-region: us-east-2 does NOT create global resources, references us-east-1
  name_suffix          = "-us-east-2"
  create_oidc_provider = false
  create_tfc_team      = false
  create_tag_options   = false
  oidc_provider_arn    = module.terraform_cloud_reference_engine_us_east_1.oidc_provider_arn
}


# Deploy to us-west-1 using the terraform-cloud-reference-engine wrapper module
module "terraform_cloud_reference_engine_us_west_1" {
  source = "./terraform-cloud-reference-engine"

  providers = {
    aws = aws.us-west-1
  }

  region                           = "us-west-1"
  portfolio_name                   = "TFC Example Portfolio - us-west-1"
  tfc_organization                 = var.tfc_organization
  tfc_team                         = var.tfc_team
  tfc_aws_audience                 = var.tfc_aws_audience
  tfc_hostname                     = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing              = var.enable_xray_tracing
  token_rotation_interval_in_days  = var.token_rotation_interval_in_days
  terraform_version                = var.terraform_version

  name_suffix          = "-us-west-1"
  create_oidc_provider = false
  create_tfc_team      = false
  create_tag_options   = false
  oidc_provider_arn    = module.terraform_cloud_reference_engine_us_east_1.oidc_provider_arn
}


# Deploy to us-west-2 using the terraform-cloud-reference-engine wrapper module
module "terraform_cloud_reference_engine_us_west_2" {
  source = "./terraform-cloud-reference-engine"

  providers = {
    aws = aws.us-west-2
  }

  region                           = "us-west-2"
  portfolio_name                   = "TFC Example Portfolio - us-west-2"
  tfc_organization                 = var.tfc_organization
  tfc_team                         = var.tfc_team
  tfc_aws_audience                 = var.tfc_aws_audience
  tfc_hostname                     = var.tfc_hostname
  cloudwatch_log_retention_in_days = var.cloudwatch_log_retention_in_days
  enable_xray_tracing              = var.enable_xray_tracing
  token_rotation_interval_in_days  = var.token_rotation_interval_in_days
  terraform_version                = var.terraform_version

  name_suffix          = "-us-west-2"
  create_oidc_provider = false
  create_tfc_team      = false
  create_tag_options   = false
  oidc_provider_arn    = module.terraform_cloud_reference_engine_us_east_1.oidc_provider_arn
}
