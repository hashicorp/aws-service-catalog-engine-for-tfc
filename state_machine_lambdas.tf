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

  environment {
    variables = {
      TFE_TOKEN = var.tfe_token
    }
  }

  source_code_hash = data.archive_file.send_apply.output_base64sha256

  runtime = "go1.x"
}
