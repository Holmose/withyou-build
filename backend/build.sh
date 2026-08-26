#!/bin/bash
# Go 后端编译脚本 - 用于 withyou-runtime SSE 事件透传

set -e

IMAGE="golang:1.26-bookworm"
CONTAINER_NAME="deeix-build-helper"
MOUNT_SRC="$HOME/Documents/Docker/deeix-chat-withyou/backend"
MOUNT_DST="/app"

echo "=== 编译 deeix-chat 后端 ==="

# 检查是否有现成容器
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "使用已有容器..."
    docker start ${CONTAINER_NAME} 2>/dev/null || docker rm ${CONTAINER_NAME}
fi

# 编译
docker run --name ${CONTAINER_NAME} \
    -v "${MOUNT_SRC}:${MOUNT_DST}" \
    -w ${MOUNT_DST} \
    -e GOPROXY=https://goproxy.cn,direct \
    ${IMAGE} \
    bash -c "apt-get update > /dev/null 2>&1 && apt-get install -y libsqlite3-dev > /dev/null 2>&1 && go build -ldflags='-s -w' -o ${MOUNT_DST}/deeix-chat ${MOUNT_DST}/cmd/server"

echo "=== 编译完成 ==="
ls -la ${MOUNT_SRC}/deeix-chat
