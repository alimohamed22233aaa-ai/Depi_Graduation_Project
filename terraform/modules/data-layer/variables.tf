# ==============================================================================
# DATA LAYER MODULE VARIABLES
# ==============================================================================

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "onlineboutique"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "dev"
}

variable "vpc_id" {
  description = "ID of the VPC hosting EKS and the data layer"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs in at least two Availability Zones"
  type        = list(string)

  validation {
    condition     = length(var.private_subnet_ids) >= 2
    error_message = "Provide at least two private subnet IDs."
  }
}

variable "eks_nodes_security_group_id" {
  description = "Security group attached to the EKS worker nodes"
  type        = string
}

variable "db_username" {
  description = "RDS master username"
  type        = string
  default     = "boutique_admin"
}

variable "postgres_engine_version" {
  description = "PostgreSQL version supported in the selected AWS region"
  type        = string
}

variable "redis_engine_version" {
  description = "Redis engine version supported by ElastiCache"
  type        = string
  default     = "7.0"
}

