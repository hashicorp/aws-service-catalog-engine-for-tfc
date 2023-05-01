variable "tfe_hostname" {
  type = string
  description = "TFC/E hostname (defaults to TFC: app.terraform.io)"
  default = "app.terraform.io"
}

variable "tfe_organization" {
  type = string
  description = "Name of the organization to manage infrastructure with in TFC/E"
}

variable "tfe_team" {
  type = string
  description = "Name of the TFC/E team to use to provision infrastructure with in TFC/E"
}