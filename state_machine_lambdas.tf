# Send Apply Lambda

data "aws_iam_policy_document" "send_apply" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "send_apply_lambda_execution" {
  name               = "terraform_engine_send_apply_role"
  assume_role_policy = data.aws_iam_policy_document.send_apply.json
}

resource "aws_iam_role_policy_attachment" "send_apply_lambda_execution" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.send_apply_lambda_execution.name
  policy_arn = each.value
}

resource "aws_iam_role_policy" "terraform_engine_send_apply_role" {
  name   = "terraform_engine_send_apply_role_policy"
  role   = aws_iam_role.send_apply_lambda_execution.id
  policy = data.aws_iam_policy_document.policy_for_send_apply_lambda.json
}


data "aws_iam_policy_document" "policy_for_send_apply_lambda" {
  version = "2012-10-17"

  statement {
    sid = "s3Access"

    effect = "Allow"

    actions = ["s3:GetObject"]

    resources = ["*"]

  }
}

data "archive_file" "send_apply" {
  type        = "zip"
  output_path = "send_apply.zip"
  source_file = "lambda-functions/send-apply/main"
}

resource "aws_lambda_function" "send_apply_command_function" {
  filename      = data.archive_file.send_apply.output_path
  function_name = "terraform_engine_send_apply_lambda"
  role          = aws_iam_role.send_apply_lambda_execution.arn
  handler       = "main"
  timeout = 60

  environment {
    variables = {
      TFE_TOKEN = var.tfe_token
    }
  }

  source_code_hash = data.archive_file.send_apply.output_base64sha256

  runtime = "go1.x"
}

# Poll Run Status Lambda

data "aws_iam_policy_document" "poll_run_status" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "poll_run_status" {
  name               = "terraform_engine_poll_run_status_role"
  assume_role_policy = data.aws_iam_policy_document.poll_run_status.json
}

resource "aws_iam_role_policy_attachment" "poll_run_status" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.poll_run_status.name
  policy_arn = each.value
}

data "archive_file" "poll_run_status" {
  type        = "zip"
  output_path = "poll_run_status.zip"
  source_file = "lambda-functions/poll-run-status/main"
}

resource "aws_lambda_function" "poll_run_status" {
  filename      = data.archive_file.poll_run_status.output_path
  function_name = "terraform_engine_poll_run_status_lambda"
  role          = aws_iam_role.poll_run_status.arn
  handler       = "main"
  timeout = 30

  environment {
    variables = {
      TFE_TOKEN = var.tfe_token
    }
  }

  source_code_hash = data.archive_file.poll_run_status.output_base64sha256

  runtime = "go1.x"
}

# Notify Run Result Lambda

data "aws_iam_policy_document" "notify_run_result_assume_role_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "notify_run_result" {
  name               = "terraform_engine_notify_run_result_role"
  assume_role_policy = data.aws_iam_policy_document.notify_run_result_assume_role_policy.json
}

resource "aws_iam_role_policy_attachment" "notify_run_result" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.notify_run_result.name
  policy_arn = each.value
}

resource "aws_iam_role_policy" "notify_run_result" {
  name   = "terraform_engine_notify_run_result_role_policy"
  role   = aws_iam_role.notify_run_result.id
  policy = data.aws_iam_policy_document.notify_run_result.json
}


data "aws_iam_policy_document" "notify_run_result" {
  version = "2012-10-17"

  statement {
    sid = "ServiceCatalogAccess"

    effect = "Allow"

    actions = ["servicecatalog:NotifyProvisionProductEngineWorkflowResult"]

    resources = ["*"]

  }
}

data "archive_file" "notify_run_result" {
  type        = "zip"
  output_path = "notify_run_result.zip"
  source_file = "lambda-functions/notify-run-result/main"
}

resource "aws_lambda_function" "notify_run_result" {
  filename      = data.archive_file.notify_run_result.output_path
  function_name = "terraform_engine_notify_run_result_lambda"
  role          = aws_iam_role.notify_run_result.arn
  handler       = "main"
  timeout = 30

  source_code_hash = data.archive_file.notify_run_result.output_base64sha256

  runtime = "go1.x"
}
