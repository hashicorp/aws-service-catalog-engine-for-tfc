data "aws_iam_policy_document" "provision_handler" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "provisioning_handler_lambda_execution" {
  name               = "terraform_engine_provisioning_handler_lambda_execution_role"
  assume_role_policy = data.aws_iam_policy_document.provision_handler.json
}

resource "aws_iam_role_policy" "provision_handler_lambda_execution_role_policy" {
  name   = "provision_handler_lambda_execution_role_policy"
  role   = aws_iam_role.provisioning_handler_lambda_execution.id
  policy = data.aws_iam_policy_document.policy_for_provision_handler.json
}


data "aws_iam_policy_document" "policy_for_provision_handler" {
  version = "2012-10-17"

  statement {
    sid = "AllowSqs"

    effect = "Allow"

    actions = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"]

    resources = [aws_sqs_queue.terraform_engine_provision_operation_queue.arn, aws_sqs_queue.terraform_engine_update_queue.arn]

  }

  statement {
    sid = "AllowKmsDecrypt"

    effect = "Allow"

    actions = ["kms:Decrypt"]

    resources = [aws_kms_key.queue_key.arn]

  }

    statement {
      sid = "AllowStepFunction"

      effect = "Allow"

      actions = ["states:StartExecution"]

      resources = [aws_sfn_state_machine.manage_provisioned_product.arn]

    }
}

resource "aws_iam_role_policy_attachment" "provision_handler_lambda_execution" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.provisioning_handler_lambda_execution.name
  policy_arn = each.value
}

data "archive_file" "provision_handler" {
  type        = "zip"
  output_path = "provisioning_operations_handler.zip"
  source_dir  = "lambda-functions/provisioning-operations-handler"
}

resource "aws_lambda_function" "provision_handler" {
  filename      = data.archive_file.provision_handler.output_path
  function_name = "terraform_engine_provisioning_handler_lambda"
  role          = aws_iam_role.provisioning_handler_lambda_execution.arn
  handler       = "provisioning_operations_handler.handle_sqs_records"

  source_code_hash = data.archive_file.provision_handler.output_base64sha256

  runtime = "python3.9"

  environment {
    variables = {
      STATE_MACHINE_ARN = aws_sfn_state_machine.manage_provisioned_product.arn
    }
  }
}

resource "aws_lambda_event_source_mapping" "provision_handler_provisioning_queue" {
  event_source_arn        = aws_sqs_queue.terraform_engine_provision_operation_queue.arn
  function_name           = aws_lambda_function.provision_handler.arn
  batch_size              = 10
  enabled                 = true
  function_response_types = ["ReportBatchItemFailures"]
}

