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

    resources = [aws_sqs_queue.terraform_engine_provision_operation_queue.arn, aws_sqs_queue.terraform_engine_update_queue.arn, aws_sqs_queue.terraform_engine_terminate_queue.arn]

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

    resources = [aws_sfn_state_machine.provision_state_machine.arn, aws_sfn_state_machine.update_state_machine.arn, aws_sfn_state_machine.terminate_state_machine.arn]

  }
}

resource "aws_iam_role_policy_attachment" "provision_handler_lambda_execution" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.provisioning_handler_lambda_execution.name
  policy_arn = each.value
}

data "archive_file" "provision_handler" {
  type        = "zip"
  output_path = "dist/provisioning_operations_handler.zip"
  source_file = "lambda-functions/golang/provisioning-operations-handler/main"
}

# Lambda for provisioning products

resource "aws_lambda_function" "provision_handler" {
  filename      = data.archive_file.provision_handler.output_path
  function_name = "TerraformEngineProvisionHandlerLambda"
  role          = aws_iam_role.provisioning_handler_lambda_execution.arn
  handler       = "main"

  source_code_hash = data.archive_file.provision_handler.output_base64sha256

  runtime = "go1.x"

  environment {
    variables = {
      TERRAFORM_ORGANIZATION = var.tfc_organization
      STATE_MACHINE_ARN      = aws_sfn_state_machine.provision_state_machine.arn
    }
  }
}

resource "aws_lambda_event_source_mapping" "provision_handler_provision_queue" {
  event_source_arn        = aws_sqs_queue.terraform_engine_provision_operation_queue.arn
  function_name           = aws_lambda_function.provision_handler.arn
  batch_size              = 10
  enabled                 = true
  function_response_types = ["ReportBatchItemFailures"]
}

# Lambda for terminating products

resource "aws_lambda_function" "terminate_handler" {
  filename      = data.archive_file.provision_handler.output_path
  function_name = "TerraformEngineTerminateHandlerLambda"
  role          = aws_iam_role.provisioning_handler_lambda_execution.arn
  handler       = "main"

  source_code_hash = data.archive_file.provision_handler.output_base64sha256

  runtime = "go1.x"

  environment {
    variables = {
      TERRAFORM_ORGANIZATION = var.tfc_organization
      STATE_MACHINE_ARN      = aws_sfn_state_machine.terminate_state_machine.arn
    }
  }
}

resource "aws_lambda_event_source_mapping" "terminate_handler_terminate_queue" {
  event_source_arn        = aws_sqs_queue.terraform_engine_terminate_queue.arn
  function_name           = aws_lambda_function.terminate_handler.arn
  batch_size              = 10
  enabled                 = true
  function_response_types = ["ReportBatchItemFailures"]
}

# Lambda for updating products

resource "aws_lambda_function" "update_handler" {
  filename      = data.archive_file.provision_handler.output_path
  function_name = "TerraformEngineUpdateHandlerLambda"
  role          = aws_iam_role.provisioning_handler_lambda_execution.arn
  handler       = "main"

  source_code_hash = data.archive_file.provision_handler.output_base64sha256

  runtime = "go1.x"

  environment {
    variables = {
      TERRAFORM_ORGANIZATION = var.tfc_organization
      STATE_MACHINE_ARN      = aws_sfn_state_machine.update_state_machine.arn
    }
  }
}

resource "aws_lambda_event_source_mapping" "update_handler_update_queue" {
  event_source_arn        = aws_sqs_queue.terraform_engine_update_queue.arn
  function_name           = aws_lambda_function.update_handler.arn
  batch_size              = 10
  enabled                 = true
  function_response_types = ["ReportBatchItemFailures"]
}
