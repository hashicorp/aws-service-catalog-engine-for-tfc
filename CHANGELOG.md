# v1.0.2 (10/09/2023)

## Improvements
* Lambda functions now use the newer `provided.al2` runtime and `arm64` architecture. The `go1.x` was [deprecated](https://aws.amazon.com/blogs/compute/migrating-aws-lambda-functions-from-the-go1-x-runtime-to-the-custom-runtime-on-amazon-linux-2/), and will no longer receive security updates after December 31st, 2023. 

# v1.0.1 (10/09/2023)

## Bug Fixes
* Runs in Terraform Cloud that produce multiple State Versions (due to a bug or outdated version of Terraform Enterprise) will no longer cause provisioning to fail. The Engine now fetches and inspects each State Version to find the most recent version, and uses that version to parse the `output` values.

# v1.0.0 (07/31/2023)

Initial release