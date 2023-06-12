# Lambda Policies

data "aws_iam_policy_document" "send_apply" {
  version = "2012-10-17"

  statement {
    sid = "s3Access"

    effect = "Allow"

    actions = ["s3:GetObject"]

    resources = ["*"]

  }

  statement {
    sid = "tfeCredentialsAccess"

    effect = "Allow"

    actions = ["secretsmanager:GetSecretValue"]

    resources = ["*"]
  }
}

data "aws_iam_policy_document" "send_destroy" {
  version = "2012-10-17"

  statement {
    sid = "tfeCredentialsAccess"

    effect = "Allow"

    actions = ["secretsmanager:GetSecretValue"]

    resources = ["*"]
  }
}

data "aws_iam_policy_document" "poll_run_status" {
  version = "2012-10-17"

  statement {
    sid = "tfeCredentialsAccess"

    effect = "Allow"

    actions = ["secretsmanager:GetSecretValue"]

    resources = ["*"]
  }
}

data "aws_iam_policy_document" "notify_run_result" {
  version = "2012-10-17"

  statement {
    sid = "ServiceCatalogAccess"

    effect = "Allow"

    actions = [
      "servicecatalog:NotifyProvisionProductEngineWorkflowResult",
      "servicecatalog:NotifyTerminateProvisionedProductEngineWorkflowResult",
      "servicecatalog:NotifyUpdateProvisionedProductEngineWorkflowResult"
    ]

    resources = ["*"]

  }

  statement {
    sid = "tfeCredentialsAccess"

    effect = "Allow"

    actions = ["secretsmanager:GetSecretValue"]

    resources = ["*"]
  }
}


# Lambda Functions

locals {
  default_lambda_function_timeout = 60
  default_lambda_function_memory_size = 128

  send_apply_lambda_name = "ServiceCatalogEngineForTerraformCloudSendApply"
  send_destroy_lambda_name = "ServiceCatalogEngineForTerraformCloudSendDestroy"
  poll_run_status_lambda_name = "ServiceCatalogEngineForTerraformCloudPollRunStatus"
  notify_run_result_lambda_name = "ServiceCatalogEngineForTerraformCloudNotifyRunResult"

  lambda_functions = {
    (local.send_apply_lambda_name): {
      policy_document = data.aws_iam_policy_document.send_apply.json
      source_file = "lambda-functions/golang/send-apply/main"
      timeout = 120
      # TODO: Play with this memory size and see how the lambda duration changes
      memory_size = 1024
    }
    (local.send_destroy_lambda_name): {
      policy_document = data.aws_iam_policy_document.send_destroy.json
      source_file = "lambda-functions/golang/send-destroy/main"

    }
    (local.poll_run_status_lambda_name): {
      policy_document = data.aws_iam_policy_document.poll_run_status.json
      source_file = "lambda-functions/golang/poll-run-status/main"
      timeout = 30
    }
    (local.notify_run_result_lambda_name): {
      policy_document = data.aws_iam_policy_document.notify_run_result.json
      source_file = "lambda-functions/golang/notify-run-result/main"
    }
  }
}

data "aws_iam_policy_document" "basic_lambda_assume_role_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}


resource "aws_iam_role" "state_machine_lambda" {
  for_each = local.lambda_functions

  name               = "${each.key}Role"
  assume_role_policy = data.aws_iam_policy_document.basic_lambda_assume_role_policy.json
}

resource "aws_iam_role_policy_attachment" "lambda_basic_execution" {
  for_each   = local.lambda_functions

  role       = aws_iam_role.state_machine_lambda[each.key].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_xray_write_only_access" {
  for_each   = local.lambda_functions

  role       = aws_iam_role.state_machine_lambda[each.key].name
  policy_arn = "arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess"
}

resource "aws_iam_role_policy" "state_machine_lambda_policy" {
  for_each = local.lambda_functions

  name               = "${each.key}RolePolicy"
  role       = aws_iam_role.state_machine_lambda[each.key].name
  policy = each.value.policy_document
}

data "archive_file" "state_machine_lambda_executable" {
  for_each = local.lambda_functions

  type        = "zip"
  output_path = "dist/${lower(each.key)}.zip"
  source_file = each.value.source_file
}

# Explicitly create the AWS Cloudwatch Log Group for the Lambda so that we can control the log retention
resource "aws_cloudwatch_log_group" "lambda_cloudwatch_log_group" {
  for_each = local.lambda_functions

  name              = "/aws/lambda/${each.key}"
  retention_in_days = var.cloudwatch_log_retention_in_days
}

resource "aws_lambda_function" "state_machine_lambda" {
  for_each = local.lambda_functions

  function_name = each.key
  filename      = data.archive_file.state_machine_lambda_executable[each.key].output_path
  role          = aws_iam_role.state_machine_lambda[each.key].arn
  handler       = "main"
  timeout = lookup(local.lambda_functions[each.key], "timeout", local.default_lambda_function_timeout)
  memory_size = lookup(local.lambda_functions[each.key], "memory_size", local.default_lambda_function_memory_size)

  environment {
    variables = {
      TFE_CREDENTIALS_SECRET_ID = aws_secretsmanager_secret_version.tfc_credentials.arn
      TFE_CREDENTIALS_SECRET_VERSION_ID = aws_secretsmanager_secret_version.tfc_credentials.version_id
    }
  }

  source_code_hash = data.archive_file.state_machine_lambda_executable[each.key].output_base64sha256

  runtime = "go1.x"

  depends_on = [aws_cloudwatch_log_group.lambda_cloudwatch_log_group]
}


# ARNs for each of the Lambda Functions created in this file (for resources in other files to reference easily)
locals {
  send_apply_lambda_arn = lookup(aws_lambda_function.state_machine_lambda, local.send_apply_lambda_name, {arn: ""}).arn
  send_destroy_lambda_arn = lookup(aws_lambda_function.state_machine_lambda, local.send_destroy_lambda_name, {arn: ""}).arn
  poll_run_status_lambda_arn = lookup(aws_lambda_function.state_machine_lambda, local.poll_run_status_lambda_name, {arn: ""}).arn
  notify_run_result_lambda_arn = lookup(aws_lambda_function.state_machine_lambda, local.notify_run_result_lambda_name, {arn: ""}).arn
}