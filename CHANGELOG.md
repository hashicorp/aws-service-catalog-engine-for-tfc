# v1.0.5 (08/31/2026)

## Improvements
* Lambda functions now use the newer `provided.al2023` runtime. The `provided.al2` was deprecated.

# v1.0.4 (02/19/2026)

## Improvements
* Added logic to conditionally provision the AWS OIDC Provider based on a new variable, `provision_oidc_provider`. This variable defaults to `true`. If it is `false`, it will instead attempt to lookup an existing AWS OIDC Provider for `app.terraform.io` and use that in the engine. This change allows users who may already have an existing OIDC provider in AWS to provision this Service Catalog Engine without collisions. By @dominic-retli-hashi in https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/pull/35
* Region is now extracted from the AWS provider in terraform config files rather than defaulting to overriding the region based on where the engine is deployed. This allows users to provision products in the region of their choosing. By @dominic-retli-hashi in https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/pull/36


# v1.0.4 (02/22/2024)

## Improvements
* Added logic to conditionally provision the AWS OIDC Provider based on a new variable, `provision_oidc_provider`. This variable defaults to `true`. If it is `false`, it will instead attempt to lookup an existing AWS OIDC Provider for `app.terraform.io` and use that in the engine. This change allows users who may already have an existing OIDC provider in AWS to provision this Service Catalog Engine without collisions. By @dominic-retli-hashi in https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/pull/35
* Region is now extracted from the AWS provider in terraform config files rather than defaulting to overriding the region based on where the engine is deployed. This allows users to provision products in the region of their choosing. By @dominic-retli-hashi in https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/pull/36

# v1.0.3 (02/22/2024)

## Bug Fixes
* [#23](https://github.com/hashicorp/aws-service-catalog-engine-for-tfc/issues/23) thanks [@yukikgiants](https://github.com/yukikgiants)! - Fix invalid path error when trying to use engine directory as a standalone module.

# v1.0.2 (11/28/2023)

## Improvements
* Lambda functions now use the newer `provided.al2` runtime and `arm64` architecture. The `go1.x` was [deprecated](https://aws.amazon.com/blogs/compute/migrating-aws-lambda-functions-from-the-go1-x-runtime-to-the-custom-runtime-on-amazon-linux-2/), and will no longer receive security updates after December 31st, 2023. 

# v1.0.1 (10/09/2023)

## Bug Fixes
* Runs in Terraform Cloud that produce multiple State Versions (due to a bug or outdated version of Terraform Enterprise) will no longer cause provisioning to fail. The Engine now fetches and inspects each State Version to find the most recent version, and uses that version to parse the `output` values.

# v1.0.0 (07/31/2023)

Initial release
