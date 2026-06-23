#!/bin/bash
# 一键销毁环境（测试完省钱）
ENV=$1

if [ -z "$ENV" ]; then
    echo "用法: bash destroy.sh <dev|staging|prod>"
    exit 1
fi

echo "⚠️  即将销毁 $ENV 环境的所有资源！"
read -p "确认销毁？输入环境名称确认: " CONFIRM

if [ "$CONFIRM" != "$ENV" ]; then
    echo "输入不匹配，取消操作"
    exit 1
fi

cd environments/$ENV
terraform destroy -auto-approve

echo "✅ $ENV 环境已销毁"