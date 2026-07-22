#!/bin/sh
set -eu
umask 077

BASE_URL="${1:-https://aggregate.lwylink.xyz}"
BASE_URL="${BASE_URL%/}"
ADMIN_USERNAME="${ADMIN_USERNAME:-modelroute}"
ADMIN_PASSWORD_FILE="${ADMIN_PASSWORD_FILE:-/opt/modelroute-platform/shared/bootstrap-admin.txt}"
ROOT_DIR=$(CDPATH= cd -P -- "$(dirname -- "$0")/../.." && pwd -P)
cd "$ROOT_DIR"

if [ ! -r "$ADMIN_PASSWORD_FILE" ] || [ ! -r .env ]; then
  echo "管理员密码文件或生产环境文件不可读" >&2
  exit 1
fi

WORK_DIR="$(mktemp -d)"
ADMIN_COOKIE_JAR="$WORK_DIR/admin-cookies.txt"
USER_COOKIE_JAR="$WORK_DIR/user-cookies.txt"
NEW_API_COOKIE_JAR="$WORK_DIR/new-api-cookies.txt"
APPLICATION_ID=""
TEST_USER_ID=""
NEW_API_ADMIN_ID=""
NEW_API_BASE_URL=""

cleanup() {
  if [ -n "$APPLICATION_ID" ]; then
    printf '{"reason":"契约测试中断清理"}' |
      curl -sS -b "$ADMIN_COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' \
        --data-binary @- -X POST "$BASE_URL/api/admin/merchant-applications/$APPLICATION_ID/reject" >/dev/null 2>&1 || true
  fi
  curl -sS -b "$USER_COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/auth/logout" >/dev/null 2>&1 || true
  curl -sS -b "$ADMIN_COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/auth/logout" >/dev/null 2>&1 || true
  if [ -n "$TEST_USER_ID" ] && [ -n "$NEW_API_ADMIN_ID" ] && [ -n "$NEW_API_BASE_URL" ]; then
    curl -sS -b "$NEW_API_COOKIE_JAR" -H "New-Api-User: $NEW_API_ADMIN_ID" \
      -X DELETE "$NEW_API_BASE_URL/api/user/$TEST_USER_ID" >/dev/null 2>&1 || true
  fi
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT INT TERM

assert_status() {
  expected="$1"
  path="$2"
  actual="$(curl -sS -o "$WORK_DIR/status.json" -w '%{http_code}' "$BASE_URL$path")"
  if [ "$actual" != "$expected" ]; then
    echo "$path 状态码为 $actual，预期 $expected" >&2
    exit 1
  fi
}

assert_status 401 /api/merchant/application
assert_status 401 /api/admin/merchant-applications

python3 - "$ADMIN_PASSWORD_FILE" "$ADMIN_USERNAME" "$WORK_DIR/admin-login.json" <<'PY'
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
with open(sys.argv[3], "w", encoding="utf-8") as output:
    json.dump({"username": sys.argv[2], "password": password}, output)
PY
curl -fsS -c "$ADMIN_COOKIE_JAR" -H 'Content-Type: application/json' \
  --data-binary @"$WORK_DIR/admin-login.json" "$BASE_URL/api/auth/login" >"$WORK_DIR/login.json"

curl -fsS -b "$ADMIN_COOKIE_JAR" "$BASE_URL/api/me" >"$WORK_DIR/me.json"
python3 - "$WORK_DIR/me.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert int(data["data"]["role"]) >= 10
assert int(data["data"]["status"]) == 1
assert data["data"].get("is_merchant") is True
assert data["data"].get("is_platform_merchant") is True
PY

NEW_API_CONTAINER_ID="$(docker compose --env-file .env ps -q new-api)"
test -n "$NEW_API_CONTAINER_ID"
docker inspect "$NEW_API_CONTAINER_ID" >"$WORK_DIR/new-api-inspect.json"
NEW_API_BASE_URL="$(python3 - "$WORK_DIR/new-api-inspect.json" <<'PY'
import json
import sys

networks = json.load(open(sys.argv[1], encoding="utf-8"))[0]["NetworkSettings"]["Networks"]
addresses = [value["IPAddress"] for name, value in networks.items() if name.endswith("_backend")]
assert len(addresses) == 1 and addresses[0]
print("http://" + addresses[0] + ":3000")
PY
)"

curl -fsS -c "$NEW_API_COOKIE_JAR" -H 'Content-Type: application/json' \
  --data-binary @"$WORK_DIR/admin-login.json" "$NEW_API_BASE_URL/api/user/login" >"$WORK_DIR/new-api-login.json"
NEW_API_ADMIN_ID="$(python3 - "$WORK_DIR/new-api-login.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert int(data["data"]["role"]) >= 10
print(data["data"]["id"])
PY
)"

TEST_USERNAME="$(python3 - "$WORK_DIR/user-create.json" "$WORK_DIR/user-login.json" <<'PY'
import json
import secrets
import sys

username = "mrc" + secrets.token_hex(6)
password = secrets.token_urlsafe(12)
with open(sys.argv[1], "w", encoding="utf-8") as output:
    json.dump({"username": username, "password": password, "display_name": "Merchant Contract", "role": 1}, output)
with open(sys.argv[2], "w", encoding="utf-8") as output:
    json.dump({"username": username, "password": password}, output)
print(username)
PY
)"

curl -fsS -b "$NEW_API_COOKIE_JAR" -H "New-Api-User: $NEW_API_ADMIN_ID" -H 'Content-Type: application/json' \
  --data-binary @"$WORK_DIR/user-create.json" "$NEW_API_BASE_URL/api/user/" >"$WORK_DIR/create-user.json"
python3 - "$WORK_DIR/create-user.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
PY

curl -fsS -b "$NEW_API_COOKIE_JAR" -H "New-Api-User: $NEW_API_ADMIN_ID" \
  "$NEW_API_BASE_URL/api/user/search?keyword=$TEST_USERNAME&p=1&page_size=10" >"$WORK_DIR/search-user.json"
TEST_USER_ID="$(python3 - "$WORK_DIR/search-user.json" "$TEST_USERNAME" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
matches = [item for item in data["data"].get("items", []) if item.get("username") == sys.argv[2]]
assert len(matches) == 1
assert int(matches[0]["role"]) == 1
print(matches[0]["id"])
PY
)"

curl -fsS -c "$USER_COOKIE_JAR" -H 'Content-Type: application/json' \
  --data-binary @"$WORK_DIR/user-login.json" "$BASE_URL/api/auth/login" >"$WORK_DIR/user-platform-login.json"
curl -fsS -b "$USER_COOKIE_JAR" "$BASE_URL/api/me" >"$WORK_DIR/user-me.json"
python3 - "$WORK_DIR/user-me.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
me = data["data"]
assert int(me["role"]) == 1
assert int(me["status"]) == 1
assert me.get("is_merchant") is False
assert me.get("is_platform_merchant") is False
PY

python3 - "$WORK_DIR/application.json" <<'PY'
import json
import secrets
import sys
import time

stamp = int(time.time())
payload = {
    "display_name": f"ModelRoute 契约测试 {stamp}",
    "website_url": "https://example.com",
    "business_description": "用于验证商户申请、权限与密钥生命周期的自动契约测试。",
    "channel_name": f"契约测试线路 {stamp}",
    "channel_type": 1,
    "base_url": "https://api.example.com/v1",
    "api_key": "sk-contract-" + secrets.token_hex(24),
    "models": ["gpt-4o-mini"],
}
with open(sys.argv[1], "w", encoding="utf-8") as output:
    json.dump(payload, output, ensure_ascii=False)
PY

curl -fsS -b "$USER_COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' \
  --data-binary @"$WORK_DIR/application.json" "$BASE_URL/api/merchant/application" >"$WORK_DIR/create-application.json"
APPLICATION_ID="$(python3 - "$WORK_DIR/create-application.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
encoded = json.dumps(data, ensure_ascii=False)
assert "api_key" not in encoded
assert "encrypted_credential" not in encoded
assert data["data"]["status"] == "pending"
assert data["data"]["initial_channel"]["credential_configured"] is True
print(data["data"]["id"])
PY
)"

curl -fsS -b "$USER_COOKIE_JAR" "$BASE_URL/api/merchant/application" >"$WORK_DIR/application-state.json"
python3 - "$WORK_DIR/application-state.json" "$APPLICATION_ID" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert int(data["data"]["id"]) == int(sys.argv[2])
assert data["data"]["status"] == "pending"
PY

curl -fsS -b "$ADMIN_COOKIE_JAR" "$BASE_URL/api/admin/merchant-applications?p=1&page_size=100" >"$WORK_DIR/applications.json"
curl -fsS -b "$ADMIN_COOKIE_JAR" "$BASE_URL/api/admin/channel-reviews?p=1&page_size=100" >"$WORK_DIR/reviews.json"
curl -fsS -b "$ADMIN_COOKIE_JAR" "$BASE_URL/api/admin/channels?p=1&page_size=100" >"$WORK_DIR/channels.json"
python3 - "$WORK_DIR/applications.json" "$WORK_DIR/reviews.json" "$WORK_DIR/channels.json" "$APPLICATION_ID" <<'PY'
import json
import sys

applications = json.load(open(sys.argv[1], encoding="utf-8"))
reviews = json.load(open(sys.argv[2], encoding="utf-8"))
channels = json.load(open(sys.argv[3], encoding="utf-8"))
application_id = int(sys.argv[4])
assert applications.get("success") is True
match = next(item for item in applications["data"]["items"] if int(item["id"]) == application_id)
review_id = int(match["initial_channel"]["id"])
assert any(int(item["id"]) == review_id for item in reviews["data"]["items"])
assert channels.get("success") is True
assert isinstance(channels["data"].get("items"), list)
encoded = json.dumps([applications, reviews, channels], ensure_ascii=False)
assert "encrypted_credential" not in encoded
assert "api_key" not in encoded
PY

printf '{"reason":"自动契约测试完成"}' |
curl -fsS -b "$ADMIN_COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' \
  --data-binary @- -X POST "$BASE_URL/api/admin/merchant-applications/$APPLICATION_ID/reject" >"$WORK_DIR/reject.json"

curl -fsS -b "$USER_COOKIE_JAR" "$BASE_URL/api/merchant/application" >"$WORK_DIR/rejected-state.json"
python3 - "$WORK_DIR/reject.json" "$WORK_DIR/rejected-state.json" "$APPLICATION_ID" <<'PY'
import json
import sys

rejected = json.load(open(sys.argv[1], encoding="utf-8"))
state = json.load(open(sys.argv[2], encoding="utf-8"))
assert rejected.get("success") is True
assert int(rejected["data"]["review_id"]) > 0
assert int(state["data"]["id"]) == int(sys.argv[3])
assert state["data"]["status"] == "rejected"
assert state["data"]["initial_channel"]["credential_configured"] is False
PY
APPLICATION_ID=""

curl -fsS -b "$USER_COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/auth/logout" >/dev/null
curl -fsS -b "$NEW_API_COOKIE_JAR" -H "New-Api-User: $NEW_API_ADMIN_ID" \
  -X DELETE "$NEW_API_BASE_URL/api/user/$TEST_USER_ID" >"$WORK_DIR/delete-user.json"
python3 - "$WORK_DIR/delete-user.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
PY
TEST_USER_ID=""

echo "商户与渠道契约验证通过"
