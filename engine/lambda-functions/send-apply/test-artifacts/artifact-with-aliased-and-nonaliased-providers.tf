terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

provider "aws" {
  alias = "eu"
}

resource "aws_instance" "example" {
  ami           = "ami-0742b4e673072066f"
  instance_type = "t3.micro"
}

resource "aws_instance" "west_example" {
  provider      = aws.west
  ami           = "ami-0abcdef1234567890"
  instance_type = "t3.micro"
}