# ==============================================================================
# VPC MODULE VARIABLES
# ==============================================================================

variable "vpc_cidr" {
  type        = string
  description = "The IPv4 CIDR block for the VPC (e.g., 10.0.0.0/16)"
}

variable "project_name" {
  type        = string
  description = "Prefix name used for tagging and identifying all VPC-related resources"
}

variable "public_subnets_count" {
  type        = number
  description = "Number of public subnets to create"
  default     = 3
}

variable "private_subnets_count" {
  type        = number
  description = "Number of private subnets to create"
  default     = 3
}

variable "availability_zones" {
  type        = list(string)
  description = "List of AWS Availability Zones to distribute subnets across"
}