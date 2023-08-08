# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

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
  name               = "ServiceCatalogTerraformCloudUpdateOperationStateMachineRole"
  assume_role_policy = data.aws_iam_policy_document.update_product_state_machine_assumed_policy.json
}

resource "aws_iam_role_policy" "update_state_machine" {
  name   = "ServiceCatalogTerraformCloudUpdateOperationStateMachineRolePolicy"
  role   = aws_iam_role.update_state_machine.id
  policy = data.aws_iam_policy_document.update_state_machine.json
}

data "aws_iam_policy_document" "update_state_machine" {
  version = "2012-10-17"

  statement {
    sid = "LambdaInvocationPermissions"

    effect = "Allow"

    actions = ["lambda:InvokeFunction"]

    resources = [local.send_apply_lambda_arn, local.poll_run_status_lambda_arn, local.notify_run_result_lambda_arn, aws_lambda_function.parameter_parser.arn]

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

  statement {
    sid = "XRayLogging"

    effect = "Allow"

    actions = [
      "xray:PutTraceSegments",
      "xray:PutTelemetryRecords",
      "xray:GetSamplingRules",
      "xray:GetSamplingTargets"
    ]

    resources = ["*"]
  }
}

resource "aws_cloudwatch_log_group" "update_state_machine" {
  name              = "ServiceCatalogTerraformCloudUpdateOperationStateMachine"
  retention_in_days = var.cloudwatch_log_retention_in_days
}

resource "aws_sfn_state_machine" "update_state_machine" {
  name     = "ServiceCatalogTerraformCloudUpdateOperationStateMachine"
  role_arn = aws_iam_role.update_state_machine.arn

  logging_configuration {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.update_state_machine.arn}:*"
  }

  tracing_configuration {
    enabled = var.enable_xray_tracing
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
      "Next": "Default state for apply"
    },
    "Default state for apply": {
      "Type": "Pass",
      "Comment": "Set default values for state so that future steps do not error on missing parameters",
      "Parameters": {
        "terraformRunId": ""
      },
      "ResultPath": "$.sendApplyResult",
      "Next": "Send apply"
    },
    "Send apply": {
      "Type": "Task",
      "Resource": "${local.send_apply_lambda_arn}",
      "Parameters": {
        "awsAccountId.$": "$.identity.awsAccountId",
        "terraformOrganization.$": "$.terraformOrganization",
        "provisionedProductId.$": "$.provisionedProductId",
        "provisioningArtifactId.$": "$.provisioningArtifactId",
        "artifact.$": "$.artifact",
        "launchRoleArn.$": "$.launchRoleArn",
        "productId.$": "$.productId",
        "tracerTag.$": "$.tracerTag",
        "parameters.$": "$.parameters",
        "tags.$": "$.tags"
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
      "Seconds": 10,
      "Next": "Poll update status"
    },
    "Poll update status": {
      "Type": "Task",
      "Resource": "${local.poll_run_status_lambda_arn}",
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
      "Default": "Convert poll update status error"
    },
    "Convert poll update status error": {
        "Type": "Pass",
        "Comment": "Restructures error from the poll update status task to a format the notify run result task understands",
        "Parameters": {
            "Error": "Error applying run in TFC",
            "Cause.$": "$.pollRunResult.errorMessage",
            "isWrapperError": true
        },
        "ResultPath": "$.errorInfo",
        "Next": "Notify update result failure"
    },
    "Notify update result": {
      "Type": "Task",
      "Resource": "${local.notify_run_result_lambda_arn}",
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
      "Catch": [
          {
              "ErrorEquals": [ "States.TaskFailed" ],
              "ResultPath": "$.errorInfo",
              "Next": "Notify update result failure"
          }
      ],
      "End": true
    },
    "Notify update result failure": {
      "Type": "Task",
      "Resource": "${local.notify_run_result_lambda_arn}",
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
    }
  }
}
EOF
}
