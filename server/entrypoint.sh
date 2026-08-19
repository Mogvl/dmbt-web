#!/bin/sh
# dmbt-web server entrypoint:
# 1. 确保数据目录存在且可写 — 宿主机 bind mount (./data:/data) 由 docker 以
#    root 创建, 容器内非 root 用户无法写入, 必须在启动时修正属主。
# 2. 降权到 animegarden 用户运行服务。
set -e

DATA="${DATA_DIR:-/data}"
mkdir -p "$DATA"
chown -R animegarden:animegarden "$DATA"

exec su-exec animegarden /app/server "$@"
