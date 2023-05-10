data "aws_iam_policy_document" "update_product_state_machine_assumed_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "update_state_machine" {
  name               = "ServiceCatalogTFCUpdateOperationStateMachineRole"
  assume_role_policy = data.aws_iam_policy_document.update_product_state_machine_assumed_policy.json
}

resource "aws_iam_role_policy" "update_state_machine" {
  name   = "ServiceCatalogTFCUpdateOperationStateMachineRolePolicy"
  role   = aws_iam_role.update_state_machine.id
  policy = data.aws_iam_policy_document.update_state_machine.json
}

data "aws_iam_policy_document" "update_state_machine" {
  version = "2012-10-17"

  statement {
    sid = "LambdaInvocationPermissions"

    effect = "Allow"

    actions = ["lambda:InvokeFunction"]

    resources = [aws_lambda_function.send_apply_command_function.arn, aws_lambda_function.poll_run_status.arn, aws_lambda_function.notify_run_result.arn, aws_lambda_function.parameter_parser.arn]

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

resource "aws_cloudwatch_log_group" "update_state_machine" {
  name = "ServiceCatalogTFCUpdateOperationStateMachine"
}

resource "aws_sfn_state_machine" "update_state_machine" {
  name     = "ServiceCatalogTFCUpdateOperationStateMachine"
  role_arn = aws_iam_role.update_state_machine.arn
  logging_configuration {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.update_state_machine.arn}:*"
  }

  tracing_configuration {
    enabled = true
  }

  definition = <<EOF
{
  "Comment": "A state machine that manages the updating experience.",
  "StartAt": "Generate tracer tag",
  "States": {
    "Generate tracer tag": {
      "Type": "Pass",
      "Comment": "Adds a tag to be passed to Terraform default-tags which traces the AWS resources created by it",
      "Parameters": {
        "key": "SERVICE_CATALOG_TERRAFORM_INTEGRATION-DO_NOT_DELETE",
        "value.$": "$.provisionedProductId"
      },
      "ResultPath": "$.tracerTag",
      "Next": "Send apply"
    },
    "Send apply": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.send_apply_command_function.arn}",
      "Parameters": {
        "awsAccountId.$": "$.identity.awsAccountId",
        "terraformOrganization.$": "$.terraformOrganization",
        "provisionedProductId.$": "$.provisionedProductId",
        "artifact.$": "$.artifact",
        "launchRoleArn.$": "$.launchRoleArn"
      },
      "ResultSelector": {
        "terraformRunId.$": "$.terraformRunId"
      },
      "ResultPath": "$.sendApplyResult",
      "Catch": [
          {
              "ErrorEquals": [ "States.TaskFailed" ],
              "ResultPath": "$.errorInfo",
              "Next": "Notify update result failure"
          }
      ],
      "Next": "Wait for update to complete"
    },
    "Wait for update to complete": {
      "Type": "Wait",
      "Seconds": 1,
      "Next": "Poll update status"
    },
    "Poll update status": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.poll_run_status.arn}",
      "Parameters": {
        "terraformRunId.$": "$.sendApplyResult.terraformRunId"
      },
      "ResultPath": "$.pollRunResult",
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
          "Next": "Notify update result failure"
        },
        {
          "ErrorEquals": [ "States.Timeout" ],
          "ResultPath": "$.errorInfo",
          "Next": "Notify update result failure"
        }
      ],
      "Next": "Did the update complete successfully?"
    },
    "Did the update complete successfully?": {
      "Type": "Choice",
      "Comment": "Looks-up the current status of the command invocation and delegates accordingly to handle it",
      "Choices": [
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "inProgress",
          "Next": "Wait for update to complete"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "failed",
          "Next": "Convert poll update status error"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "success",
          "Next": "Notify update result"
        }
      ],
      "Default": "Failure"
    },
    "Convert poll update status error": {
        "Type": "Pass",
        "Comment": "Restructures error from the poll update status task to a format the notify run result task understands",
        "Parameters": {
            "Error": "Error applying run in TFC",
            "Cause.$": "$.pollRunStatus.errorMessage",
            "isWrapperError": true
        },
        "ResultPath": "$.errorInfo",
        "Next": "Notify update result failure"
    },
    "Notify update result": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.notify_run_result.arn}",
      "Parameters": {
        "terraformRunId.$": "$.sendApplyResult.terraformRunId",
        "workflowToken.$": "$.token",
        "recordId.$": "$.recordId",
        "tracerTag.$": "$.tracerTag",
        "serviceCatalogOperation": "UPDATING",
        "awsAccountId.$": "$.identity.awsAccountId",
        "terraformOrganization.$": "$.terraformOrganization",
        "provisionedProductId.$": "$.provisionedProductId"
      },
      "End": true
    },
    "Notify update result failure": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.notify_run_result.arn}",
      "Parameters": {
        "terraformRunId.$": "$.sendApplyResult.terraformRunId",
        "workflowToken.$": "$.token",
        "recordId.$": "$.recordId",
        "tracerTag.$": "$.tracerTag",
        "serviceCatalogOperation": "UPDATING",
        "awsAccountId.$": "$.identity.awsAccountId",
        "terraformOrganization.$": "$.terraformOrganization",
        "provisionedProductId.$": "$.provisionedProductId",
        "error.$": "$.errorInfo.Error",
        "errorMessage.$": "$.errorInfo.Cause"
      },
      "End": true
    },
    "Failure": {
      "Type": "Fail"
    }
  }
}
EOF
}
