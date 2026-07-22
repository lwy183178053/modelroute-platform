#!/bin/sh
set -eu

BASE_URL=${1:-http://127.0.0.1:28100}

live=$(curl -fsS "$BASE_URL/api/health/live")
printf '%s' "$live" | grep -q '"success":true'

ready=$(curl -fsS "$BASE_URL/api/health/ready")
printf '%s' "$ready" | grep -q '"status":"ready"'

page=$(curl -fsS "$BASE_URL/")
printf '%s' "$page" | grep -q '<title>ModelRoute'

status=$(curl -sS -o /tmp/modelroute-v1-check.json -w '%{http_code}' \
  -X POST "$BASE_URL/v1/chat/completions" \
  -H 'Content-Type: application/json' \
  --data '{"model":"foundation-check","messages":[]}')
test "$status" = "401"
grep -q '"code":"invalid_api_key"' /tmp/modelroute-v1-check.json
rm -f /tmp/modelroute-v1-check.json

echo "ModelRoute 部署验收通过：页面、存活、就绪和模型入口鉴权状态正常。"
