terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}


provider "aws" {
  region  = var.region
  profile = "depi_project"
}
module "name" {
  source          = "../modules/ECR"
  repository_name = var.repository_name
  environment     = var.environment
}