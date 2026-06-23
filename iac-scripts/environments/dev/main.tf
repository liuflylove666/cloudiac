terraform {
  required_providers {
    alicloud = {
      source  = "aliyun/alicloud"
      version = "~> 1.209"
    }
  }

  # 远程状态存储（多人协作、状态不丢）
  # 先用本地，后面升级到 OSS
  # backend "oss" {
  #   bucket = "tf-state-bucket"
  #   prefix = "dev"
  # }
}

provider "alicloud" {
  region = var.region
}

# 调用 VPC 模块
module "vpc" {
  source = "../../modules/vpc"

  project             = var.project
  env                 = var.env
  vpc_cidr            = var.vpc_cidr
  public_subnet_cidr  = var.public_subnet_cidr
  private_subnet_cidr = var.private_subnet_cidr
  zone_id             = var.zone_id
  admin_ip            = var.admin_ip
}

# 调用 ECS 模块
module "ecs" {
  source = "../../modules/ecs"

  project           = var.project
  env               = var.env
  instance_count    = var.ecs_count
  cpu_count         = var.cpu_count
  memory_size       = var.memory_size
  vswitch_id        = module.vpc.private_vswitch_id
  security_group_id = module.vpc.web_sg_id
  public_key        = var.public_key
  role              = "web"
}

# Dev 环境不创建 RDS，用 Docker MySQL 代替（省钱）
# 不创建 SLB（只有1台ECS，不需要负载均衡）

# 输出关键信息
output "ecs_public_ips" {
  value       = module.ecs.public_ips
  description = "ECS 公网 IP，用于 SSH 和访问"
}

output "vpc_id" {
  value = module.vpc.vpc_id
}