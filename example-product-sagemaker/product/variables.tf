variable "volume_size" {
  description = "Size in GB of the ML storage volume attached to the instance."
  type        = number
  default     = 20

  validation {
    condition     = var.volume_size >= 20 && var.volume_size <= 200
    error_message = "The volume size must be between 20 and 200 GB."
  }
}
