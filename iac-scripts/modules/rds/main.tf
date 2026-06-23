# ============================================
# RDS MySQL 数据库模块
# ============================================

resource "alicloud_db_instance" "main" {
  engine           = "MySQL"
  engine_version   = "8.0"
  instance_type    = var.instance_type    # rds.mysql.s1.small
  instance_storage = var.storage_size
  instance_name    = "${var.project}-${var.env}-rds"
  vswitch_id       = var.vswitch_id

  security_ips = var.allowed_ips    # 只允许 ECS 内网 IP 访问

  tags = {
    Project     = var.project
    Environment = var.env
    ManagedBy   = "terraform"
  }
}

# 创建数据库
resource "alicloud_db_database" "app" {
  instance_id = alicloud_db_instance.main.id
  name        = var.db_name
  character_set = "utf8mb4"
}

# 创建数据库账号
resource "alicloud_db_account" "app" {
  db_instance_id   = alicloud_db_instance.main.id
  account_name     = var.db_user
  account_password = var.db_password
  account_type     = "Normal"
}

# 授权账号访问数据库
resource "alicloud_db_account_privilege" "app" {
  instance_id  = alicloud_db_instance.main.id
  account_name = alicloud_db_account.app.account_name
  privilege    = "ReadWrite"
  db_names     = [alicloud_db_database.app.name]
}