# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

data "aws_iam_openid_connect_provider" "tfc_provider" {
  count = var.provision_oidc_provider ? 0 : 1
  url   = "https://${var.tfc_hostname}"
}

# If provision_oidc_provider is true (default), the module will create an OIDC provider in AWS using the TLS certificate from the Terraform Cloud hostname. If provision_oidc_provider is false, the module will use the ARN provided in tfc_provider_arn.
locals {
  oidc_provider_arn = var.provision_oidc_provider ? aws_iam_openid_connect_provider.tfc_provider[0].arn : data.aws_iam_openid_connect_provider.tfc_provider[0].arn
}

# Data source used to grab the TLS certificate for Terraform Cloud:
# https://registry.terraform.io/providers/hashicorp/tls/latest/docs/data-sources/certificate
data "tls_certificate" "tfc_certificate" {
  count = var.provision_oidc_provider ? 1 : 0
  url   = "https://${var.tfc_hostname}"
}

# Creates an OIDC provider which is restricted to:
# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_openid_connect_provider
resource "aws_iam_openid_connect_provider" "tfc_provider" {
  count           = var.provision_oidc_provider ? 1 : 0
  url             = data.tls_certificate.tfc_certificate[0].url
  client_id_list  = [var.tfc_aws_audience]
  thumbprint_list = [data.tls_certificate.tfc_certificate[0].certificates[0].sha1_fingerprint]
}
