terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {}

resource "aws_instance" "example" {
  ami           = "ami-0742b4e673072066f"
  instance_type = "t3.micro"
}
