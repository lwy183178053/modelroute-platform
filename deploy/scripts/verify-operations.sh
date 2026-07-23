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
cleanup() {
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

status="$(curl -sS -o "$WORK_DIR/anonymous.json" -w '%{http_code}' "$BASE_URL/api/usage")"
test "$status" = "401"

for entry in \
  "usage|/api/usage?p=1&page_size=5" \
  "overview|/api/admin/overview" \
  "channel-logs|/api/admin/channel-logs?p=1&page_size=5" \
  "settings|/api/admin/platform-settings" \
  "redemptions|/api/admin/redemptions?p=1&page_size=5" \
  "channels|/api/admin/channels?p=1&page_size=5"
do
  name=${entry%%|*}
  path=${entry#*|}
  curl -fsS -b "$COOKIE_JAR" "$BASE_URL$path" >"$WORK_DIR/$name.json"
done

python3 - "$WORK_DIR" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
for path in root.glob("*.json"):
    if path.name in {"anonymous.json", "login.json"}:
        continue
    payload = json.loads(path.read_text(encoding="utf-8"))
    assert payload.get("success") is True, f"{path.name}: {payload}"

settings = json.loads((root / "settings.json").read_text(encoding="utf-8"))
encoded = json.dumps(settings, ensure_ascii=False).lower()
for forbidden in ("smtp_password", "smtp_token", "merchant_key", "encryption_key"):
    assert forbidden not in encoded, f"设置接口泄露敏感字段: {forbidden}"
PY

echo "运营管理只读验收通过：用量、总览、设置、兑换码、任务和渠道列表正常。"
