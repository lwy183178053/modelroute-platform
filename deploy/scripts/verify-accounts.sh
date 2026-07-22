#!/bin/sh
set -eu

BASE_URL="${1:-https://aggregate.lwylink.xyz}"
BASE_URL="${BASE_URL%/}"
ADMIN_USERNAME="${ADMIN_USERNAME:-modelroute}"
ADMIN_PASSWORD_FILE="${ADMIN_PASSWORD_FILE:-/opt/modelroute-platform/shared/bootstrap-admin.txt}"

if [ ! -r "$ADMIN_PASSWORD_FILE" ]; then
  echo "管理员密码文件不可读" >&2
  exit 1
fi

WORK_DIR="$(mktemp -d)"
COOKIE_JAR="$WORK_DIR/cookies.txt"
KEY_ID=""

cleanup() {
  if [ -n "$KEY_ID" ]; then
    curl -sS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >/dev/null 2>&1 || true
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
printf '{"name":"%s","expired_time":-1,"remain_quota":0,"unlimited_quota":true,"model_limits_enabled":false,"model_limits":"","allow_ips":null,"group":"default","cross_group_retry":false}' "$KEY_NAME" |
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/keys" >"$WORK_DIR/create-key.json"

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

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/keys/$KEY_ID/reveal" >"$WORK_DIR/reveal-key.json"
python3 - "$WORK_DIR/reveal-key.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert str(data["data"].get("key", "")).startswith("sk-")
PY

printf '{"enabled":false}' |
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- -X POST "$BASE_URL/api/keys/$KEY_ID/status" >"$WORK_DIR/key-status.json"

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >"$WORK_DIR/delete-key.json"
KEY_ID=""

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/admin/audit-logs?p=1&page_size=5" >"$WORK_DIR/audit.json"
python3 - "$WORK_DIR/audit.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert isinstance(data["data"].get("items"), list)
PY

echo "账号契约验证通过"
