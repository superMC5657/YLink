#!/usr/bin/env bash
# YLink 本地联调一键停止:停 api / worker / 前端 dev(容器默认保留,数据不丢)
# 用法:bash scripts/dev-down.sh
# 若也想停掉 MySQL/Redis 容器:bash scripts/dev-down.sh --containers
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT/.dev"

stop_pid() {
  local pid_file="$1" name="$2"
  [ -f "$pid_file" ] || { echo "  $name:无 PID 记录,跳过"; return; }
  local pid
  pid="$(cat "$pid_file")"
  if [ -n "$pid" ]; then
    # Windows 用 taskkill 杀进程树(go run/pnpm 有子进程),非 Windows 用 kill
    if taskkill //F //T //PID "$pid" >/dev/null 2>&1; then
      echo "  已停止 $name (pid $pid)"
    elif kill "$pid" >/dev/null 2>&1; then
      echo "  已停止 $name (pid $pid)"
    else
      echo "  $name (pid $pid) 已不在运行"
    fi
  fi
  rm -f "$pid_file"
}

echo "停止 YLink 本地服务..."
stop_pid "$RUN_DIR/vite.pid"   "前端 dev"
stop_pid "$RUN_DIR/api.pid"    "后端 api"
stop_pid "$RUN_DIR/worker.pid" "后端 worker"

if [ "${1:-}" = "--containers" ]; then
  echo "停止 MySQL/Redis 容器(数据保留在 volume)..."
  docker stop yl-backend-mysql yl-backend-redis >/dev/null 2>&1 && echo "  已停止 yl-backend-mysql yl-backend-redis"
fi

echo "完成。如需重启:bash scripts/dev-up.sh"
