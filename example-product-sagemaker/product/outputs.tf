output "notebook_url" {
  description = "URL of the SageMaker notebook instance."
  value       = "https://${aws_sagemaker_notebook_instance.datasci.url}/lab"
}

output "bucket_name" {
  description = "Name of the S3 storage bucket for your data."
  value       = aws_s3_bucket.data.bucket
}
