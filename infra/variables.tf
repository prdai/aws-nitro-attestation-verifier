variable "project_name" {
  description = "Name prefix for all resources."
  type        = string
  default     = "aws-nitro-attestation-verifier"
}

variable "aws_region" {
  description = "AWS region to deploy into."
  type        = string
  default     = "us-east-1"
}

variable "availability_zone" {
  description = "Single availability zone for the low-cost public subnet. Defaults to the cheapest observed us-east-1 Spot AZ during authoring."
  type        = string
  default     = "us-east-1d"
}

variable "instance_type" {
  description = "Low-cost Nitro Enclaves-capable instance selected for ECS on EC2. c6g.large is the cheapest on-demand option checked and a low-cost Spot fallback."
  type        = string
  default     = "c6g.large"
}

variable "use_spot" {
  description = "Use Spot for the ECS EC2 capacity host. Cheapest for experiments, but the enclave dies when the instance is interrupted."
  type        = bool
  default     = true
}

variable "root_volume_size_gib" {
  description = "Root EBS volume size for the ECS host. The ECS-optimized AL2023 AMI requires at least 30 GiB."
  type        = number
  default     = 30

  validation {
    condition     = var.root_volume_size_gib >= 30
    error_message = "root_volume_size_gib must be at least 30 because the ECS-optimized AL2023 AMI snapshot is 30 GiB."
  }
}

variable "http_port" {
  description = "Public HTTP port exposed on the EC2 host."
  type        = number
  default     = 8080
}

variable "allowed_http_cidr_blocks" {
  description = "CIDR blocks allowed to reach the EC2 host over HTTP. Defaults to public internet for experiments."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "ssh_key_name" {
  description = "Existing EC2 key pair name to attach to the parent host for SSH. Leave null to disable SSH key login."
  type        = string
  default     = null
}

variable "allowed_ssh_cidr_blocks" {
  description = "CIDR blocks allowed to reach the EC2 host over SSH. Keep this as your public IP /32, not 0.0.0.0/0."
  type        = list(string)
  default     = []
}

variable "enclave_memory_mib" {
  description = "Memory preallocated for Nitro Enclaves on the parent host. Lowering this does not reduce EC2 cost; it only leaves more memory for the parent."
  type        = number
  default     = 512
}

variable "enclave_cpu_count" {
  description = "vCPU count preallocated for Nitro Enclaves. c6gd.large has 2 vCPUs, so keep this at 1 for the cheapest host."
  type        = number
  default     = 1
}

variable "tags" {
  description = "Additional tags for created resources."
  type        = map(string)
  default     = {}
}
