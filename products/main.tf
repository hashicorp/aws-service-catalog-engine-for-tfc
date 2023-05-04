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

data "aws_caller_identity" "current" {}

provider "aws" {
  # Configuration options
  region = "us-west-2"

  default_tags {
    tags = {
      "projects" = "aws-service-catalog-engine"
    }
  }
}

resource "random_string" "random" {
  length  = 16
  special = false
  lower   = true
  upper   = false
}

# # # #
# THE TEMPLATE OF THE PRODUCT

resource "aws_s3_bucket" "my_bucket" {
  bucket = "service-catalog-example-product-${random_string.random.result}"
}

resource "aws_s3_bucket_object" "object" {
  bucket = aws_s3_bucket.my_bucket.bucket
  key    = "product.tar.gz"
  source = "${path.module}/product.tar.gz"
  etag   = filemd5("${path.module}/product.tar.gz")
}

# # # #
# THE PRODUCT IN SERVICE CATALOG

resource "aws_servicecatalog_portfolio" "portfolio" {
  name          = "Example Portfolio"
  description   = "List of my examples"
  provider_name = "Taylor Swift"
}

resource "aws_servicecatalog_product" "example" {
  name  = "service-catalog-example-product-${random_string.random.result}"
  owner = "Swift"
  type  = "TERRAFORM_OPEN_SOURCE"

  provisioning_artifact_parameters {
    # TODO: re-enable this to test
    disable_template_validation = true
    template_url                = "https://s3.amazonaws.com/${aws_s3_bucket.my_bucket.bucket}/${aws_s3_bucket_object.object.key}"
    type                        = "TERRAFORM_OPEN_SOURCE"
  }
}

resource "aws_servicecatalog_product_portfolio_association" "example" {
  portfolio_id = aws_servicecatalog_portfolio.portfolio.id
  product_id   = aws_servicecatalog_product.example.id
}

resource "aws_servicecatalog_constraint" "example" {
  description  = "Back off, man. I'm a scientist."
  portfolio_id = aws_servicecatalog_portfolio.portfolio.id
  product_id   = aws_servicecatalog_product.example.id
  type         = "LAUNCH"

  parameters = jsonencode({
    "RoleArn" : aws_iam_role.example_product_launch_role.arn,
    "TerraformOrganization" : "tf-rocket-tfcb-test"
  })
}

data "aws_iam_openid_connect_provider" "tfc_provider" {
  arn = var.tfc_provider_arn
}

resource "aws_iam_role" "example_product_launch_role" {
  name = "example_product_launch_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = "AllowServiceCatalogToAssume"
        Principal = {
          Service = "servicecatalog.amazonaws.com"
        }
      },
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Condition = {
          StringLike = {
            "aws:PrincipalArn" = [
              # TODO: Make sure this Role Arn actually points to the TFE Parameter Parser
              "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/ServiceCatalogTerraformTFEParameterParserRole*"
            ]
          }
        }
      },
      {
        Action = "sts:AssumeRoleWithWebIdentity",
        Effect = "Allow",
        Principal = {
          Federated = var.tfc_provider_arn
        },
        Condition = {
          StringEquals = {
            "${var.tfc_hostname}:aud" = one(data.aws_iam_openid_connect_provider.tfc_provider.client_id_list)
          },
          StringLike = {
            // TODO: Make sure to narrow workspace and project values down
            "${var.tfc_hostname}:sub" = "organization:${var.tfc_organization}:project:*:workspace:*:run_phase:*"
          }
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "example_product_launch_constraint_policy" {
  name   = "example_product_launch_constraint_policy"
  role   = aws_iam_role.example_product_launch_role.id
  policy = data.aws_iam_policy_document.example_product_launch_constraint_policy.json
}

data "aws_iam_policy_document" "example_product_launch_constraint_policy" {
  version = "2012-10-17"

  statement {
    sid = "S3Access"

    effect = "Allow"

    actions = [
      "s3:*",
    ]

    resources = ["*"]

  }

  statement {
    sid = "ResourceGroups"

    effect = "Allow"

    actions = [
      "resource-groups:CreateGroup",
      "resource-groups:ListGroupResources",
      "resource-groups:DeleteGroup",
      "resource-groups:Tag"
    ]

    resources = ["*"]
  }

  statement {
    sid = "Tagging"

    effect = "Allow"

    actions = [
      "tag:GetResources",
      "tag:GetTagKeys",
      "tag:GetTagValues",
      "tag:TagResources",
      "tag:UntagResources"
    ]

    resources = ["*"]
  }
}