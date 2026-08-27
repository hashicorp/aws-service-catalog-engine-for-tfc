# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.61.0"
    }
    tfe = {
      source  = "hashicorp/tfe"
      version = "0.45.0"
    }
  }
}

locals {
  # use this variable to update the runtime of the golang lambda functions
  lambda_runtime = "provided.al2023"
}
