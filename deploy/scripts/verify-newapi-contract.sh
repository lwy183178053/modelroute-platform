#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
COMPOSE_FILE="$ROOT_DIR/deploy/newapi-contract/compose.yaml"
PROJECT_NAME=${NEW_API_CONTRACT_PROJECT:-modelroute-newapi-contract-$$}
PORT=${NEW_API_CONTRACT_PORT:-33100}
BASE_URL="http://127.0.0.1:$PORT"
export NEW_API_CONTRACT_PORT="$PORT"

dc() {
  docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" "$@"
}
cleanup() {
  dc down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

dc up -d --wait --wait-timeout 240
RUNNER="$ROOT_DIR/deploy/newapi-contract/run_contract.py"
if command -v wslpath >/dev/null 2>&1 && command -v python.exe >/dev/null 2>&1; then
  python.exe "$(wslpath -w "$RUNNER")" "$BASE_URL" "$(wslpath -w "$COMPOSE_FILE")" "$PROJECT_NAME"
else
  python3 "$RUNNER" "$BASE_URL" "$COMPOSE_FILE" "$PROJECT_NAME"
fi

echo "New API 候选容器契约通过：初始化、账号、令牌、渠道、模型映射、分层表达式、三类协议、stream、cached_tokens、refund 日志和 no_available_channel 均已验证。"
