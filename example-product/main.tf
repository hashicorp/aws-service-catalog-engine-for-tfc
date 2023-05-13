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
  }
}

provider "aws" {
  region = "us-west-2"
}

resource "random_string" "random" {
  length  = 16
  special = false
  upper = false
}

resource "aws_s3_bucket" "my-bucket" {
  bucket = "aws-tfc-service-catalog-example-${random_string.random.result}"
}

output "bucket_name" {
  value = aws_s3_bucket.my-bucket.bucket
}
