
# ==============================================================================
# DATA LAYER SECURITY GROUPS
# ==============================================================================

resource "aws_security_group" "postgres_sg" {
  name        = "${var.project_name}-postgres-sg"
  description = "Allow PostgreSQL traffic from EKS only"
  vpc_id      = var.vpc_id

  ingress {
    description     = "PostgreSQL access from EKS nodes"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [var.eks_nodes_security_group_id]
  }

  egress {
    description = "Allow outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.project_name}-postgres-sg"
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

resource "aws_security_group" "redis_sg" {
  name        = "${var.project_name}-redis-sg"
  description = "Allow Redis traffic from EKS only"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Redis access from EKS nodes"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [var.eks_nodes_security_group_id]
  }

  egress {
    description = "Allow outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.project_name}-redis-sg"
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

# ==============================================================================
# POSTGRESQL - AMAZON RDS
# ==============================================================================

resource "aws_db_subnet_group" "postgres" {
  name        = "${var.project_name}-db-subnet-group"
  description = "Private subnets for PostgreSQL"
  subnet_ids  = var.private_subnet_ids

  tags = {
    Name        = "${var.project_name}-db-subnet-group"
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

resource "aws_db_instance" "postgres" {
  identifier = "${var.project_name}-postgres"

  engine         = "postgres"
  engine_version = var.postgres_engine_version
  instance_class = "db.t3.micro"

  db_name  = "onlineboutique"
  username = var.db_username

  # AWS manages the master password through Secrets Manager.
  manage_master_user_password = true

  allocated_storage     = 20
  max_allocated_storage = 50
  storage_type          = "gp3"
  storage_encrypted     = true

  db_subnet_group_name   = aws_db_subnet_group.postgres.name
  vpc_security_group_ids = [aws_security_group.postgres_sg.id]

  publicly_accessible = false

  backup_retention_period = 1
  auto_minor_version_upgrade = true

  # Development environment only: data may be lost on deletion.
  deletion_protection = false
  skip_final_snapshot = true

  tags = {
    Name        = "${var.project_name}-postgres"
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

# ==============================================================================
# REDIS - AMAZON ELASTICACHE
# ==============================================================================

resource "aws_elasticache_subnet_group" "redis" {
  name        = "${var.project_name}-redis-subnet-group"
  description = "Private subnets for Redis"
  subnet_ids  = var.private_subnet_ids

  tags = {
    Name        = "${var.project_name}-redis-subnet-group"
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id = "${var.project_name}-redis"
  description          = "Redis for Online Boutique"

  engine         = "redis"
  engine_version = var.redis_engine_version
  node_type      = "cache.t3.micro"

  num_cache_clusters         = 1
  automatic_failover_enabled = false
  multi_az_enabled           = false

  parameter_group_name = "default.redis7"
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.redis.name
  security_group_ids = [aws_security_group.redis_sg.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true

  tags = {
    Name        = "${var.project_name}-redis"
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}
