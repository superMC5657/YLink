#!/usr/bin/env bash
# YLink 全容器联调一键启动:PostgreSQL/Redis + api + worker 全部跑 Docker compose,
# 前端构建产物(dist/)由 Caddy 容器托管(静态 SPA + /api/* 反代 api:8081,同域无 CORS)。
# 与 scripts/dev.sh(宿主机进程 api/worker + Vite dev server)二选一,互不干扰。
# 用法:bash scripts/dev-docker.sh          # 启动(构建前端 → 构建镜像 → 起全部容器)
#      bash scripts/dev-docker.sh -stop  # 关闭(docker compose stop 全部服务,volume 数据保留)
# 前置:pnpm install 已执行(node_modules 就绪);Docker Desktop 运行中。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="$ROOT/server"
RUN_DIR="$ROOT/.dev"
SRC_ENV="$SERVER/.env.dev"        # 密钥来源(缺失时用默认值兜底,见 server/.env.example 模板)
COMPOSE_ENV="$RUN_DIR/env.docker" # 容器模式 env(compose --env-file + api/worker env_file 共用)
OVERRIDE="$RUN_DIR/docker-compose.dev.yml"
CADDYFILE="$RUN_DIR/Caddyfile.dev"

# 基础设施默认值兜底(与 dev.sh 同源:server/.env.dev,缺失时用默认,保证全新检出可跑)
pg_env() { grep -E "^$1=" "$SRC_ENV" 2>/dev/null | head -1 | cut -d= -f2- || true; }
PG_USER="${PG_USER:-$(pg_env POSTGRES_USER)}";   PG_USER="${PG_USER:-ylink}"
PG_PASS="${PG_PASS:-$(pg_env POSTGRES_PASSWORD)}"; PG_PASS="${PG_PASS:-ylink_root}"
PG_DB="${PG_DB:-$(pg_env POSTGRES_DB)}";         PG_DB="${PG_DB:-ylink-backend}"
REDIS_PASS="${REDIS_PASS:-$(pg_env REDIS_PASSWORD)}"; REDIS_PASS="${REDIS_PASS:-ylink_redis}"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD="Admin@123456"
DEMO_EMAIL="demo@test.com"
DEMO_PASSWORD="Passw0rd"
JWT_SECRET="dev-join-test-secret-32bytes-key!!"
SMTP_HOST="$(pg_env APP_SMTP_HOST)";  SMTP_PORT="$(pg_env APP_SMTP_PORT)"
SMTP_USERNAME="$(pg_env APP_SMTP_USERNAME)"; SMTP_PASSWORD="$(pg_env APP_SMTP_PASSWORD)"
SMTP_FROM="$(pg_env APP_SMTP_FROM_NAME)"

mkdir -p "$RUN_DIR"

# 供 compose 插值(bind mount 绝对路径);Git Bash 下用 cygpath 转 Windows 路径,否则原样(正斜杠)
DEV_CADDYFILE="$(cygpath -w "$CADDYFILE" 2>/dev/null || echo "$CADDYFILE")"
DEV_DIST="$(cygpath -w "$ROOT/dist" 2>/dev/null || echo "$ROOT/dist")"
# api/worker 的 env_file 走容器模式 env(base compose:${ENV_FILE:-.env.dev})
export ENV_FILE="$COMPOSE_ENV" DEV_CADDYFILE DEV_DIST

say() { printf '\n[%s] %s\n' "$(date +%H:%M:%S)" "$*"; }

# ---------- compose override:替换 caddy 的 Caddyfile + 挂载前端 dist ----------
# volumes 按挂载目标合并:同 target(/etc/caddy/Caddyfile)后者替换前者,其余保留。
cat > "$OVERRIDE" <<'EOF'
services:
  caddy:
    volumes:
      - ${DEV_CADDYFILE}:/etc/caddy/Caddyfile:ro
      - ${DEV_DIST}:/srv/panel:ro
EOF

# ---------- 生成容器模式 env(api/worker 容器内用服务名连接,与 .env.dev 的宿主机 DSN 区分) ----------
# 无副作用,start/-stop 共用:保证 -stop 时 --env-file 一定存在(缺失会让 compose 直接报错)。
{
  echo "APP_ENV=development"
  echo "APP_BASE_URL=http://localhost:8081"
  echo "APP_DATABASE_DSN=host=postgres port=5433 user=${PG_USER} password=${PG_PASS} dbname=${PG_DB} sslmode=disable TimeZone=Asia/Shanghai"
  echo "APP_REDIS_ADDR=redis:6379"
  echo "APP_REDIS_PASSWORD=${REDIS_PASS}"
  echo "APP_JWT_SECRET=${JWT_SECRET}"
  echo "ADMIN_EMAIL=${ADMIN_EMAIL}"
  echo "ADMIN_PASSWORD=${ADMIN_PASSWORD}"
  echo "DEMO_EMAIL=${DEMO_EMAIL}"
  echo "DEMO_PASSWORD=${DEMO_PASSWORD}"
  # SMTP 从 server/.env.dev 抄(若已配置);key 与 viper 长名映射一致,避免后端读不到真实账号
  if [ -n "$SMTP_USERNAME" ] || [ -n "$SMTP_PASSWORD" ]; then
    echo "APP_SMTP_HOST=${SMTP_HOST}"
    echo "APP_SMTP_PORT=${SMTP_PORT}"
    echo "APP_SMTP_USERNAME=${SMTP_USERNAME}"
    echo "APP_SMTP_PASSWORD=${SMTP_PASSWORD}"
    echo "APP_SMTP_FROM_NAME=${SMTP_FROM}"
  fi
  # compose 插值变量(--env-file 读取)
  echo "POSTGRES_USER=${PG_USER}"
  echo "POSTGRES_PASSWORD=${PG_PASS}"
  echo "POSTGRES_DB=${PG_DB}"
  echo "REDIS_PASSWORD=${REDIS_PASS}"
} > "$COMPOSE_ENV"

# ---------- 0b. 关闭模式 ----------
if [ "${1:-}" = "-stop" ]; then
  echo "关闭 YLink 全容器服务..."
  if docker info >/dev/null 2>&1; then
    (cd "$SERVER" && docker compose --env-file "$COMPOSE_ENV" -f docker-compose.yml -f "$OVERRIDE" stop postgres redis api worker caddy) \
      && echo "  已停止全部容器(docker compose stop,volume 数据保留)"
  else
    echo "  Docker daemon 未运行,跳过容器关闭"
  fi
  echo "完成。启动:bash scripts/dev-docker.sh"
  exit 0
fi


# ---------- 2. 检查 Docker ----------
say "[1/3] 检查 Docker daemon..."
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

# ---------- 3. 预拉镜像(caddy/postgres/redis 运行镜像 + api/worker 构建基础镜像,避免 compose 构建/up 时直连 Docker Hub 超时) ----------
say "[2/4] 预拉基础设施与构建基础镜像(daoCloud 加速)..."
ensure_image caddy:2-alpine
ensure_image postgres:16-alpine
ensure_image redis:7-alpine
ensure_image golang:1.26-alpine  # server/Dockerfile build 阶段(与 go.mod go 1.26.1 匹配)
ensure_image alpine:3.20          # server/Dockerfile 最终阶段

# ---------- 4. 前端构建(产物 dist/,由 Caddy 静态托管) ----------
say "[3/4] 构建前端(pnpm build → dist/)..."
(cd "$ROOT" && pnpm build)
echo "  构建完成:$ROOT/dist"

# ---------- 5. 生成 dev Caddyfile:localhost 托管 dist(/srv/panel 挂载点),同域反代 /api/* ----------
cat > "$CADDYFILE" <<'EOF'
localhost {
	root * /srv/panel
	handle /api/* {
		reverse_proxy api:8081
	}
	handle {
		try_files {path} /index.html
		file_server
	}
	header {
		X-Content-Type-Options nosniff
		Referrer-Policy no-referrer
	}
}
EOF

# ---------- 6. 构建镜像 + 启动全部服务 ----------
say "[4/4] 构建 api/worker 镜像并启动全部服务..."
(cd "$SERVER" && docker compose --env-file "$COMPOSE_ENV" -f docker-compose.yml -f "$OVERRIDE" up -d --build postgres redis api worker caddy)

# 等待 api 就绪(经宿主 127.0.0.1:8081 探活;容器链路由 depends_on healthcheck 保证)
for i in $(seq 1 30); do
  curl -s -o /dev/null http://localhost:8081/healthz 2>/dev/null && break
  [ $i -eq 30 ] && { echo "错误:api 启动超时,查看:docker compose logs api"; exit 1; }
  sleep 2
done
# 前端静态经 Caddy 探活
for i in $(seq 1 10); do
  curl -s -o /dev/null http://localhost/ 2>/dev/null && break
  [ $i -eq 10 ] && { echo "警告:Caddy(http://localhost)未就绪,查看:docker compose logs caddy"; break; }
  sleep 2
done

echo ""
echo "全部就绪 🎉"
echo "------------------------------------------------------------"
echo "  前端:      http://localhost        (Caddy 托管 dist 静态 + /api/v1 同域反代)"
echo "  后端直连:  http://localhost:8081   (含 /swagger/index.html,开发模式开启)"
echo "  演示账号:  demo@test.com / Passw0rd"
echo "  管理员:    admin@example.com / Admin@123456"
echo "  停止:      bash scripts/dev-docker.sh -stop(全部容器,volume 数据保留)"
echo "  日志:      docker compose -f $SERVER/docker-compose.yml -f $OVERRIDE logs -f api"
echo "------------------------------------------------------------"
