#!/bin/sh
set -eu
umask 077

BASE_URL="${1:-https://aggregate.lwylink.xyz}"
BASE_URL="${BASE_URL%/}"
ADMIN_USERNAME="${ADMIN_USERNAME:-3180615598@qq.com}"
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

curl -fsS "$BASE_URL/api/models" >"$WORK_DIR/models.json"
curl -fsS "$BASE_URL/api/offers" >"$WORK_DIR/offers.json"
python3 - "$WORK_DIR/models.json" "$WORK_DIR/offers.json" <<'PY'
import json
import sys

models = json.load(open(sys.argv[1], encoding="utf-8"))
offers = json.load(open(sys.argv[2], encoding="utf-8"))
assert models.get("success") is True and isinstance(models.get("data"), list)
assert offers.get("success") is True and isinstance(offers.get("data"), list)
encoded = json.dumps([models, offers], ensure_ascii=False)
assert "internal_alias" not in encoded
assert "mkt-ov-" not in encoded
PY

KEY_NAME="routing-contract-$(date +%s)"
printf '{"name":"%s","expired_time":-1,"remain_quota":0,"unlimited_quota":true,"model_limits_enabled":false,"model_limits":"","allow_ips":null,"group":"default","cross_group_retry":false}' "$KEY_NAME" |
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/keys" >"$WORK_DIR/create-key.json"

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/keys?p=1&page_size=100" >"$WORK_DIR/keys.json"
KEY_ID="$(python3 - "$WORK_DIR/keys.json" "$KEY_NAME" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
match = next(item for item in data["data"]["items"] if item["name"] == sys.argv[2])
print(match["id"])
PY
)"

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/routing-policies/$KEY_ID" >"$WORK_DIR/policy.json"
python3 - "$WORK_DIR/policy.json" "$KEY_ID" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
policy = data["data"]
assert data.get("success") is True
assert int(policy["newapi_token_id"]) == int(sys.argv[2])
assert policy["mode"] == "smart" and policy["priority"] == "stability"
assert int(policy["minimum_multiplier_bps"]) == 0
assert int(policy["maximum_multiplier_bps"]) == 50000
PY

status="$(printf '{"mode":"fixed","priority":"stability","minimum_multiplier_bps":0,"maximum_multiplier_bps":50000,"merchant_ids":[]}' |
  curl -sS -o "$WORK_DIR/invalid-fixed.json" -w '%{http_code}' -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- -X PUT "$BASE_URL/api/routing-policies/$KEY_ID")"
if [ "$status" != "400" ]; then
  echo "空固定商户策略状态码为 $status，预期 400" >&2
  exit 1
fi

printf '{"mode":"smart","priority":"price","minimum_multiplier_bps":0,"maximum_multiplier_bps":50000,"merchant_ids":[]}' |
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- -X PUT "$BASE_URL/api/routing-policies/$KEY_ID" >"$WORK_DIR/saved-policy.json"

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/keys/$KEY_ID/reveal" >"$WORK_DIR/key.json"
MODELROUTE_KEY="$(python3 - "$WORK_DIR/key.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
key = data["data"]["key"]
print(key if key.startswith("sk-") else "sk-" + key)
PY
)"
status="$(printf '{"model":"modelroute-contract-missing","messages":[{"role":"user","content":"ping"}]}' |
  curl -sS -o "$WORK_DIR/model-error.json" -w '%{http_code}' -H "Authorization: Bearer $MODELROUTE_KEY" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/v1/chat/completions")"
if [ "$status" != "503" ]; then
  echo "无模型路由状态码为 $status，预期 503" >&2
  exit 1
fi
python3 - "$WORK_DIR/model-error.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert isinstance(data.get("error"), dict)
assert data["error"].get("code") in {"routing_policy_not_found", "no_available_channel"}
assert "mkt-ov-" not in json.dumps(data, ensure_ascii=False)
PY

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >"$WORK_DIR/delete-key.json"
KEY_ID=""

echo "模型报价与路由契约验证通过"
