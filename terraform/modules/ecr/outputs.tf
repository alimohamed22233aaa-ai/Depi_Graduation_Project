# ==============================================================================
# ECR MODULE OUTPUTS
# ==============================================================================

output "repository_urls" {
  description = "Map of microservice names to their corresponding ECR repository URLs"
  value       = { for k, repo in aws_ecr_repository.services : k => repo.repository_url }
}