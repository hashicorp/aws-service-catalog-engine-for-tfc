terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "4.63.0"
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

# # # #
# THE TEMPLATE OF THE PRODUCT

data "aws_s3_object" "artifact" {
  bucket = var.artifact_bucket_name
  key    = var.artifact_object_key
}

# # # #
# THE PRODUCT IN SERVICE CATALOG

resource "aws_servicecatalog_product" "example" {
  name  = var.product_name
  owner = var.service_catalog_product_owner
  type  = "TERRAFORM_OPEN_SOURCE"

  provisioning_artifact_parameters {
    disable_template_validation = true
    template_url                = "https://s3.amazonaws.com/${data.aws_s3_object.artifact.bucket}/${data.aws_s3_object.artifact.key}"
    type                        = "TERRAFORM_OPEN_SOURCE"
  }
}

resource "aws_servicecatalog_product_portfolio_association" "example" {
  for_each = local.unique_portfolio_ids
  portfolio_id = each.value
  product_id   = aws_servicecatalog_product.example.id
}

resource "aws_servicecatalog_constraint" "example" {
  # need to wait a bit after the role is created as service catalog will immediately try to assume the role to test it.
  depends_on = [time_sleep.wait_for_launch_constraint_role_to_be_assumable]

  for_each = local.unique_portfolio_ids
  description  = "Launch constraint for the ${var.product_name} product."
  portfolio_id = each.value
  product_id   = aws_servicecatalog_product.example.id
  type         = "LAUNCH"

  parameters = jsonencode({
    "RoleArn" : aws_iam_role.example_product_launch_role.arn
  })
}

data "aws_iam_openid_connect_provider" "tfc_provider" {
  arn = var.tfc_provider_arn
}

resource "aws_iam_role" "example_product_launch_role" {
  name = "${local.class_case_product_name}LaunchRole"

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
              "${var.parameter_parser_role_arn}*"
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

resource "time_sleep" "wait_for_launch_constraint_role_to_be_assumable" {
  depends_on = [aws_iam_role.example_product_launch_role, aws_iam_role_policy.example_product_launch_constraint_policy]

  create_duration = "15s"
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