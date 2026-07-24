#!/bin/sh
set -eu
umask 077

BASE_URL="${1:-https://aggregate.lwylink.xyz}"
BASE_URL="${BASE_URL%/}"
ADMIN_USERNAME="${ADMIN_USERNAME:-3180615598@qq.com}"
ADMIN_PASSWORD_FILE="${ADMIN_PASSWORD_FILE:-/opt/modelroute-platform/shared/bootstrap-admin.txt}"
ROOT_DIR=$(CDPATH= cd -P -- "$(dirname -- "$0")/../.." && pwd -P)
cd "$ROOT_DIR"

if [ ! -r "$ADMIN_PASSWORD_FILE" ] || [ ! -r .env ]; then
  echo "管理员密码文件或生产环境文件不可读" >&2
  exit 1
fi
set -a
. ./.env
set +a

WORK_DIR="$(mktemp -d)"
COOKIE_JAR="$WORK_DIR/cookies.txt"
KEY_ID=""
KEY_NAME=""

purge_test_key() {
  case "$KEY_ID" in
    ''|*[!0-9]*) return ;;
  esac
  case "$KEY_NAME" in
    contract-*) ;;
    *) return ;;
  esac
  docker compose --env-file .env exec -T postgres psql -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_DB_NAME" \
    -c "DELETE FROM token_bindings WHERE newapi_token_id = $KEY_ID" </dev/null >/dev/null 2>&1 || true
  docker compose --env-file .env exec -T postgres psql -v ON_ERROR_STOP=1 \
    -U "$POSTGRES_SUPERUSER" -d "$NEWAPI_DB_NAME" \
    -c "DELETE FROM tokens WHERE id = $KEY_ID AND name = '$KEY_NAME'" </dev/null >/dev/null 2>&1 || true
}

cleanup() {
  if [ -n "$KEY_ID" ]; then
    curl -sS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >/dev/null 2>&1 || true
    purge_test_key
  fi
  curl -sS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/auth/logout" >/dev/null 2>&1 || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT INT TERM

python3 - "$ADMIN_PASSWORD_FILE" "$ADMIN_USERNAME" <<'PY' |
import json
import pathlib
import sys

text = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8").strip()
values = {}
for line in text.splitlines():
    if "=" in line:
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip()
password = values.get("password", text if "\n" not in text else "")
if not password:
    raise SystemExit("管理员密码文件缺少 password")
print(json.dumps({"username": sys.argv[2], "password": password}))
PY
curl -fsS -c "$COOKIE_JAR" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/auth/login" >"$WORK_DIR/login.json"

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/me" >"$WORK_DIR/me.json"
python3 - "$WORK_DIR/me.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert int(data["data"]["role"]) >= 10
assert int(data["data"]["status"]) == 1
PY

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/admin/users?p=1&page_size=5" >"$WORK_DIR/users.json"
python3 - "$WORK_DIR/users.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert isinstance(data["data"].get("items"), list)
PY

KEY_NAME="contract-$(date +%s)"
printf '{"name":"%s","provider":"OpenAI","routing_policy_ids":[],"expired_time":-1,"remain_quota":0,"unlimited_quota":true,"model_limits_enabled":false,"model_limits":"","allow_ips":null,"group":"default","cross_group_retry":false}' "$KEY_NAME" |
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/keys" >"$WORK_DIR/create-key.json"
python3 - "$WORK_DIR/create-key.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
PY

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/keys?p=1&page_size=100" >"$WORK_DIR/keys.json"
KEY_ID="$(python3 - "$WORK_DIR/keys.json" "$KEY_NAME" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
matches = [item for item in data["data"].get("items", []) if item.get("name") == sys.argv[2]]
assert len(matches) == 1
print(matches[0]["id"])
PY
)"
python3 - "$WORK_DIR/keys.json" "$KEY_ID" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
match = next(item for item in data["data"].get("items", []) if int(item["id"]) == int(sys.argv[2]))
assert match["provider"] == "OpenAI"
assert int(match["model_count"]) == 0
assert match.get("routing_policies") == []
assert "*" in str(match.get("key", ""))
PY

printf '{"enabled":false}' |
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- -X POST "$BASE_URL/api/keys/$KEY_ID/status" >"$WORK_DIR/key-status.json"

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >"$WORK_DIR/delete-key.json"
purge_test_key
KEY_ID=""
KEY_NAME=""

echo "账号契约验证通过"
