#!/bin/bash
# 部署脚本 - 编译并部署到 deeix-chat-app

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== 1. 编译 ==="
./build.sh

echo ""
echo "=== 2. 部署 ==="
docker cp "$SCRIPT_DIR/deeix-chat" deeix-chat-app:/app/deeix-chat && echo "Binary 复制成功"
docker restart deeix-chat-app && echo "容器重启成功"

echo ""
echo "=== 3. 验证 ==="
sleep 3
curl -s http://localhost:8080/health | head -1 || echo "健康检查失败"
