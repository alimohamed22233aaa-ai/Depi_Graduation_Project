# ==============================================================================
# ECR MODULE VARIABLES
# ==============================================================================

variable "services" {
  type        = list(string)
  description = "List of microservices names to create ECR repositories for"
}

variable "project_name" {
  type        = string
  description = "Prefix name used for tagging and identifying ECR repositories"
}