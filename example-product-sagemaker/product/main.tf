terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }

    random = {
      source  = "hashicorp/random"
      version = "3.5.1"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      "created-by" = "HCP Terraform via AWS Service Catalog"
    }
  }
}

resource "random_id" "suffix" {
  byte_length = 4
}

locals {
  suffix = lower(random_id.suffix.hex)
}

resource "aws_sagemaker_notebook_instance" "datasci" {
  name          = "analytics-team-${local.suffix}"
  instance_type = "ml.t2.medium"
  role_arn      = aws_iam_role.sagemaker.arn
  volume_size   = var.volume_size
}

resource "aws_s3_bucket" "data" {
  bucket        = "analytics-team-sagemaker-data-${local.suffix}"
  force_destroy = true
}

resource "aws_s3_bucket_versioning" "data" {
  bucket = aws_s3_bucket.data.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "data" {
  bucket = aws_s3_bucket.data.bucket

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "data" {
  bucket                  = aws_s3_bucket.data.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

data "aws_iam_policy_document" "sagemaker_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["sagemaker.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "sagemaker_s3_access" {
  statement {
    actions   = ["s3:*"]
    resources = [aws_s3_bucket.data.arn, "${aws_s3_bucket.data.arn}/"]
  }
}

resource "aws_iam_role" "sagemaker" {
  name               = "analytics-team-${local.suffix}"
  assume_role_policy = data.aws_iam_policy_document.sagemaker_assume_role.json

  inline_policy {
    name   = "analytics-team-sagemaker-s3-access"
    policy = data.aws_iam_policy_document.sagemaker_s3_access.json
  }
}
