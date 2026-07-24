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
POLICY_ID=""
KEY_ID=""

cleanup() {
  if [ -n "$KEY_ID" ]; then
    curl -sS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >/dev/null 2>&1 || true
  fi
  if [ -n "$POLICY_ID" ]; then
    curl -sS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/routing-policies/$POLICY_ID" >/dev/null 2>&1 || true
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
MODEL_VALUES="$(python3 - "$WORK_DIR/models.json" "$WORK_DIR/offers.json" <<'PY'
import json
import sys

models = json.load(open(sys.argv[1], encoding="utf-8"))
offers = json.load(open(sys.argv[2], encoding="utf-8"))
assert models.get("success") is True and isinstance(models.get("data"), list)
assert offers.get("success") is True and isinstance(offers.get("data"), list)
encoded = json.dumps([models, offers], ensure_ascii=False)
assert "internal_alias" not in encoded
assert "mkt-ov-" not in encoded
providers = {"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Qwen", "Moonshot"}
candidates = [item for item in models["data"] if item.get("status") == "active" and item.get("provider") in providers]
if not candidates:
    raise SystemExit("没有可用于路由契约验证的启用模型")
routable_names = {
    str(item.get("model", ""))
    for item in offers["data"]
    if int(item.get("channel_count", 0) or 0) > 0
}
selected = next((item for item in candidates if item.get("public_name") in routable_names), candidates[0])
expect_routable = 1 if selected.get("public_name") in routable_names else 0
print(f'{int(selected["id"])}|{selected["provider"]}|{selected["public_name"]}|{expect_routable}')
PY
)"
MODEL_ID="${MODEL_VALUES%%|*}"
MODEL_REST="${MODEL_VALUES#*|}"
MODEL_PROVIDER="${MODEL_REST%%|*}"
MODEL_REST="${MODEL_REST#*|}"
MODEL_NAME="${MODEL_REST%%|*}"
EXPECT_ROUTABLE="${MODEL_REST##*|}"

STAMP="$(date +%s)"
POLICY_NAME="routing-contract-$STAMP"
python3 - "$MODEL_ID" "$POLICY_NAME" <<'PY' |
import json
import sys

print(json.dumps({
    "canonical_model_id": int(sys.argv[1]),
    "name": sys.argv[2],
    "priority": "price",
    "minimum_multiplier_bps": 0,
    "maximum_multiplier_bps": 50000,
}))
PY
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/routing-policies" >"$WORK_DIR/create-policy.json"

POLICY_ID="$(python3 - "$WORK_DIR/create-policy.json" "$MODEL_ID" "$MODEL_PROVIDER" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
policy = data["data"]
assert data.get("success") is True
assert int(policy["model_id"]) == int(sys.argv[2])
assert policy["provider"] == sys.argv[3]
assert int(policy["minimum_multiplier_bps"]) == 0
assert int(policy["maximum_multiplier_bps"]) == 50000
print(policy["id"])
PY
)"

status="$(python3 - "$MODEL_ID" "$POLICY_NAME" <<'PY' |
import json
import sys

print(json.dumps({
    "canonical_model_id": int(sys.argv[1]),
    "name": sys.argv[2],
    "priority": "success",
    "minimum_multiplier_bps": 0,
    "maximum_multiplier_bps": 50000,
}))
PY
  curl -sS -o "$WORK_DIR/immutable-model.json" -w '%{http_code}' -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- -X PUT "$BASE_URL/api/routing-policies/$POLICY_ID")"
if [ "$status" != "400" ]; then
  echo "修改路由模型状态码为 $status，预期 400" >&2
  exit 1
fi

python3 - "$POLICY_NAME" <<'PY' |
import json
import sys

print(json.dumps({
    "name": sys.argv[1],
    "priority": "success",
    "minimum_multiplier_bps": 0,
    "maximum_multiplier_bps": 50000,
}))
PY
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- -X PUT "$BASE_URL/api/routing-policies/$POLICY_ID" >"$WORK_DIR/update-policy.json"
python3 - "$WORK_DIR/update-policy.json" "$MODEL_ID" "$MODEL_PROVIDER" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
policy = data["data"]
assert data.get("success") is True
assert int(policy["model_id"]) == int(sys.argv[2])
assert policy["provider"] == sys.argv[3]
assert policy["priority"] == "success"
PY

KEY_NAME="provider-key-contract-$STAMP"
python3 - "$MODEL_PROVIDER" "$POLICY_ID" "$KEY_NAME" <<'PY' |
import json
import sys

print(json.dumps({
    "name": sys.argv[3],
    "expired_time": -1,
    "remain_quota": 0,
    "unlimited_quota": True,
    "model_limits_enabled": False,
    "model_limits": "",
    "allow_ips": None,
    "group": "default",
    "cross_group_retry": False,
    "provider": sys.argv[1],
    "routing_policy_ids": [int(sys.argv[2])],
}))
PY
curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/keys" >"$WORK_DIR/create-key.json"

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/keys?p=1&page_size=200" >"$WORK_DIR/keys.json"
KEY_VALUES="$(python3 - "$WORK_DIR/keys.json" "$KEY_NAME" "$MODEL_PROVIDER" "$MODEL_ID" "$POLICY_ID" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
match = next(item for item in data["data"]["items"] if item["name"] == sys.argv[2])
assert match["provider"] == sys.argv[3]
assert int(match["model_count"]) == 1
routes = match.get("routing_policies") or []
assert len(routes) == 1
assert int(routes[0]["model_id"]) == int(sys.argv[4])
assert int(routes[0]["routing_policy_id"]) == int(sys.argv[5])
key = str(match.get("key", ""))
raw_key = key[3:] if key.startswith("sk-") else key
assert len(raw_key) >= 32 and "*" not in raw_key
print(f'{match["id"]}|{key if key.startswith("sk-") else "sk-" + key}')
PY
)"
KEY_ID="${KEY_VALUES%%|*}"
MODELROUTE_KEY="${KEY_VALUES#*|}"

curl -fsS -H "Authorization: Bearer $MODELROUTE_KEY" "$BASE_URL/v1/models" >"$WORK_DIR/allowed-models.json"
python3 - "$WORK_DIR/allowed-models.json" "$MODEL_NAME" "$EXPECT_ROUTABLE" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
items = data.get("data", []) if isinstance(data, dict) else data
assert isinstance(items, list)
names = [item.get("id") for item in items if isinstance(item, dict)]
assert all(name == sys.argv[2] for name in names)
if sys.argv[3] == "1":
    assert sys.argv[2] in names
encoded = json.dumps(data, ensure_ascii=False)
assert "internal_alias" not in encoded and "mkt-ov-" not in encoded
PY

status="$(printf '{"model":"modelroute-contract-missing","messages":[{"role":"user","content":"ping"}]}' |
  curl -sS -o "$WORK_DIR/model-error.json" -w '%{http_code}' -H "Authorization: Bearer $MODELROUTE_KEY" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/v1/chat/completions")"
if [ "$status" != "403" ]; then
  echo "未绑定模型状态码为 $status，预期 403" >&2
  exit 1
fi
python3 - "$WORK_DIR/model-error.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert isinstance(data.get("error"), dict)
assert data["error"]["code"] == "model_not_allowed"
assert "mkt-ov-" not in json.dumps(data, ensure_ascii=False)
PY

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/routing-policies/$POLICY_ID" >"$WORK_DIR/delete-policy.json"
python3 - "$WORK_DIR/delete-policy.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("success") is True
assert int(data["data"]["affected_key_count"]) >= 1
PY
POLICY_ID=""

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/keys?p=1&page_size=200" >"$WORK_DIR/keys-after-delete.json"
python3 - "$WORK_DIR/keys-after-delete.json" "$KEY_ID" "$MODEL_PROVIDER" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
match = next(item for item in data["data"]["items"] if int(item["id"]) == int(sys.argv[2]))
assert match["provider"] == sys.argv[3]
assert int(match["model_count"]) == 0
assert not (match.get("routing_policies") or [])
PY

curl -fsS -H "Authorization: Bearer $MODELROUTE_KEY" "$BASE_URL/v1/models" >"$WORK_DIR/models-after-delete.json"
python3 - "$WORK_DIR/models-after-delete.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
items = data.get("data", []) if isinstance(data, dict) else data
assert isinstance(items, list) and len(items) == 0
PY

curl -fsS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X DELETE "$BASE_URL/api/keys/$KEY_ID" >"$WORK_DIR/delete-key.json"
KEY_ID=""

echo "单厂商密钥与模型路由契约验证通过"
