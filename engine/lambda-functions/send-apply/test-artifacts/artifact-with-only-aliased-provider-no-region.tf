terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  alias = "west"
}

resource "aws_instance" "west_example" {
  provider      = aws.west
  ami           = "ami-0abcdef1234567890"
  instance_type = "t3.micro"
}