data "aws_iam_policy_document" "manage_provisioned_product" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "tfc_manage_provisioned_product" {
  name               = "tfc_manage_provisioned_product_state_machine"
  assume_role_policy = data.aws_iam_policy_document.manage_provisioned_product.json
}

resource "aws_iam_role_policy" "manage_provisioned_product_role_policy" {
  name   = "tfc_manage_provisioned_product_role_policy"
  role   = aws_iam_role.tfc_manage_provisioned_product.id
  policy = data.aws_iam_policy_document.policy_for_manage_provisioned_product.json
}


data "aws_iam_policy_document" "policy_for_manage_provisioned_product" {
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

resource "aws_cloudwatch_log_group" "tfc_manage_provisioned_product" {
  name = "tfc_manage_provisioned_product_state_machine"
}

resource "aws_sfn_state_machine" "manage_provisioned_product" {
  name     = "tfc_manage_provisioned_product"
  role_arn = aws_iam_role.tfc_manage_provisioned_product.arn
  logging_configuration {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.tfc_manage_provisioned_product.arn}:*"
  }

  tracing_configuration {
    enabled = true
  }

  definition = <<EOF
{
  "Comment": "A state machine that manages the provisioning experience.",
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
      "Next": "Wait for apply to complete"
    },
    "Wait for apply to complete": {
      "Type": "Wait",
      "Seconds": 1,
      "Next": "Poll run status"
    },
    "Poll run status": {
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
          "Next": "Failure"
        },
        {
          "ErrorEquals": [ "States.Timeout" ],
          "ResultPath": "$.errorInfo",
          "Next": "Failure"
        }
      ],
      "Next": "Did the run complete successfully?"
    },
    "Did the run complete successfully?": {
      "Type": "Choice",
      "Comment": "Looks-up the current status of the command invocation and delegates accordingly to handle it",
      "Choices": [
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "inProgress",
          "Next": "Wait for apply to complete"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "failed",
          "Next": "Failure"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "awaitingDecision",
          "Next": "Failure"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "success",
          "Next": "Notify run result"
        }
      ],
      "Default": "Failure"
    },
    "Notify run result": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.notify_run_result.arn}",
      "Parameters": {
        "workflowToken.$": "$.token",
        "recordId.$": "$.recordId",
        "tracerTag.$": "$.tracerTag",
        "serviceCatalogOperation": "PROVISIONING"
      },
      "End": true
    },
    "Failure": {
      "Type": "Fail",
      "Cause": "boop",
      "Error": "womp womp"
    }
  }
}
EOF
}
