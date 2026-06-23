# ============================================
# ECS 服务器模块
# ============================================

# 查询最新的镜像
data "alicloud_images" "rocky" {
  name_regex  = "^rockylinux_9"
  most_recent = true
  owners      = "system"
}

# 查询可用的实例规格
data "alicloud_instance_types" "main" {
  cpu_core_count = var.cpu_count
  memory_size    = var.memory_size
  network_type   = "Vpc"
}

# 创建 ECS 实例
resource "alicloud_instance" "main" {
  count = var.instance_count

  instance_name        = "${var.project}-${var.env}-ecs-${count.index + 1}"
  image_id             = data.alicloud_images.rocky.images[0].id
  instance_type        = data.alicloud_instance_types.main.instance_types[0].id
  vswitch_id           = var.vswitch_id
  security_groups      = [var.security_group_id]

  system_disk_category = "cloud_essd"
  system_disk_size     = 40

  key_name        = alicloud_key_pair.main.key_name

  internet_max_bandwidth_out = var.bandwidth

  tags = {
    Project     = var.project
    Environment = var.env
    Role        = var.role
    ManagedBy   = "terraform"
  }

  # ECS 创建后执行初始化脚本（必须放在 resource 块内部）
  user_data = base64encode(<<-EOF
    #!/bin/bash
    timedatectl set-timezone Asia/Shanghai
    yum update -y
    yum install -y wget curl vim git net-tools
    systemctl stop firewalld
    systemctl disable firewalld
    echo "ECS 初始化完成" > /tmp/init.log
  EOF
  )
}

# 创建密钥对
resource "alicloud_key_pair" "main" {
  key_pair_name = "${var.project}-${var.env}-keypair"
  public_key    = var.public_key
}

# 为 ECS 分配弹性公网 IP（EIP）
resource "alicloud_eip_address" "main" {
  count            = var.instance_count
  address_name     = "${var.project}-${var.env}-eip-${count.index + 1}"
  internet_charge_type = "PayByTraffic"
  bandwidth        = "10"
}

# 绑定 EIP 到 ECS
resource "alicloud_eip_association" "main" {
  count         = var.instance_count
  allocation_id = alicloud_eip_address.main[count.index].id
  instance_id   = alicloud_instance.main[count.index].id
}