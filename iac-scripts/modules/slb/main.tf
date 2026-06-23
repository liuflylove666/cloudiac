# ============================================
# SLB 负载均衡模块
# ============================================

resource "alicloud_slb_load_balancer" "main" {
  load_balancer_name = "${var.project}-${var.env}-slb"
  load_balancer_spec = "slb.s1.small"
  vswitch_id         = var.vswitch_id
  address_type       = "internet"    # 公网 SLB

  tags = {
    Project     = var.project
    Environment = var.env
    ManagedBy   = "terraform"
  }
}

# HTTP 监听
resource "alicloud_slb_listener" "http" {
  load_balancer_id = alicloud_slb_load_balancer.main.id
  backend_port     = 80
  frontend_port    = 80
  protocol         = "http"
  bandwidth        = 10

  health_check            = "on"
  health_check_uri        = "/health"
  health_check_timeout    = 5
  health_check_interval   = 10
  healthy_threshold       = 2
  unhealthy_threshold     = 3
}

# 后端服务器组
resource "alicloud_slb_server_group" "main" {
  load_balancer_id = alicloud_slb_load_balancer.main.id
  name             = "${var.project}-${var.env}-servers"

  dynamic "servers" {
    for_each = var.backend_server_ids
    content {
      server_ids = [servers.value]
      port       = 80
      weight     = 100
    }
  }
}