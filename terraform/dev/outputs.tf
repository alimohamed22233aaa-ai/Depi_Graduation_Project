output "ecr_repository_urls" {
  description = "URLs of the ECR repositories"
  value       = module.ecr.repository_urls
}

output "eks_cluster_name" {
  description = "Name of the EKS Cluster"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint of the EKS Cluster"
  value       = module.eks.cluster_endpoint
}

output "postgres_endpoint" {
  description = "PostgreSQL RDS connection endpoint"
  value       = module.data_layer.postgres_endpoint
}

output "redis_endpoint" {
  description = "Redis ElastiCache connection endpoint"
  value       = module.data_layer.redis_endpoint
}