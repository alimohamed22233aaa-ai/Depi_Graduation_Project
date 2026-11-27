# ==============================================================================
# ECR REPOSITORIES CONFIGURATION
# ==============================================================================

# Create an ECR repository for each microservice dynamically using a loop
resource "aws_ecr_repository" "services" {
  for_each             = toset(var.services)
  name                 = "${var.project_name}/${each.key}"
  image_tag_mutability = "IMMUTABLE"

  # Automatically scan images for vulnerabilities upon push
  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = {
    Name = "${var.project_name}-${each.key}-ecr"
  }
}

# Lifecycle policy to automatically clean up untagged or old images to save storage
resource "aws_ecr_lifecycle_policy" "cleanup_policy" {
  for_each   = aws_ecr_repository.services
  repository = each.value.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep last 5 tagged images, expire untagged images older than 7 days"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 7
        }
        action = {
          type = "expire"
        }
      },
      {
        rulePriority = 2
        description  = "Keep the latest 5 dev images"

        selection = {
          tagStatus     = "tagged"
          tagPrefixList = ["dev-"]
          countType     = "imageCountMoreThan"
          countNumber   = 5
        }

        action = {
          type = "expire"
        }
      }
    ]
  })
}