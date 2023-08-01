# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


data "tfe_organization" "organization" {
  name = var.tfc_organization
}

resource "tfe_team" "provisioning_team" {
  name         = var.tfc_team
  organization = data.tfe_organization.organization.name
  organization_access {
    manage_projects   = true
    manage_workspaces = true
  }
}

resource "tfe_team_token" "test_team_token" {
  team_id = tfe_team.provisioning_team.id
}

resource "aws_secretsmanager_secret" "team_token_values" {
  name = "terraform-cloud-credentials-for-service-catalog-engine"
}

resource "aws_secretsmanager_secret_version" "tfc_credentials" {
  secret_id = aws_secretsmanager_secret.team_token_values.id
  secret_string = jsonencode({
    hostname = var.tfc_hostname
    id       = tfe_team.provisioning_team.id
    token    = tfe_team_token.test_team_token.token
  })
}
