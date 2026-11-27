
# ==============================================================================
# EKS CLUSTER
# ==============================================================================

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = "${var.project_name}-${var.environment}-eks"
  cluster_version = var.cluster_version

  # Kubernetes API endpoint access
  cluster_endpoint_public_access       = true
  cluster_endpoint_private_access      = true
  cluster_endpoint_public_access_cidrs = var.cluster_endpoint_public_access_cidrs

  # Networking
  vpc_id                   = var.vpc_id
  subnet_ids               = var.private_subnet_ids
  control_plane_subnet_ids = var.private_subnet_ids

  # IAM Roles for Service Accounts
  enable_irsa = true

  # Allow the Terraform execution identity to administer the cluster
  enable_cluster_creator_admin_permissions = true

  # Manage EKS add-ons through EKS
  bootstrap_self_managed_addons = false

  cluster_addons = {
    vpc-cni = {
      most_recent    = true
      before_compute = true
      preserve       = true
    }

    kube-proxy = {
      most_recent = true
      preserve    = true
    }

    coredns = {
      most_recent = true
      preserve    = true
    }
  }

  # Managed EC2 worker nodes
  eks_managed_node_groups = {
    default = {
      name            = "${var.project_name}-${var.environment}"
      use_name_prefix = false

      ami_type       = "AL2023_x86_64_STANDARD"
      instance_types = var.node_instance_types
      capacity_type  = "ON_DEMAND"

      subnet_ids = var.private_subnet_ids

      min_size     = var.min_size
      max_size     = var.max_size
      desired_size = var.desired_size

      update_config = {
        max_unavailable_percentage = 33
      }

      labels = {
        role        = "general"
        environment = var.environment
      }

      tags = {
        Name        = "${var.project_name}-${var.environment}-worker"
        Project     = var.project_name
        Environment = var.environment
        ManagedBy   = "Terraform"
      }
    }
  }

  tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}
