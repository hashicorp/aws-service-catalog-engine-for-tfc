data "aws_iam_policy_document" "terminate_product_state_machine_assumed_policy" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "terminate_state_machine" {
  name               = "ServiceCatalogTFCTerminateOperationStateMachineRole"
  assume_role_policy = data.aws_iam_policy_document.terminate_product_state_machine_assumed_policy.json
}

resource "aws_iam_role_policy" "terminate_state_machine" {
  name   = "ServiceCatalogTFCTerminateOperationStateMachineRolePolicy"
  role   = aws_iam_role.terminate_state_machine.id
  policy = data.aws_iam_policy_document.terminate_state_machine.json
}


data "aws_iam_policy_document" "terminate_state_machine" {
  version = "2012-10-17"

  statement {
    sid = "LambdaInvocationPermissions"

    effect = "Allow"

    actions = ["lambda:InvokeFunction"]

    resources = [aws_lambda_function.send_destroy.arn, aws_lambda_function.poll_run_status.arn, aws_lambda_function.notify_run_result.arn, aws_lambda_function.parameter_parser.arn]

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

resource "aws_cloudwatch_log_group" "terminate_state_machine" {
  name = "ServiceCatalogTFCTerminateOperationStateMachine"
}

resource "aws_sfn_state_machine" "terminate_state_machine" {
  name     = "ServiceCatalogTFCTerminateOperationStateMachine"
  role_arn = aws_iam_role.terminate_state_machine.arn
  logging_configuration {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.terminate_state_machine.arn}:*"
  }

  tracing_configuration {
    enabled = true
  }

  definition = <<EOF
{
  "Comment": "A state machine that terminates a provisioned product",
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
      "Next": "Send destroy"
    },
    "Send destroy": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.send_destroy.arn}",
      "Parameters": {
        "awsAccountId.$": "$.identity.awsAccountId",
        "terraformOrganization.$": "$.terraformOrganization",
        "provisionedProductId.$": "$.provisionedProductId"
      },
      "ResultSelector": {
        "terraformRunId.$": "$.terraformRunId"
      },
      "ResultPath": "$.sendDestroyResult",
      "Catch": [
          {
              "ErrorEquals": [ "States.TaskFailed" ],
              "ResultPath": "$.errorInfo",
              "Next": "Notify destroy result failure"
          }
      ],
      "Next": "Wait for destroy to complete"
    },
    "Wait for destroy to complete": {
      "Type": "Wait",
      "Seconds": 1,
      "Next": "Poll destroy status"
    },
    "Poll destroy status": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.poll_run_status.arn}",
      "Parameters": {
        "terraformRunId.$": "$.sendDestroyResult.terraformRunId"
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
      "Next": "Did the destroy complete successfully?"
    },
    "Did the destroy complete successfully?": {
      "Type": "Choice",
      "Comment": "Looks-up the current status of the command invocation and delegates accordingly to handle it",
      "Choices": [
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "inProgress",
          "Next": "Wait for destroy to complete"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "failed",
          "Next": "Convert poll destroy status error"
        },
        {
          "Variable": "$.pollRunResult.productProvisioningStatus",
          "StringEquals": "success",
          "Next": "Notify destroy result"
        }
      ],
      "Default": "Failure"
    },
    "Convert poll destroy status error": {
        "Type": "Pass",
        "Comment": "Restructures error from the poll destroy status task to a format the notify run result task understands",
        "Parameters": {
            "Error": "Error applying run in TFC",
            "Cause.$": "$.pollRunStatus.errorMessage",
            "isWrapperError": true
        },
        "ResultPath": "$.errorInfo",
        "Next": "Notify destroy result failure"
    },
    "Notify destroy result": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.notify_run_result.arn}",
      "Parameters": {
        "terraformRunId.$": "$.sendDestroyResult.terraformRunId",
        "workflowToken.$": "$.token",
        "recordId.$": "$.recordId",
        "tracerTag.$": "$.tracerTag",
        "serviceCatalogOperation": "TERMINATING",
        "awsAccountId.$": "$.identity.awsAccountId",
        "terraformOrganization.$": "$.terraformOrganization",
        "provisionedProductId.$": "$.provisionedProductId"
      },
      "End": true
    },
    "Notify destroy result failure": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.notify_run_result.arn}",
      "Parameters": {
        "terraformRunId.$": "$.sendDestroyResult.terraformRunId",
        "workflowToken.$": "$.token",
        "recordId.$": "$.recordId",
        "tracerTag.$": "$.tracerTag",
        "serviceCatalogOperation": "TERMINATING",
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
