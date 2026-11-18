# ==============================================================================
# 1. NETWORKING MODULE
# ==============================================================================
module "networking" {
  source = "../modules/networking"

  project_name          = var.project_name
  vpc_cidr              = var.vpc_cidr
  public_subnets_count  = 3
  private_subnets_count = 3

  availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]
}

# ==============================================================================
# 2. ECR MODULE
# ==============================================================================
module "ecr" {
  source = "../modules/ecr"

  project_name = var.project_name

  services = [
    "adservice",
    "cartservice",
    "ordersservice",
    "emailservice",
    "recommendationservice",
    "searchservice",
    "currencyservice",
    "paymentservice",
    "frontend",
    "productcatalogservice",
    "checkoutservice",
    "shippingservice"
  ]
}

# ==============================================================================
# 3. EKS MODULE
# ==============================================================================
module "eks" {
  source = "../modules/eks"

  project_name        = var.project_name
  environment         = var.environment
  vpc_id              = module.networking.vpc_id
  private_subnet_ids  = module.networking.private_subnets
  cluster_version     = "1.37"
  node_instance_types = ["c7i-flex.large"]

  desired_size = 2
  min_size     = 1
  max_size     = 3
}

# ==============================================================================
# 4. DATA LAYER MODULE
# ==============================================================================
module "data_layer" {
  source = "../modules/data-layer"

  project_name                = var.project_name
  environment                 = var.environment
  vpc_id                      = module.networking.vpc_id
  private_subnet_ids          = module.networking.private_subnets
  eks_nodes_security_group_id = module.eks.node_security_group_id

  postgres_engine_version = "18.6"
  redis_engine_version    = "7.0"
  db_username             = "dbadmin"
}
