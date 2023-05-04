variable "tfc_provider_arn" {
  description = "Arn of the AWS IAM OpenID Connect Provider that establishes trust with TFC"
}

variable "tfc_hostname" {
  type = string
  description = "TFC hostname (defaults to TFC: app.terraform.io)"
  default = "app.terraform.io"
}

variable "tfc_organization" {
  type = string
  description = "Name of the organization to manage infrastructure with in TFC"
}