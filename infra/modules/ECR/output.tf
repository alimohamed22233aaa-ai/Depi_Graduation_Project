output "ecr_repository_url" {
  description = "the url of repository"
  value       = aws_ecr_repository.depi.repository_url
}