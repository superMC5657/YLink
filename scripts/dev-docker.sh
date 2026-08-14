#!/usr/bin/env bash
# YLink 全容器联调一键启动:PostgreSQL/Redis + api + worker 全部跑 Docker compose,
# 前端构建产物(dist/)由 Caddy 容器托管(静态 SPA + /api/* 反代 api:8081,同域无 CORS)。
# 与 scripts/dev.sh(宿主机进程 api/worker + Vite dev server)二选一,互不干扰。
# 用法:bash scripts/dev-docker.sh          # 启动(构建前端 → 构建镜像 → 起全部容器)
#      bash scripts/dev-docker.sh -stop  # 关闭(docker compose stop 全部服务,volume 数据保留)
# 前置:pnpm install 已执行(node_modules 就绪);Docker Desktop 运行中。
# 环境:所有配置以 server/.env.dev 为唯一来源,缺失即报错,脚本不内置任何默认值;
#       复制模板:cp server/.env.example server/.env.dev,再按需修改。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="$ROOT/server"
RUN_DIR="$ROOT/.dev"
SRC_ENV="$SERVER/.env.dev"        # 唯一 env 来源:api/worker env_file 与 compose 插值都读它,缺失即报错
OVERRIDE="$SERVER/docker-compose.dev.yml"  # 全容器联调 override,随仓库入库;缺失即报错
CADDYFILE="$RUN_DIR/Caddyfile.dev"

# ---------- 0. env/override 来源检查:两者都是仓库内文件,缺失即报错(不生成、不兜底) ----------
if [ ! -f "$SRC_ENV" ]; then
  echo "错误:$SRC_ENV 不存在。dev-docker.sh 的所有配置以 .env.dev 为唯一来源,请先创建:"
  echo "  cp $SERVER/.env.example $SRC_ENV"
  echo "  然后按需修改数据库口令、SMTP、管理员/演示账号等"
  exit 1
fi
if [ ! -f "$OVERRIDE" ]; then
  echo "错误:$OVERRIDE 不存在。全容器联调依赖该入库文件,请先拉取最新代码:"
  echo "  git pull"
  exit 1
fi
pg_env() { grep -E "^$1=" "$SRC_ENV" | head -1 | cut -d= -f2- || true; }
# 必需变量缺失时直接报错,不设默认值兜底,避免静默用错口令
require_env() {
  local name="$1"
  if [ -z "$(pg_env "$name")" ]; then
    echo "错误:$SRC_ENV 缺少必需变量 $name(参考 $SERVER/.env.example 补齐)"
    exit 1
  fi
}

mkdir -p "$RUN_DIR"

# 供 compose 插值(bind mount 绝对路径);Git Bash 下用 cygpath 转 Windows 路径,否则原样(正斜杠)
DEV_CADDYFILE="$(cygpath -w "$CADDYFILE" 2>/dev/null || echo "$CADDYFILE")"
DEV_DIST="$(cygpath -w "$ROOT/dist" 2>/dev/null || echo "$ROOT/dist")"
# api/worker 的 env_file 与 compose 插值统一用 server/.env.dev(base compose:${ENV_FILE:-.env.dev));
# 应用配置(APP_*/ADMIN_*/DEMO_*/SMTP 等)全部由 env_file 从 .env.dev 导入,脚本不写任何默认值;
# override(server/docker-compose.dev.yml,入库文件)只覆盖容器模式差异:DSN/Redis 用容器服务名 +
# 强制 APP_ENV=development;其 caddy volumes 的 ${DEV_CADDYFILE}/${DEV_DIST} 由这里 export 注入。
ENV_FILE="$SRC_ENV"
export ENV_FILE DEV_CADDYFILE DEV_DIST

say() { printf '\n[%s] %s\n' "$(date +%H:%M:%S)" "$*"; }

# ---------- 0b. 关闭模式 ----------
if [ "${1:-}" = "-stop" ]; then
  echo "关闭 YLink 全容器服务..."
  if docker info >/dev/null 2>&1; then
    if ! (cd "$SERVER" && docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f "$OVERRIDE" stop postgres redis api worker caddy); then
      echo "错误:docker compose stop 失败,请查看上方输出"
      exit 1
    fi
    echo "  已停止全部容器(docker compose stop,volume 数据保留)"
  else
    echo "  Docker daemon 未运行,跳过容器关闭"
  fi
  echo "完成。启动:bash scripts/dev-docker.sh"
  exit 0
fi

# ---------- 2. 检查 Docker 与必需基础设施变量 ----------
# 基础设施变量缺失会让 postgres/redis 容器起不来(空密码/空库名),必须显式报错
say "[1/4] 检查 Docker daemon 与必需环境变量..."
require_env POSTGRES_USER
require_env POSTGRES_PASSWORD
require_env POSTGRES_DB
require_env REDIS_PASSWORD
require_env APP_REDIS_PASSWORD
# redis 容器口令(--requirepass)与 api/worker 连接口令(APP_REDIS_PASSWORD)必须一致,否则 api 连 redis 认证失败
if [ "$(pg_env REDIS_PASSWORD)" != "$(pg_env APP_REDIS_PASSWORD)" ]; then
  echo "错误:$SRC_ENV 中 REDIS_PASSWORD 与 APP_REDIS_PASSWORD 不一致:"
  echo "  REDIS_PASSWORD=$(pg_env REDIS_PASSWORD)(redis 容器 --requirepass)"
  echo "  APP_REDIS_PASSWORD=$(pg_env APP_REDIS_PASSWORD)(api/worker 连接用)"
  echo "  两者必须相同"
  exit 1
fi
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
# 构建产物由 Caddy 同域托管(/api/* 反代 api:8081),必须用相对基址,覆盖 .env.production
# 里写死的 http://localhost:8081/api/v1(否则 Web 会跨端口直连 8081 触发浏览器 CORS 预检)。
# MSYS 路径转换:Git Bash(MINGW64)启动 Windows 原生程序 node 时,会把环境变量/参数里的
# POSIX 路径转成 Windows 路径(/api/v1 → C:/Program Files/Git/api/v1,虚拟根 / = Git 安装目录),
# 污染产物默认 apiBase(file:// 打开时请求变 file:///C:/Program%20Files/Git/...,报
# "Not allowed to load local resource")。
# MSYS_NO_PATHCONV=1 禁用全部路径转换;MSYS2_ENV_CONV_EXCL 再精确排除该变量(双保险,兼容不同 Git Bash 版本)。
say "[3/4] 构建前端(pnpm build → dist/)..."
(cd "$ROOT" && MSYS2_ENV_CONV_EXCL=VITE_API_BASE_URL MSYS_NO_PATHCONV=1 VITE_API_BASE_URL=/api/v1 pnpm build)
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
(cd "$SERVER" && docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f "$OVERRIDE" up -d --build postgres redis api worker caddy)

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

# 账号提示从 .env.dev 读取(缺失时提示未配置,后端 EnsureAdmin/EnsureDemoUser 会跳过创建)
ADMIN_EMAIL_HINT="$(pg_env ADMIN_EMAIL)";        [ -n "$ADMIN_EMAIL_HINT" ] || ADMIN_EMAIL_HINT="未配置(.env.dev)"
ADMIN_PASSWORD_HINT="$(pg_env ADMIN_PASSWORD)";  [ -n "$ADMIN_PASSWORD_HINT" ] || ADMIN_PASSWORD_HINT="未配置(.env.dev)"
DEMO_EMAIL_HINT="$(pg_env DEMO_EMAIL)";          [ -n "$DEMO_EMAIL_HINT" ] || DEMO_EMAIL_HINT="未配置(.env.dev)"
DEMO_PASSWORD_HINT="$(pg_env DEMO_PASSWORD)";    [ -n "$DEMO_PASSWORD_HINT" ] || DEMO_PASSWORD_HINT="未配置(.env.dev)"

echo ""
echo "全部就绪 🎉"
echo "------------------------------------------------------------"
echo "  前端:      http://localhost        (Caddy 托管 dist 静态 + /api/v1 同域反代)"
echo "  后端直连:  http://localhost:8081   (含 /swagger/index.html,开发模式开启)"
echo "  演示账号:  ${DEMO_EMAIL_HINT} / ${DEMO_PASSWORD_HINT}"
echo "  管理员:    ${ADMIN_EMAIL_HINT} / ${ADMIN_PASSWORD_HINT}"
echo "  停止:      bash scripts/dev-docker.sh -stop(全部容器,volume 数据保留)"
echo "  日志:      docker compose -f $SERVER/docker-compose.yml -f $OVERRIDE logs -f api"
echo "------------------------------------------------------------"
