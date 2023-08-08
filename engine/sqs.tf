# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "queue_key_policy" {
  version = "2012-10-17"

  statement {
    sid = "Enable KMS actions to principals in this account with IAM permissions"

    effect = "Allow"

    actions = ["kms:*"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }

    resources = ["*"]
  }

  statement {
    sid = "Enable AWS Service Catalog to send messages"

    effect = "Allow"

    actions = [
      "kms:DescribeKey",
      "kms:Decrypt",
      "kms:ReEncrypt",
      "kms:GenerateDataKey"
    ]

    principals {
      type        = "Service"
      identifiers = ["servicecatalog.amazonaws.com"]
    }

    resources = ["*"]
  }
}

resource "aws_kms_key" "queue_key" {
  description             = "symmetric encryption KMS key for SQS queues"
  enable_key_rotation     = true
  deletion_window_in_days = 30
  policy                  = data.aws_iam_policy_document.queue_key_policy.json
}

resource "aws_sqs_queue" "terraform_engine_provision_operation_queue" {
  name                       = "ServiceCatalogTerraformCloudProvisionOperationQueue"
  visibility_timeout_seconds = 180
  kms_master_key_id          = aws_kms_key.queue_key.key_id
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.terraform_engine_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sqs_queue" "terraform_engine_update_queue" {
  name                       = "ServiceCatalogTerraformCloudUpdateOperationQueue"
  visibility_timeout_seconds = 180
  kms_master_key_id          = aws_kms_key.queue_key.key_id
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.terraform_engine_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sqs_queue" "terraform_engine_terminate_queue" {
  name                       = "ServiceCatalogTerraformCloudTerminateOperationQueue"
  visibility_timeout_seconds = 180
  kms_master_key_id          = aws_kms_key.queue_key.key_id
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.terraform_engine_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sqs_queue" "terraform_engine_dlq" {
  name              = "ServiceCatalogTerraformCloudOperationsDLQ"
  kms_master_key_id = aws_kms_key.queue_key.key_id
}

data "aws_iam_policy_document" "queue_policy" {
  statement {
    sid    = "Enable AWS Service Catalog to send messages to the queue"
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["servicecatalog.amazonaws.com"]
    }
    actions = ["sqs:SendMessage", "sqs:GetQueueUrl"]
    resources = [
      aws_sqs_queue.terraform_engine_terminate_queue.arn,
      aws_sqs_queue.terraform_engine_provision_operation_queue.arn,
      aws_sqs_queue.terraform_engine_update_queue.arn
    ]
  }
}

resource "aws_sqs_queue_policy" "queue_policy" {
  for_each = {
    1 : aws_sqs_queue.terraform_engine_terminate_queue.id,
    2 : aws_sqs_queue.terraform_engine_provision_operation_queue.id,
    3 : aws_sqs_queue.terraform_engine_update_queue.id
  }

  queue_url = each.value
  policy    = data.aws_iam_policy_document.queue_policy.json
}
