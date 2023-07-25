# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "oidc_provider_arn" {
  type        = string
  value = aws_iam_openid_connect_provider.tfc_provider.arn
}

output "parameter_parser_role_arn" {
  type        = string
  value  = aws_iam_role.parameter_parser.arn
}

output "send_apply_lambda_role_arn" {
  type        = string
  value  = local.send_apply_lambda_role_arn
}

output "tfc_organization" {
  type        = string
  value  = local.send_apply_lambda_role_arn
}

output "tfc_hostname" {
  type        = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
  value     = "app.terraform.io"
}