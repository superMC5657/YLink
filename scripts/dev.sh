#!/usr/bin/env bash
# YLink 本地联调一键启动:PostgreSQL/Redis 容器 + 后端 api + worker + 前端 dev(连真实后端)
# 用法:bash scripts/dev.sh          # 启动(容器+后端+前端)
#      bash scripts/dev.sh -stop  # 关闭(停 api/worker/前端 + docker compose stop 容器)
set -euo pipefail

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

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="$ROOT/server"
RUN_DIR="$ROOT/.dev"
# 基础设施变量统一从 server/.env.dev 读取(与 docker-compose.yml 插值同源,compose 用 --env-file 显式指定);
# .env.dev 缺失时用默认值兜底,保证脚本可运行。
# 注意:.env.dev 含真实密钥已被 gitignore,模板见 server/.env.example(cp .env.example .env.dev)。
ENV_FILE="$SERVER/.env.dev"
pg_env() { grep -E "^$1=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2- || true; }
PG_USER="${PG_USER:-$(pg_env POSTGRES_USER)}"; PG_USER="${PG_USER:-ylink}"
PG_PASS="${PG_PASS:-$(pg_env POSTGRES_PASSWORD)}"; PG_PASS="${PG_PASS:-ylink_root}"
PG_DB="${PG_DB:-$(pg_env POSTGRES_DB)}"; PG_DB="${PG_DB:-ylink-backend}"
REDIS_PASS="${REDIS_PASS:-$(pg_env REDIS_PASSWORD)}"; REDIS_PASS="${REDIS_PASS:-ylink_redis}"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD="Admin@123456"
DEMO_EMAIL="demo@test.com"
DEMO_PASSWORD="Passw0rd"
JWT_SECRET="dev-join-test-secret-32bytes-key!!"

mkdir -p "$RUN_DIR"

# compose 的 --env-file 要求文件必须存在;server/.env.dev 缺失时生成兜底文件(仅基础设施插值变量),
# 保证脚本在全新检出后可直接运行,密钥类变量参考 server/.env.example 补齐。
if [ ! -f "$ENV_FILE" ]; then
  FALLBACK_ENV="$RUN_DIR/env.fallback"
  printf 'POSTGRES_USER=%s\nPOSTGRES_PASSWORD=%s\nPOSTGRES_DB=%s\nREDIS_PASSWORD=%s\n' \
    "$PG_USER" "$PG_PASS" "$PG_DB" "$REDIS_PASS" > "$FALLBACK_ENV"
  echo "  $ENV_FILE 不存在,已用默认值生成 $FALLBACK_ENV(基础设施可运行;真实密钥请参考 server/.env.example 补齐)"
  ENV_FILE="$FALLBACK_ENV"
fi

# ---------- 0b. 关闭模式 ----------
if [ "${1:-}" = "-stop" ]; then
  echo "关闭 YLink 本地服务..."
  stop_pid "$RUN_DIR/vite.pid"   "前端 dev"
  stop_pid "$RUN_DIR/api.pid"    "后端 api"
  stop_pid "$RUN_DIR/worker.pid" "后端 worker"
  # 关闭 docker compose 服务(postgres/redis;volume 数据保留,下次启动自动重建)
  if docker info >/dev/null 2>&1; then
    (cd "$SERVER" && docker compose --env-file "$ENV_FILE" stop postgres redis) && echo "  已关闭 postgres/redis 容器(docker compose stop,volume 数据保留)"
  else
    echo "  Docker daemon 未运行,跳过容器关闭"
  fi
  echo "完成。启动:bash scripts/dev.sh"
  exit 0
fi


say() { printf '\n[%s] %s\n' "$(date +%H:%M:%S)" "$*"; }

# ---------- 0. 端口占用检查 ----------
if curl -s -o /dev/null http://localhost:8081/healthz 2>/dev/null; then
  say "后端已在运行(http://localhost:8081),无需重复启动 api/worker。"
  API_ALREADY=1
else
  API_ALREADY=0
fi
if curl -s -o /dev/null http://localhost:5174/ 2>/dev/null; then
  say "前端 dev 已在运行(http://localhost:5174),无需重复启动。"
  VITE_ALREADY=1
else
  VITE_ALREADY=0
fi

# ---------- 1. 基础设施(PostgreSQL/Redis 容器) ----------
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

# 旧版 dev.sh 用 docker run 直接建容器(无 compose 标签),与 compose 编排不兼容;检测到则提示清理
if docker inspect yl-backend-postgres >/dev/null 2>&1 || docker inspect yl-backend-redis >/dev/null 2>&1; then
  echo "错误:检测到旧版 dev.sh(docker run 方式)创建的容器 yl-backend-postgres / yl-backend-redis。"
  echo "  新版改用 server/docker-compose.yml 编排,请先清理旧容器:"
  echo "    docker rm -f yl-backend-postgres yl-backend-redis"
  echo "  (旧容器数据在可写层,删除即丢;如需保留请先 docker commit 备份)"
  exit 1
fi

# 预拉镜像(本地缺失时走 daoCloud 加速,避免直连 Docker Hub 超时)
ensure_image postgres:16-alpine
ensure_image redis:7-alpine

# 由 server/docker-compose.yml 编排 postgres + redis;插值变量从 server/.env.dev 读取
(cd "$SERVER" && docker compose --env-file "$ENV_FILE" up -d postgres redis)
echo "  已通过 docker-compose.yml 启动 postgres + redis(项目名:${COMPOSE_PROJECT_NAME:-YLink})"

# 等待 PostgreSQL/Redis 就绪(最多 120s)
for i in $(seq 1 24); do
  PG_OK=0; REDIS_OK=0
  (cd "$SERVER" && docker compose exec -T postgres pg_isready -h127.0.0.1 -p5433 -U"$PG_USER") >/dev/null 2>&1 && PG_OK=1
  (cd "$SERVER" && docker compose exec -T redis redis-cli -a "$REDIS_PASS" ping) >/dev/null 2>&1 && REDIS_OK=1
  [ "$PG_OK" = 1 ] && [ "$REDIS_OK" = 1 ] && break
  [ $i -eq 24 ] && { echo "错误:PostgreSQL/Redis 120s 内未就绪,请 docker compose logs postgres / redis 排查"; exit 1; }
  sleep 5
done
echo "  PostgreSQL + Redis 就绪"

# ---------- 2. 数据库迁移(首次自动执行) ----------
say "[2/4] 检查数据库迁移..."
if ! (cd "$SERVER" && docker compose exec -T postgres psql -h 127.0.0.1 -p 5433 -U "$PG_USER" -d "$PG_DB" -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='users'") 2>/dev/null | grep -q 1; then
  echo "  首次启动,执行迁移(DSN 带 postgres:// 前缀)..."
  ( cd "$SERVER" && DB_URL="postgres://${PG_USER}:${PG_PASS}@127.0.0.1:5433/${PG_DB}?sslmode=disable" make migrate >/dev/null )
  echo "  迁移完成"
else
  echo "  已迁移,跳过"
fi

# ---------- 3. 后端 api + worker ----------
say "[3/4] 启动后端 api + worker..."
# APP_ENV 强制 development:模板(server/.env.example)默认 production,本地联调需开 Swagger/debug
export APP_ENV=development
export APP_DATABASE_DSN="host=127.0.0.1 port=5433 user=${PG_USER} password=${PG_PASS} dbname=${PG_DB} sslmode=disable TimeZone=Asia/Shanghai"
export APP_REDIS_ADDR=127.0.0.1:6379
export APP_REDIS_PASSWORD="$REDIS_PASS"
export APP_BASE_URL=http://localhost:8081
export APP_JWT_SECRET="$JWT_SECRET"
export ADMIN_EMAIL="$ADMIN_EMAIL" ADMIN_PASSWORD="$ADMIN_PASSWORD"
export DEMO_EMAIL="$DEMO_EMAIL" DEMO_PASSWORD="$DEMO_PASSWORD"
# SMTP 邮件(QQ 等)从 server/.env.dev 读取:APP_SMTP_USERNAME/APP_SMTP_PASSWORD/APP_SMTP_HOST/APP_SMTP_PORT/APP_SMTP_FROM_NAME
# 变量名统一为 viper 映射后的长名(config.yaml 的 smtp.username/smtp.password → APP_SMTP_USERNAME/APP_SMTP_PASSWORD),
# 与 .env.dev 里的 key 完全一致,无短名转换,避免后端读不到真实账号导致 535 认证失败。
if [ -f "$ENV_FILE" ]; then
  SMTP_HOST="$(grep -E '^APP_SMTP_HOST=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
  SMTP_PORT="$(grep -E '^APP_SMTP_PORT=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
  SMTP_USERNAME="$(grep -E '^APP_SMTP_USERNAME=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
  SMTP_PASSWORD="$(grep -E '^APP_SMTP_PASSWORD=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
  SMTP_FROM="$(grep -E '^APP_SMTP_FROM_NAME=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
else
  SMTP_HOST= SMTP_PORT= SMTP_USERNAME= SMTP_PASSWORD= SMTP_FROM=
fi
# 仅当 .env.dev 已配置 SMTP 时才导出(避免空值覆盖 config.yaml 里的 smtp 段)
if [ -n "$SMTP_USERNAME" ] || [ -n "$SMTP_PASSWORD" ]; then
  export APP_SMTP_HOST="$SMTP_HOST" APP_SMTP_PORT="$SMTP_PORT"
  export APP_SMTP_USERNAME="$SMTP_USERNAME" APP_SMTP_PASSWORD="$SMTP_PASSWORD"
  export APP_SMTP_FROM_NAME="$SMTP_FROM"
  say "  SMTP 已从 $ENV_FILE 读取(${SMTP_USERNAME:-未设置用户})"
fi
export APP_PAYMENT_EPAY_GATEWAY= APP_PAYMENT_EPAY_PID= APP_PAYMENT_EPAY_KEY=

if [ "$API_ALREADY" = 0 ]; then
  say "  构建 server/worker 二进制..."
  (
    cd "$SERVER"
    go build -o "$RUN_DIR/server.exe" ./cmd/server
    go build -o "$RUN_DIR/worker.exe" ./cmd/worker
  )
  echo "  构建完成:$RUN_DIR/server.exe $RUN_DIR/worker.exe"
  (
    cd "$SERVER"
    nohup "$RUN_DIR/server.exe" >> "$RUN_DIR/api.log" 2>&1 &
    echo $! > "$RUN_DIR/api.pid"
  )
  (
    cd "$SERVER"
    nohup "$RUN_DIR/worker.exe" >> "$RUN_DIR/worker.log" 2>&1 &
    echo $! > "$RUN_DIR/worker.pid"
  )
  for i in $(seq 1 30); do
    curl -s -o /dev/null http://localhost:8081/healthz 2>/dev/null && break
    [ $i -eq 30 ] && { echo "错误:api 启动超时,查看 $RUN_DIR/api.log"; exit 1; }
    sleep 2
  done
  echo "  api 就绪(http://localhost:8081),日志:$RUN_DIR/api.log"
else
  echo "  跳过(已在运行)"
fi

# ---------- 4. 前端 dev(连真实后端) ----------
say "[4/4] 启动前端 dev(真实后端,http://localhost:5174)..."
if [ "$VITE_ALREADY" = 0 ]; then
  (
    cd "$ROOT"
    # NO_COLOR=1:重定向到日志文件时禁用 ANSI 颜色码,避免 vite.log 出现 ESC 乱码
    NO_COLOR=1 nohup pnpm dev >> "$RUN_DIR/vite.log" 2>&1 &
    echo $! > "$RUN_DIR/vite.pid"
  )
  for i in $(seq 1 30); do
    curl -s -o /dev/null http://localhost:5174/ 2>/dev/null && break
    [ $i -eq 30 ] && { echo "错误:前端启动超时,查看 $RUN_DIR/vite.log"; exit 1; }
    sleep 2
  done
  echo "  前端就绪(http://localhost:5174),日志:$RUN_DIR/vite.log"
else
  echo "  跳过(已在运行)"
fi

say "全部就绪 🎉"
echo "------------------------------------------------------------"
echo "  前端:      http://localhost:5174"
echo "  后端 API:  http://localhost:8081  (/healthz /api/v1/config)"
echo "  演示账号:  demo@test.com / Passw0rd"
echo "  管理员:    admin@example.com / Admin@123456"
echo "  停止:      bash scripts/dev.sh -stop(含容器)"
echo "------------------------------------------------------------"
