# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "oidc_provider_arn" {
  value = aws_iam_openid_connect_provider.tfc_provider.arn
}

output "parameter_parser_role_arn" {
  value = aws_iam_role.parameter_parser.arn
}

output "send_apply_lambda_role_arn" {
  value = local.send_apply_lambda_role_arn
}

output "tfc_organization" {
  value = data.tfe_organization.organization.name
}

output "tfc_hostname" {
  value = var.tfc_hostname
}
