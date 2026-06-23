# ============================================
# VPC 网络模块
# 创建：VPC → 公网交换机 → 私网交换机 → 安全组
# ============================================

# 创建 VPC
resource "alicloud_vpc" "main" {
  vpc_name   = "${var.project}-${var.env}-vpc"
  cidr_block = var.vpc_cidr

  tags = {
    Project     = var.project
    Environment = var.env
    ManagedBy   = "terraform"
  }
}

# 公网交换机（放 SLB、堡垒机）
resource "alicloud_vswitch" "public" {
  vswitch_name = "${var.project}-${var.env}-public-vsw"
  vpc_id       = alicloud_vpc.main.id
  cidr_block   = var.public_subnet_cidr
  zone_id      = var.zone_id
}

# 私网交换机（放 ECS、RDS）
resource "alicloud_vswitch" "private" {
  vswitch_name = "${var.project}-${var.env}-private-vsw"
  vpc_id       = alicloud_vpc.main.id
  cidr_block   = var.private_subnet_cidr
  zone_id      = var.zone_id
}

# Web 安全组（对外开放 80/443）
resource "alicloud_security_group" "web" {
  security_group_name = "${var.project}-${var.env}-web-sg"
  vpc_id              = alicloud_vpc.main.id

  tags = {
    Project     = var.project
    Environment = var.env
    ManagedBy   = "terraform"
  }
}

# 安全组规则：允许 HTTP
resource "alicloud_security_group_rule" "http" {
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "80/80"
  security_group_id = alicloud_security_group.web.id
  cidr_ip           = "0.0.0.0/0"
}

# 安全组规则：允许 HTTPS
resource "alicloud_security_group_rule" "https" {
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "443/443"
  security_group_id = alicloud_security_group.web.id
  cidr_ip           = "0.0.0.0/0"
}

# 安全组规则：允许 SSH（仅限你的 IP）
resource "alicloud_security_group_rule" "ssh" {
  type              = "ingress"
  ip_protocol       = "tcp"
  port_range        = "22/22"
  security_group_id = alicloud_security_group.web.id
  cidr_ip           = var.admin_ip    # 只允许你的IP SSH进来
}

# 安全组规则：允许所有出站
resource "alicloud_security_group_rule" "egress" {
  type              = "egress"
  ip_protocol       = "all"
  port_range        = "-1/-1"
  security_group_id = alicloud_security_group.web.id
  cidr_ip           = "0.0.0.0/0"
}