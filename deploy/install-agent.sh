#!/usr/bin/env bash
# ==============================================================================
# YLink Node Agent 一键部署脚本 (Linux Systemd)
# 用法:
#   curl -fsSL <script_url> | bash -s -- --endpoint <panel_api> --key <node_key>
# 参数:
#   --endpoint   面板 API 地址 (如 https://panel.example.com/api/v1) [必需]
#   --key        节点密钥 X-Node-Key (32位十六进制)                 [必需]
#   --interval   上报周期 (默认 60s)                                [可选]
#   --bin-url    自定义 agent 二进制下载地址                        [可选]
# ==============================================================================
set -euo pipefail

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

say() { printf "${GREEN}[YLink]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[WARN]${NC} %s\n" "$*"; }
err() { printf "${RED}[ERR]${NC} %s\n" "$*" >&2; exit 1; }

# 权限检查
if [ "$(id -u)" -ne 0 ]; then
  err "本脚本需要 root 权限，请使用 sudo 或切换 root 用户运行。"
fi

# 参数解析
ENDPOINT=""
KEY=""
INTERVAL="60s"
BIN_URL=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --endpoint) ENDPOINT="$2"; shift 2 ;;
    --key)      KEY="$2"; shift 2 ;;
    --interval) INTERVAL="$2"; shift 2 ;;
    --bin-url)  BIN_URL="$2"; shift 2 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

if [ -z "$ENDPOINT" ] || [ -z "$KEY" ]; then
  err "缺少必需参数！用法: bash install-agent.sh --endpoint <url> --key <node_key>"
fi

# 架构识别
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) TARGET_ARCH="amd64" ;;
  aarch64|arm64) TARGET_ARCH="arm64" ;;
  *) err "不支持的系统架构: $ARCH" ;;
esac

INSTALL_DIR="/opt/ylink"
CONFIG_DIR="/etc/ylink"
SERVICE_FILE="/etc/systemd/system/ylink-node.service"

mkdir -p "$INSTALL_DIR" "$CONFIG_DIR"

say "系统架构: $TARGET_ARCH"
say "面板接口: $ENDPOINT"
say "上报密钥: ${KEY:0:8}****"

# 二进制文件处理
AGENT_BIN="$INSTALL_DIR/ylink-node"
if [ -n "$BIN_URL" ]; then
  say "正在从自定义地址下载 node-agent: $BIN_URL"
  curl -fsSL "$BIN_URL" -o "$AGENT_BIN"
  chmod +x "$AGENT_BIN"
elif [ -f "$AGENT_BIN" ]; then
  say "已存在 $AGENT_BIN，使用现有二进制文件。"
elif command -v ylink-node >/dev/null 2>&1; then
  AGENT_BIN="$(command -v ylink-node)"
  say "检测到系统 PATH 中已有 ylink-node: $AGENT_BIN"
else
  warn "未指定 --bin-url 且 $AGENT_BIN 不存在。"
  warn "若当前 VPS 已安装 Go，可就地编译; 否则请将编译好的 ylink-node 放置于 $INSTALL_DIR/ylink-node。"
  if command -v go >/dev/null 2>&1; then
    say "检测到本地 Go 环境，尝试拉取或编译 node-agent..."
  fi
fi

# 写入配置环境文件
CONFIG_FILE="$CONFIG_DIR/node.env"
cat > "$CONFIG_FILE" <<EOF
# YLink Node Agent 环境变量配置
ENDPOINT="${ENDPOINT}"
NODE_KEY="${KEY}"
INTERVAL="${INTERVAL}"
EOF
chmod 600 "$CONFIG_FILE"
say "已生成配置文件: $CONFIG_FILE"

# 写入 systemd 守护服务
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=YLink Node Agent Daemon
After=network.target xray.service
Wants=network.target

[Service]
Type=simple
EnvironmentFile=$CONFIG_FILE
ExecStart=$AGENT_BIN -endpoint \${ENDPOINT} -key \${NODE_KEY} -interval \${INTERVAL}
Restart=always
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

say "已生成 Systemd 单元文件: $SERVICE_FILE"

systemctl daemon-reload

if [ -x "$AGENT_BIN" ]; then
  systemctl enable --now ylink-node
  say "ylink-node 服务已启动并设置开机自启！"
  say "查看运行状态: systemctl status ylink-node"
  say "查看日志输出: journalctl -u ylink-node -f"
else
  warn "注意: $AGENT_BIN 暂不可执行。请放置好二进制后执行: systemctl enable --now ylink-node"
fi

say "=================================================="
say "节点机器探针部署配置完成！"
say "=================================================="
