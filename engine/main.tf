# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "4.63.0"
    }
    tfe = {
      source  = "hashicorp/tfe"
      version = "0.45.0"
    }
  }
}
