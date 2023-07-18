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
    sid = "tfeTokenRotation"

    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
      "secretsmanager:DescribeSecret",
      "secretsmanager:PutSecretValue",
      "secretsmanager:UpdateSecretVersionStage"
    ]

    resources = ["${aws_secretsmanager_secret.team_token_values.arn}*"]
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
  source_file = "lambda-functions/golang/token-rotation/main"
}

# Lambda for rotating team tokens
resource "aws_lambda_function" "rotate_token_handler" {
  filename      = data.archive_file.rotate_token_handler.output_path
  function_name = "TerraformCloudEngineRotateTokenHandlerLambda"
  role          = aws_iam_role.rotate_token_handler_lambda_execution.arn
  handler       = "main"

  source_code_hash = data.archive_file.rotate_token_handler.output_base64sha256

  runtime = "go1.x"

  environment {
    variables = {
      PROVISIONING_STATE_MACHINE_ARN = aws_sfn_state_machine.provision_state_machine.arn,
      UPDATING_STATE_MACHINE_ARN     = aws_sfn_state_machine.update_state_machine.arn,
      TERMINATING_STATE_MACHINE_ARN  = aws_sfn_state_machine.terminate_state_machine.arn,
      PROVISIONING_FUNCTION_NAME     = aws_lambda_function.provision_handler.function_name,
      UPDATING_FUNCTION_NAME         = aws_lambda_function.update_handler.function_name,
      TERMINATING_FUNCTION_NAME      = aws_lambda_function.terminate_handler.function_name,
      TEAM_ID                        = tfe_team.provisioning_team.id,
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

    actions = ["lambda:InvokeFunction", "lambda:ListEventSourceMappings", "lambda:UpdateEventSourceMapping"]

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

# Resources for rotating the team token every 30 days
resource "aws_cloudwatch_event_rule" "rotate_token_schedule" {
  name                = "TerraformEngineRotateToken"
  description         = "Schedule for Token Rotation"
  schedule_expression = "rate(30 days)"
}

resource "aws_cloudwatch_event_target" "token_rotation" {
  rule     = aws_cloudwatch_event_rule.rotate_token_schedule.name
  arn      = aws_sfn_state_machine.rotate_token_state_machine.id
  role_arn = aws_iam_role.token_rotation_event_role.arn
}

resource "aws_iam_role" "token_rotation_event_role" {
  name               = "ServiceCatalogEngineForTerraformCloudTokenRotation"
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

resource "aws_iam_role_policy" "token_rotation_event_role_policy" {
  name = "ServiceCatalogEngineForTerraformCloudRotationEventPolicy"
  role = aws_iam_role.token_rotation_event_role.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "states:StartExecution"
      ],
      "Resource": [
        "${aws_sfn_state_machine.rotate_token_state_machine.arn}"
      ]
    }
  ]
}
EOF
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
        "Next": "Resume SQS processing"
    },
    "Resume SQS processing": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.rotate_token_handler.arn}",
      "Parameters": {
        "operation": "RESUMING"
      },
        "Next": "Wait for all state machine executions to resume"
    },
    "Wait for all state machine executions to resume": {
      "Type": "Wait",
      "Seconds": 1,
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
