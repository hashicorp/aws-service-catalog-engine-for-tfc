# Copyright IBM Corp. 2023, 2025
# SPDX-License-Identifier: MPL-2.0

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
  name               = "ServiceCatalogTerraformCloudRotateTokenHandlerRole${var.name_suffix}"
  assume_role_policy = data.aws_iam_policy_document.rotate_token_handler.json
}

resource "aws_iam_role_policy" "rotate_token_handler_lambda_execution_role_policy" {
  name   = "ServiceCatalogTerraformCloudRotateTokenHandlerPolicy"
  role   = aws_iam_role.rotate_token_handler_lambda_execution.id
  policy = data.aws_iam_policy_document.policy_for_rotate_team_token_handler.json
}

data "aws_iam_policy_document" "policy_for_rotate_team_token_handler" {
  version = "2012-10-17"

  statement {
    sid = "tfeTokenRotation"

    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
      "secretsmanager:DescribeSecret",
      "secretsmanager:PutSecretValue",
      "secretsmanager:UpdateSecretVersionStage",
      "secretsmanager:UpdateSecret"
    ]

    resources = ["${aws_secretsmanager_secret.team_token_values.arn}*"]
  }

  statement {
    sid = "pauseStateMachines"

    effect = "Allow"

    actions = [
      "lambda:UpdateEventSourceMapping"
    ]

    condition {
      test     = "StringLike"
      variable = "lambda:FunctionArn"
      values = [
        aws_lambda_function.provision_handler.arn,
        aws_lambda_function.update_handler.arn,
        aws_lambda_function.terminate_handler.arn
      ]
    }

    resources = ["*"]
  }

  statement {
    sid = "listEventSourceMappings"

    effect = "Allow"

    actions = [
      "lambda:ListEventSourceMappings"
    ]

    resources = ["*"]
  }

  statement {
    sid = "pollStateMachines"

    effect = "Allow"

    actions = [
      "states:ListExecutions"
    ]

    resources = [
      aws_sfn_state_machine.provision_state_machine.arn,
      aws_sfn_state_machine.update_state_machine.arn,
      aws_sfn_state_machine.terminate_state_machine.arn
    ]
  }
}


resource "aws_iam_role_policy_attachment" "rotate_token_handler_lambda_execution" {
  for_each   = toset(["arn:aws:iam::aws:policy/AWSXrayWriteOnlyAccess", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
  role       = aws_iam_role.rotate_token_handler_lambda_execution.name
  policy_arn = each.value
}

data "archive_file" "rotate_token_handler" {
  type        = "zip"
  output_path = "dist/token_rotation_handler.zip"
  source_file = "${path.module}/lambda-functions/token-rotation/bootstrap"
}

# Lambda for rotating team tokens
resource "aws_lambda_function" "rotate_token_handler" {
  count         = var.create_tfc_team ? 1 : 0
  filename      = data.archive_file.rotate_token_handler.output_path
  function_name = "ServiceCatalogTerraformCloudRotateTokenHandler${var.name_suffix}"
  role          = aws_iam_role.rotate_token_handler_lambda_execution.arn
  handler       = "bootstrap"

  source_code_hash = data.archive_file.rotate_token_handler.output_base64sha256

  runtime       = "provided.al2"
  architectures = ["arm64"]

  environment {
    variables = {
      PROVISIONING_STATE_MACHINE_ARN = aws_sfn_state_machine.provision_state_machine.arn,
      UPDATING_STATE_MACHINE_ARN     = aws_sfn_state_machine.update_state_machine.arn,
      TERMINATING_STATE_MACHINE_ARN  = aws_sfn_state_machine.terminate_state_machine.arn,
      PROVISIONING_FUNCTION_NAME     = aws_lambda_function.provision_handler.function_name,
      UPDATING_FUNCTION_NAME         = aws_lambda_function.update_handler.function_name,
      TERMINATING_FUNCTION_NAME      = aws_lambda_function.terminate_handler.function_name,
      TEAM_ID                        = tfe_team.provisioning_team[0].id,
      TFE_CREDENTIALS_SECRET_ID      = aws_secretsmanager_secret.team_token_values.arn
    }
  }
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
  name               = "SCTFCTokenRotationStateMachineRole${var.name_suffix}"
  assume_role_policy = data.aws_iam_policy_document.rotate_team_token.json
}

resource "aws_iam_role_policy" "rotate_team_token_state_machine_role_policy" {
  count  = var.create_tfc_team ? 1 : 0
  name   = "SCTFCTokenRotationStateMachinePolicy${var.name_suffix}"
  role   = aws_iam_role.rotate_token_state_machine.id
  policy = data.aws_iam_policy_document.policy_for_rotate_team_token_state_machine[0].json
}


data "aws_iam_policy_document" "policy_for_rotate_team_token_state_machine" {
  count   = var.create_tfc_team ? 1 : 0
  version = "2012-10-17"

  statement {
    sid = "LambdaInvocationPermissions"

    effect = "Allow"

    actions = ["lambda:InvokeFunction"]

    resources = [aws_lambda_function.rotate_token_handler[0].arn]
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

# Resources for rotating the team token every 30 days
resource "aws_cloudwatch_event_rule" "rotate_token_schedule" {
  count               = var.create_tfc_team ? 1 : 0
  name                = "ServiceCatalogTerraformCloudRotateToken${var.name_suffix}"
  description         = "Schedule for Token Rotation"
  schedule_expression = "rate(${var.token_rotation_interval_in_days} days)"
}

resource "aws_cloudwatch_event_target" "token_rotation" {
  count    = var.create_tfc_team ? 1 : 0
  rule     = aws_cloudwatch_event_rule.rotate_token_schedule[0].name
  arn      = aws_sfn_state_machine.rotate_token_state_machine[0].id
  role_arn = aws_iam_role.token_rotation_event_role.arn
}

resource "aws_iam_role" "token_rotation_event_role" {
  name               = "ServiceCatalogTerraformCloudTokenRotationEventRole${var.name_suffix}"
  assume_role_policy = data.aws_iam_policy_document.token_rotation_event_role_policy_document.json
}
data "aws_iam_policy_document" "token_rotation_event_role_policy_document" {
  statement {
    actions = [
      "sts:AssumeRole"
    ]

    principals {
      type = "Service"
      identifiers = [
        "events.amazonaws.com"
      ]
    }
  }
}

resource "aws_iam_role_policy" "token_rotation_state_machine_event_role_policy" {
  count = var.create_tfc_team ? 1 : 0
  name  = "ServiceCatalogTerraformCloudTokenRotationEventPolicy${var.name_suffix}"
  role  = aws_iam_role.token_rotation_event_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "states:StartExecution"
        ]
        Resource = [
          aws_sfn_state_machine.rotate_token_state_machine[0].arn
        ]
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "rotate_token_state_machine" {
  count = var.create_tfc_team ? 1 : 0
  name  = "ServiceCatalogTerraformCloudTokenRotationStateMachine${var.name_suffix}"
}

resource "aws_sfn_state_machine" "rotate_token_state_machine" {
  count    = var.create_tfc_team ? 1 : 0
  name     = "ServiceCatalogTerraformCloudTokenRotationStateMachine${var.name_suffix}"
  role_arn = aws_iam_role.rotate_token_state_machine.arn
  logging_configuration {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.rotate_token_state_machine[0].arn}:*"
  }

  tracing_configuration {
    enabled = true
  }

  definition = jsonencode({
    Comment = "A state machine that manages the team token rotation experience.",
    StartAt = "Pause SQS processing",
    States = {
      "Pause SQS processing" = {
        Type = "Task",
        Resource = aws_lambda_function.rotate_token_handler[0].arn,
        Parameters = {
          operation = "PAUSING"
        },
        Next = "Wait for all state machine executions to finish"
      },
      "Wait for all state machine executions to finish" = {
        Type    = "Wait",
        Seconds = 10,
        Next    = "Poll state machine executions"
      },
      "Poll state machine executions" = {
        Type     = "Task",
        Resource = aws_lambda_function.rotate_token_handler[0].arn,
        Parameters = {
          operation = "POLLING"
        },
        ResultPath = "$.pollStateMachinesResult",
        Retry = [
          {
            ErrorEquals = [
              "Lambda.ServiceException",
              "Lambda.AWSLambdaException",
              "Lambda.SdkClientException"
            ],
            IntervalSeconds = 2,
            MaxAttempts     = 6,
            BackoffRate     = 2
          }
        ],
        Next = "Are there any outstanding state machine executions?"
      },
      "Are there any outstanding state machine executions?" = {
        Type    = "Choice",
        Comment = "Looks-up the current status of the command invocation and delegates accordingly to handle it",
        Choices = [
          {
            And = [
              {
                Variable      = "$.pollStateMachinesResult.stateMachineExecutionCount",
                NumericEquals = 0
              },
              {
                Variable     = "$.pollStateMachinesResult.eventSourceMappingStatus",
                StringEquals = "Disabled"
              }
            ],
            Next = "Rotate team token"
          }
        ],
        Default = "Wait for all state machine executions to finish"
      },
      "Rotate team token" = {
        Type     = "Task",
        Resource = aws_lambda_function.rotate_token_handler[0].arn,
        Parameters = {
          operation = "ROTATING"
        },
        Retry = [
          {
            ErrorEquals = [
              "Lambda.ServiceException",
              "Lambda.AWSLambdaException",
              "Lambda.SdkClientException"
            ],
            IntervalSeconds = 2,
            MaxAttempts     = 6,
            BackoffRate     = 2
          }
        ],
        Next = "Resume SQS processing"
      },
      "Resume SQS processing" = {
        Type     = "Task",
        Resource = aws_lambda_function.rotate_token_handler[0].arn,
        Parameters = {
          operation = "RESUMING"
        },
        Retry = [
          {
            ErrorEquals = [
              "Lambda.ServiceException",
              "Lambda.AWSLambdaException",
              "Lambda.SdkClientException"
            ],
            IntervalSeconds = 2,
            MaxAttempts     = 6,
            BackoffRate     = 2
          }
        ],
        End = true
      }
    }
  })
}
