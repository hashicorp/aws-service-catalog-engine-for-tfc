variable "tfe_hostname" {
  type = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
  default = "app.terraform.io"
}

variable "tfe_organization" {
  type = string
  description = "Name of the organization to manage infrastructure with in TFC"
}

variable "tfe_team" {
  type = string
  description = "Name of the TFC team to use to provision infrastructure with in TFC"
}

variable "tfc_aws_audience" {
  type        = string
  default     = "aws.workload.identity"
  description = "The audience value to use in run identity tokens"
}