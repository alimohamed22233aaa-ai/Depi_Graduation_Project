resource "aws_ecr_repository" "depi" {
  name                 = var.repository_name
  image_tag_mutability = "IMMUTABLE"
  
  image_scanning_configuration {
    scan_on_push = true
  }
  tags = {
    ENVIRONMENT = var.environment
  }
}



resource "aws_ecr_lifecycle_policy" "depi" {
  repository = aws_ecr_repository.depi.id

  policy = <<EOF
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Keep last 15 images",
      "selection": {
        "tagStatus": "any",
        "countType": "imageCountMoreThan",
        "countNumber": 39
      },
      "action": {
        "type": "expire"
      }
    }
  ]
}
EOF
}

