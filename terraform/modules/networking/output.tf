# ==============================================================================
# VPC MODULE OUTPUTS
# ==============================================================================

output "vpc_id" {
  description = "The ID of the created VPC"
  value       = aws_vpc.main.id
}

output "private_subnets" {
  description = "List of IDs for the private subnets"
  value       = aws_subnet.private[*].id
}

output "public_subnets" {
  description = "List of IDs for the public subnets"
  value       = aws_subnet.public[*].id
}