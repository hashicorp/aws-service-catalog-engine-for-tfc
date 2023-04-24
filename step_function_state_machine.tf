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

    resources = [aws_lambda_function.send_apply_command_function.arn]

  }

  statement {
    sid = "CloudwatchPermissions"

    effect = "Allow"

    actions = ["logs:CreateLogDelivery",
      "logs:GetLogDelivery",
      "logs:UpdateLogDelivery",
      "logs:DeleteLogDelivery",
      "logs:ListLogDeliveries",
      "logs:PutLogEvents",
      "logs:PutResourcePolicy",
      "logs:DescribeResourcePolicies",
    "logs:DescribeLogGroups"]

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
    level = "ALL"
    include_execution_data = true
    log_destination = "${aws_cloudwatch_log_group.tfc_manage_provisioned_product.arn}:*"
  }

  tracing_configuration {
    enabled = true
  }

  definition = <<EOF
{
  "Comment": "A Hello World example of the Amazon States Language using an AWS Lambda Function",
  "StartAt": "HelloWorld",
  "States": {
    "HelloWorld": {
      "Type": "Task",
      "Resource": "${aws_lambda_function.send_apply_command_function.arn}",
      "End": true
    }
  }
}
EOF
}
