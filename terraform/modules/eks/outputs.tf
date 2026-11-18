# ==============================================================================
# EKS MODULE OUTPUTS
# ==============================================================================

output "cluster_name" {
  description = "The name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "cluster_endpoint" {
  description = "Endpoint for EKS control plane"
  value       = module.eks.cluster_endpoint
}

output "cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster control plane"
  value       = module.eks.cluster_security_group_id
}

output "node_security_group_id" {
  description = "Security group ID attached to the EKS worker nodes (vital for RDS/Redis rules)"
  value       = module.eks.node_security_group_id
}

output "oidc_provider_arn" {
  description = "OIDC Provider ARN for IRSA configuration"
  value       = module.eks.oidc_provider_arn
}