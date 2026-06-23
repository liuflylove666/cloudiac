# Staging 比 Dev 多一个 SLB（因为有2台ECS）
terraform {
  required_providers {
    alicloud = {
      source  = "aliyun/alicloud"
      version = "~> 1.209"
    }
  }
}

provider "alicloud" { region = var.region }

module "vpc" {
  source              = "../../modules/vpc"
  project             = var.project
  env                 = var.env
  vpc_cidr            = var.vpc_cidr
  public_subnet_cidr  = var.public_subnet_cidr
  private_subnet_cidr = var.private_subnet_cidr
  zone_id             = var.zone_id
  admin_ip            = var.admin_ip
}

module "ecs" {
  source            = "../../modules/ecs"
  project           = var.project
  env               = var.env
  instance_count    = var.ecs_count
  cpu_count         = var.cpu_count
  memory_size       = var.memory_size
  vswitch_id        = module.vpc.private_vswitch_id
  security_group_id = module.vpc.web_sg_id
  public_key        = var.public_key
}

module "slb" {
  source             = "../../modules/slb"
  project            = var.project
  env                = var.env
  vswitch_id         = module.vpc.public_vswitch_id
  backend_server_ids = module.ecs.instance_ids
}

output "slb_ip"         { value = module.slb.slb_ip }
output "ecs_public_ips" { value = module.ecs.public_ips }