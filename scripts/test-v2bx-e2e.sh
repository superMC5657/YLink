#!/usr/bin/env bash
# scripts/test-v2bx-e2e.sh
# 模拟 V2bX 节点守护程序与 YLink 服务端 UniProxy 兼容层进行端到端全链路通讯测试。
# 用法:
#   API_HOST=http://localhost:8081 NODE_ID=1 NODE_KEY=your_key bash scripts/test-v2bx-e2e.sh

set -euo pipefail

API_HOST="${API_HOST:-http://localhost:8081}"
NODE_ID="${NODE_ID:-1}"
NODE_KEY="${NODE_KEY:-node_key_test}"

echo "=========================================================="
echo "  YLink UniProxy / V2bX E2E Verification Harness"
echo "  Target: $API_HOST | NodeID: $NODE_ID"
echo "=========================================================="

pass_count=0
fail_count=0

assert_eq() {
  local desc="$1"
  local expected="$2"
  local actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "  [PASS] $desc"
    pass_count=$((pass_count + 1))
  else
    echo "  [FAIL] $desc (Expected: '$expected', Actual: '$actual')"
    fail_count=$((fail_count + 1))
  fi
}

echo ""
echo "--- 1. 鉴权与安全机制校验 ---"

# 1.1 缺失 Token 拦截 (期望 HTTP 401 与 40100 错误码)
RESP=$(curl -s -w "\n%{http_code}" "$API_HOST/api/v1/server/UniProxy/config?node_id=$NODE_ID" || true)
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_eq "缺失 Token 拦截为 401" "401" "$HTTP_CODE"

CODE_VAL=$(echo "$BODY" | grep -o '"code":[0-9]*' | cut -d: -f2 || true)
assert_eq "错误 Envelope 包含业务码 40100" "40100" "$CODE_VAL"

# 1.2 伪造 Token 拦截
RESP=$(curl -s -w "\n%{http_code}" "$API_HOST/api/v1/server/UniProxy/config?node_id=$NODE_ID&token=invalid_fake_token" || true)
HTTP_CODE=$(echo "$RESP" | tail -n1)
assert_eq "伪造 Token 拦截为 401" "401" "$HTTP_CODE"

echo ""
echo "--- 2. 节点配置下发 (GET /config) 与 ETag 校验 ---"

RESP=$(curl -s -i "$API_HOST/api/v1/server/UniProxy/config?node_id=$NODE_ID&token=$NODE_KEY" || true)
HTTP_CODE=$(echo "$RESP" | grep -E "^HTTP/[0-9.]+ " | tail -n1 | awk '{print $2}')
assert_eq "正确 Token 获取节点配置返回 200" "200" "$HTTP_CODE"

ETAG=$(echo "$RESP" | grep -i "^etag:" | tr -d '\r' | awk '{print $2}' || true)
if [ -n "$ETAG" ]; then
  RESP_304=$(curl -s -w "%{http_code}" -H "If-None-Match: $ETAG" "$API_HOST/api/v1/server/UniProxy/config?node_id=$NODE_ID&token=$NODE_KEY" || true)
  assert_eq "带匹配 ETag 请求下发配置返回 304 Not Modified" "304" "$RESP_304"
fi

echo ""
echo "--- 3. 有效用户下发 (GET /user) 与 ETag 校验 ---"

RESP=$(curl -s -i "$API_HOST/api/v1/server/UniProxy/user?node_id=$NODE_ID&token=$NODE_KEY" || true)
HTTP_CODE=$(echo "$RESP" | grep -E "^HTTP/[0-9.]+ " | tail -n1 | awk '{print $2}')
assert_eq "正确 Token 获取用户列表返回 200" "200" "$HTTP_CODE"

ETAG=$(echo "$RESP" | grep -i "^etag:" | tr -d '\r' | awk '{print $2}' || true)
if [ -n "$ETAG" ]; then
  RESP_304=$(curl -s -w "%{http_code}" -H "If-None-Match: $ETAG" "$API_HOST/api/v1/server/UniProxy/user?node_id=$NODE_ID&token=$NODE_KEY" || true)
  assert_eq "带匹配 ETag 请求用户下发返回 304 Not Modified" "304" "$RESP_304"
fi

echo ""
echo "--- 4. 增量流量上报 (POST /push) 兼容性校验 ---"

# 4.1 JSON 数组格式上报
RESP=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '[{"user_id": 1, "u": 1048576, "d": 2097152}]' \
  "$API_HOST/api/v1/server/UniProxy/push?node_id=$NODE_ID&token=$NODE_KEY" || true)
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_eq "JSON 数组格式增量流量上报返回 200" "200" "$HTTP_CODE"

# 4.2 V2bX 原生 Map 格式上报
RESP=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '{"1": [1048576, 2097152]}' \
  "$API_HOST/api/v1/server/UniProxy/push?node_id=$NODE_ID&token=$NODE_KEY" || true)
HTTP_CODE=$(echo "$RESP" | tail -n1)
BODY=$(echo "$RESP" | sed '$d')
assert_eq "V2bX 原生 Map 格式增量流量上报返回 200" "200" "$HTTP_CODE"

echo ""
echo "=========================================================="
echo "  测试结果: PASS: $pass_count, FAIL: $fail_count"
echo "=========================================================="

if [ "$fail_count" -gt 0 ]; then
  exit 1
fi
