# ==============================================================================
# VPC CONFIGURATION
# ==============================================================================

# Main Virtual Private Cloud (VPC) serving as the isolated network boundary
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.project_name}-vpc"
  }
}

# ==============================================================================
# GATEWAY & CONNECTIVITY
# ==============================================================================

# Internet Gateway to provide public internet access for public subnets
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.project_name}-igw"
  }
}

# ==============================================================================
# SUBNETS (PUBLIC & PRIVATE)
# ==============================================================================

# Public Subnets: Accessible from the internet, used for ALBs and Internet-facing resources
resource "aws_subnet" "public" {
  count                   = var.public_subnets_count
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name                     = "${var.project_name}-public-subnet-${count.index}"
    "kubernetes.io/role/elb" = "1"
  }
}

# Private Subnets: Isolated from the internet, used for EKS Nodes and internal workloads
resource "aws_subnet" "private" {
  count             = var.private_subnets_count
  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, count.index + var.public_subnets_count)
  availability_zone = var.availability_zones[count.index]

  tags = {
    Name                         = "${var.project_name}-private-subnet-${count.index}"
    "kubernetes.io/internal-elb" = "1" 
  }
}

# ==============================================================================
# ROUTING & NAT GATEWAYS
# ==============================================================================

# Public Route Table: Directs external traffic through the Internet Gateway
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "${var.project_name}-public-rt"
  }
}

# Associate Public Subnets with the Public Route Table
resource "aws_route_table_association" "public" {
  count          = var.public_subnets_count
  route_table_id = aws_route_table.public.id
  subnet_id      = aws_subnet.public[count.index].id
}

# Elastic IPs for NAT Gateways (One per public subnet / AZ)
resource "aws_eip" "eip" {
  count  = var.public_subnets_count
  domain = "vpc"

  tags = {
    Name = "${var.project_name}-nat-eip-${count.index}"
  }
}

# NAT Gateways: One NAT Gateway per public subnet for High Availability
resource "aws_nat_gateway" "main" {
  count         = var.public_subnets_count
  allocation_id = aws_eip.eip[count.index].id
  subnet_id     = aws_subnet.public[count.index].id

  tags = {
    Name = "${var.project_name}-nat-${count.index + 1}"
  }

  depends_on = [aws_internet_gateway.main]
}

# Private Route Table: Directs outbound traffic from private subnets through their respective NAT Gateways
resource "aws_route_table" "private" {
  count  = var.private_subnets_count
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main[count.index % var.public_subnets_count].id
  }

  tags = {
    Name = "${var.project_name}-private-rt-${count.index}"
  }
}

# Associate Private Subnets with their respective Private Route Tables
resource "aws_route_table_association" "private" {
  count          = var.private_subnets_count
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}