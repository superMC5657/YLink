#!/usr/bin/env bash
# YLink 本地联调一键启动:MySQL/Redis 容器 + 后端 api + worker + 前端 dev(连真实后端)
# 用法:bash scripts/dev-up.sh
# 停止:bash scripts/dev-down.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="$ROOT/server"
RUN_DIR="$ROOT/.dev"
MYSQL_PASS="ylink_root"
REDIS_PASS="ylink_redis"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD="Admin@123456"
DEMO_EMAIL="demo@test.com"
DEMO_PASSWORD="Passw0rd"
JWT_SECRET="dev-join-test-secret-32bytes-key!!"

mkdir -p "$RUN_DIR" "$SERVER/logs"

say() { printf '\n[%s] %s\n' "$(date +%H:%M:%S)" "$*"; }

# ---------- 0. 端口占用检查 ----------
if curl -s -o /dev/null http://localhost:8080/healthz 2>/dev/null; then
  say "后端已在运行(http://localhost:8080),无需重复启动 api/worker。"
  API_ALREADY=1
else
  API_ALREADY=0
fi
if curl -s -o /dev/null http://localhost:5173/ 2>/dev/null; then
  say "前端 dev 已在运行(http://localhost:5173),无需重复启动。"
  VITE_ALREADY=1
else
  VITE_ALREADY=0
fi

# ---------- 1. 基础设施(MySQL/Redis 容器) ----------
say "[1/4] 检查 Docker 与基础设施容器..."
docker info >/dev/null 2>&1 || { echo "错误:Docker daemon 未运行,请先启动 Docker Desktop"; exit 1; }

ensure_image() {
  # 本地无镜像时优先走 daoCloud 加速源(直连 Docker Hub 常超时),再 tag 回标准名
  local name="$1"
  if ! docker image inspect "$name" >/dev/null 2>&1; then
    echo "  拉取镜像 $name(经 daoCloud 加速)..."
    docker pull "docker.m.daocloud.io/library/$name"
    docker tag "docker.m.daocloud.io/library/$name" "$name"
  fi
}

if ! docker inspect yl-mysql >/dev/null 2>&1; then
  ensure_image mysql:8.0
  docker run -d --name yl-mysql \
    -e MYSQL_ROOT_PASSWORD="$MYSQL_PASS" -e MYSQL_DATABASE=ylink \
    -p 127.0.0.1:3306:3306 \
    mysql:8.0 --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci >/dev/null
  echo "  已创建 yl-mysql"
else
  docker start yl-mysql >/dev/null 2>&1 || true
  echo "  yl-mysql 已存在,启动中"
fi

if ! docker inspect yl-redis >/dev/null 2>&1; then
  ensure_image redis:7-alpine
  docker run -d --name yl-redis \
    -p 127.0.0.1:6379:6379 \
    redis:7-alpine redis-server --requirepass "$REDIS_PASS" >/dev/null
  echo "  已创建 yl-redis"
else
  docker start yl-redis >/dev/null 2>&1 || true
  echo "  yl-redis 已存在,启动中"
fi

# 等待 MySQL/Redis 就绪(最多 120s)
for i in $(seq 1 24); do
  MYSQL_OK=0; REDIS_OK=0
  docker exec yl-mysql mysqladmin ping -h127.0.0.1 -uroot -p"$MYSQL_PASS" --silent >/dev/null 2>&1 && MYSQL_OK=1
  docker exec yl-redis redis-cli -a "$REDIS_PASS" ping >/dev/null 2>&1 && REDIS_OK=1
  [ "$MYSQL_OK" = 1 ] && [ "$REDIS_OK" = 1 ] && break
  [ $i -eq 24 ] && { echo "错误:MySQL/Redis 5s 内未就绪,请 docker logs yl-mysql / yl-redis 排查"; exit 1; }
  sleep 5
done
echo "  MySQL + Redis 就绪"

# ---------- 2. 数据库迁移(首次自动执行) ----------
say "[2/4] 检查数据库迁移..."
if ! docker exec yl-mysql mysql -uroot -p"$MYSQL_PASS" -e "SHOW TABLES LIKE 'users'" ylink 2>/dev/null | grep -q users; then
  echo "  首次启动,执行迁移(DSN 带 mysql:// 前缀)..."
  ( cd "$SERVER" && DB_URL="mysql://root:${MYSQL_PASS}@tcp(127.0.0.1:3306)/ylink?charset=utf8mb4&parseTime=true&loc=Local" make migrate >/dev/null )
  echo "  迁移完成"
else
  echo "  已迁移,跳过"
fi

# ---------- 3. 后端 api + worker ----------
say "[3/4] 启动后端 api + worker..."
export APP_DATABASE_DSN="root:${MYSQL_PASS}@tcp(127.0.0.1:3306)/ylink?charset=utf8mb4&parseTime=true&loc=Local"
export APP_REDIS_ADDR=127.0.0.1:6379
export APP_REDIS_PASSWORD="$REDIS_PASS"
export APP_BASE_URL=http://localhost:8080
export APP_JWT_SECRET="$JWT_SECRET"
export ADMIN_EMAIL="$ADMIN_EMAIL" ADMIN_PASSWORD="$ADMIN_PASSWORD"
export DEMO_EMAIL="$DEMO_EMAIL" DEMO_PASSWORD="$DEMO_PASSWORD"
export APP_SMTP_USER= APP_SMTP_PASS=
export APP_PAYMENT_EPAY_GATEWAY= APP_PAYMENT_EPAY_PID= APP_PAYMENT_EPAY_KEY=

if [ "$API_ALREADY" = 0 ]; then
  (
    cd "$SERVER"
    nohup go run ./cmd/server >> "$SERVER/logs/api.log" 2>&1 &
    echo $! > "$RUN_DIR/api.pid"
  )
  (
    cd "$SERVER"
    nohup go run ./cmd/worker >> "$SERVER/logs/worker.log" 2>&1 &
    echo $! > "$RUN_DIR/worker.pid"
  )
  for i in $(seq 1 30); do
    curl -s -o /dev/null http://localhost:8080/healthz 2>/dev/null && break
    [ $i -eq 30 ] && { echo "错误:api 启动超时,查看 $SERVER/logs/api.log"; exit 1; }
    sleep 2
  done
  echo "  api 就绪(http://localhost:8080),日志:server/logs/api.log"
else
  echo "  跳过(已在运行)"
fi

# ---------- 4. 前端 dev(连真实后端) ----------
say "[4/4] 启动前端 dev(真实后端,http://localhost:5173)..."
if [ "$VITE_ALREADY" = 0 ]; then
  (
    cd "$ROOT"
    nohup pnpm dev >> "$RUN_DIR/vite.log" 2>&1 &
    echo $! > "$RUN_DIR/vite.pid"
  )
  for i in $(seq 1 30); do
    curl -s -o /dev/null http://localhost:5173/ 2>/dev/null && break
    [ $i -eq 30 ] && { echo "错误:前端启动超时,查看 $RUN_DIR/vite.log"; exit 1; }
    sleep 2
  done
  echo "  前端就绪(http://localhost:5173),日志:.dev/vite.log"
else
  echo "  跳过(已在运行)"
fi

say "全部就绪 🎉"
echo "------------------------------------------------------------"
echo "  前端:      http://localhost:5173"
echo "  后端 API:  http://localhost:8080  (/healthz /api/v1/config)"
echo "  演示账号:  demo@test.com / Passw0rd"
echo "  管理员:    admin@example.com / Admin@123456"
echo "  停止:      bash scripts/dev-down.sh"
echo "------------------------------------------------------------"
