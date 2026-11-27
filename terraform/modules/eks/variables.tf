# ==============================================================================
# EKS MODULE VARIABLES
# ==============================================================================

variable "project_name" {
  type        = string
  description = "Prefix name used for tagging and identifying EKS resources"
}

variable "environment" {
  type        = string
  description = "Environment name (e.g., dev)"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID where the EKS cluster will be deployed"
}

variable "private_subnet_ids" {
  type        = list(string)
  description = "List of private subnet IDs for EKS worker nodes"
}

variable "cluster_version" {
  type        = string
  description = "Kubernetes version for the EKS cluster"
  default     = "1.30"
}

variable "node_instance_types" {
  type        = list(string)
  description = "EC2 instance types for the EKS managed node group"
  default     = ["t3.small"]
}

variable "desired_size" {
  type        = number
  description = "Desired number of worker nodes"
  default     = 2
}

variable "min_size" {
  type        = number
  description = "Minimum number of worker nodes"
  default     = 1
}

variable "max_size" {
  type        = number
  description = "Maximum number of worker nodes"
  default     = 3
}

variable "cluster_endpoint_public_access_cidrs" {
  type        = list(string)
  description = "List of CIDR blocks which can access the Amazon EKS public API server endpoint"
  default     = ["0.0.0.0/0"]
}