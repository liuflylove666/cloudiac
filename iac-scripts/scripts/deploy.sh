#!/bin/bash
# ============================================
# 一键部署脚本
# 用法：bash scripts/deploy.sh dev
# ============================================

set -e

ENV=$1
if [ -z "$ENV" ]; then
    echo "用法: bash deploy.sh <dev|staging|prod>"
    exit 1
fi

echo "======================================"
echo "  开始部署环境: $ENV"
echo "======================================"

# 第一阶段：Terraform 拉起云资源
echo ""
echo ">>> 第一阶段：Terraform 创建云资源..."
cd environments/$ENV

terraform init
terraform plan -out=tfplan
terraform apply tfplan

# 获取 ECS 公网 IP
ECS_IPS=$(terraform output -json ecs_public_ips | python3 -c "
import json, sys
ips = json.load(sys.stdin)
print('\n'.join(ips))
")

echo ""
echo "ECS 创建完成，IP 列表："
echo "$ECS_IPS"

# 第二阶段：等待 ECS 初始化完成
echo ""
echo ">>> 等待 ECS 初始化（约60秒）..."
sleep 60

# 第三阶段：生成 Ansible inventory
echo ""
echo ">>> 生成 Ansible 主机清单..."
cd ../../

cat > ansible/inventory/hosts.yml << EOF
all:
  vars:
    ansible_user: root
    ansible_ssh_private_key_file: ~/.ssh/id_rsa
    ansible_ssh_common_args: '-o StrictHostKeyChecking=no'
    env: $ENV

  children:
    web:
      hosts:
$(echo "$ECS_IPS" | while read ip; do
    echo "        $ip:"
done)
EOF

echo "主机清单已生成："
cat ansible/inventory/hosts.yml

# 第四阶段：Ansible 配置服务器
echo ""
echo ">>> 第四阶段：Ansible 配置服务器..."
cd ansible
ansible-playbook -i inventory/hosts.yml site.yml

echo ""
echo "======================================"
echo "  ✅ $ENV 环境部署完成！"
echo "======================================"
echo ""
echo "ECS 公网 IP："
echo "$ECS_IPS"