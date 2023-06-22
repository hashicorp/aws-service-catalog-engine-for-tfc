data "aws_iam_policy_document" "rotate_token_handler" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "rotate_token_handler_lambda_execution" {
  name               = "terraform_engine_rotate_token_handler_lambda_execution_role"
  assume_role_policy = data.aws_iam_policy_document.rotate_token_handler.json
}

resource "aws_iam_role_policy" "rotate_token_handler_lambda_execution_role_policy" {
  name   = "rotate_token_handler_lambda_execution_role_policy"
  role   = aws_iam_role.rotate_token_handler_lambda_execution.id
  policy = data.aws_iam_policy_document.policy_for_rotate_team_token_handler.json
}

data "aws_iam_policy_document" "policy_for_rotate_team_token_handler" {
  version = "2012-10-17"

  statement {
    sid = "AllowStepFunction"

    effect = "Allow"

    actions = ["states:StartExecution"]

    resources = [aws_sfn_state_machine.provision_state_machine.arn, aws_sfn_state_machine.update_state_machine.arn, aws_sfn_state_machine.terminate_state_machine.arn]

  }

  statement {
    sid = "tfeCredentialsAccess"

    effect = "Allow"

    actions = ["secretsmanager:GetSecretValue"]

    resources = ["*"]
  }
}

resource "aws_iam_role_policy_attachment" "rotate_token_handler_lambda_execution" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.rotate_token_handler_lambda_execution.name
  policy_arn = each.value
}

data "archive_file" "rotate_token_handler" {
  type        = "zip"
  output_path = "dist/rotate_token_handler.zip"
  source_dir  = "lambda-functions/golang/rotate_token_handler/main"
}

# Lambda for rotating team tokens

resource "aws_lambda_function" "rotate_token_handler" {
  filename      = data.archive_file.rotate_token_handler.output_path
  function_name = "TerraformEngineRotateTokenHandlerLambda"
  role          = aws_iam_role.rotate_token_handler_lambda_execution.arn
  handler       = "main"

  source_code_hash = data.archive_file.rotate_token_handler.output_base64sha256

  runtime = "go1.x"

  environment {
    variables = {
      PROVISIONING_STATE_MACHINE_ARN = aws_sfn_state_machine.provision_state_machine.arn,
      UPDATING_STATE_MACHINE_ARN = aws_sfn_state_machine.update_state_machine.arn,
      TERMINATING_STATE_MACHINE_ARN = aws_sfn_state_machine.terminate_state_machine.arn,
      TEAM_ID = tfe_team.provisioning_team.id
    }
  }
}

resource "aws_lambda_event_source_mapping" "rotate_token_handler_provision_queue" {
  event_source_arn        = aws_sqs_queue.terraform_engine_provision_operation_queue.arn
  function_name           = aws_lambda_function.rotate_token_handler.arn
  batch_size              = 10
  enabled                 = true
  function_response_types = ["ReportBatchItemFailures"]
}

data "aws_iam_policy_document" "rotate_team_token" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "rotate_token_state_machine" {
  name               = "ServiceCatalogTFCTokenRotationStateMachineRole"
  assume_role_policy = data.aws_iam_policy_document.rotate_team_token.json
}

resource "aws_iam_role_policy" "rotate_team_token_role_policy" {
  name   = "ServiceCatalogTFCTokenRotationStateMachineRolePolicy"
  role   = aws_iam_role.rotate_token_state_machine.id
  policy = data.aws_iam_policy_document.policy_for_rotate_team_token.json
}


data "aws_iam_policy_document" "policy_for_rotate_team_token" {
  version = "2012-10-17"

  statement {
    sid = "LambdaInvocationPermissions"

    effect = "Allow"

    actions = ["lambda:InvokeFunction"]

    resources = [aws_lambda_function.rotate_token_handler.arn]

  }

  statement {
    sid = "CloudwatchPermissions"

    effect = "Allow"

    actions = [
      "logs:CreateLogDelivery",
      "logs:GetLogDelivery",
      "logs:UpdateLogDelivery",
      "logs:DeleteLogDelivery",
      "logs:ListLogDeliveries",
      "logs:PutLogEvents",
      "logs:PutResourcePolicy",
      "logs:DescribeResourcePolicies",
      "logs:DescribeLogGroups"
    ]

    resources = ["*"]

  }
}

resource "aws_cloudwatch_log_group" "rotate_token_state_machine" {
  name = "ServiceCatalogTFCTokenRotationStateMachine"
}

resource "aws_sfn_state_machine" "rotate_token_state_machine" {
  name     = "ServiceCatalogTFCTokenRotationStateMachine"
  role_arn = aws_iam_role.rotate_token_state_machine.arn
  logging_configuration {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.rotate_token_state_machine.arn}:*"
  }

  tracing_configuration {
    enabled = true
  }

  definition = <<EOF
{
  "Comment": "A state machine that manages the team token rotation experience.",
  "StartAt": "Pause SQS processing",
  "States": {
    "Pause SQS processing": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.rotate_token_handler.arn}",
      "Parameters": {
        "operation": "PAUSING"
      },
      "Next": "Wait for all state machine executions to finish"
    },
    "Wait for all state machine executions to finish": {
      "Type": "Wait",
      "Seconds": 1,
      "Next": "Poll state machine executions"
    },
    "Poll state machine executions": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.rotate_token_handler.arn}",
      "Parameters": {
        "operation": "POLLING"
      },
      "ResultPath": "$.pollStateMachinesResult",
      "Retry": [
        {
          "ErrorEquals": [
            "Lambda.ServiceException",
            "Lambda.AWSLambdaException",
            "Lambda.SdkClientException"
          ],
          "IntervalSeconds": 2,
          "MaxAttempts": 6,
          "BackoffRate": 2
        }
      ],
      "Catch": [
        {
          "ErrorEquals": [ "States.TaskFailed" ],
          "ResultPath": "$.errorInfo",
          "Next": "Notify team token rotation failure"
        },
        {
          "ErrorEquals": [ "States.Timeout" ],
          "ResultPath": "$.errorInfo",
          "Next": "Notify team token rotation failure"
        }
      ],
      "Next": "Are there any outstanding state machine executions?"
    },
    "Are there any outstanding state machine executions?": {
      "Type": "Choice",
      "Comment": "Looks-up the current status of the command invocation and delegates accordingly to handle it",
      "Choices": [
        {
          "Variable": "$.pollStateMachinesResult.stateMachinesExecutionCount",
          "NumericEquals": 0,
          "Next": "Rotate team token"
        }
      ],
      "Default": "Are there any outstanding state machine executions?"
    },
    "Rotate team token": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.rotate_token_handler.arn}",
      "Parameters": {
        "operation": "ROTATING"
      },
      "End": true
    },
    "Notify team token rotation failure": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.rotate_token_handler.arn}",
      "Parameters": {
        "operation": "ERRORING"
      },
      "End": true
    }
  }
}
EOF
}
