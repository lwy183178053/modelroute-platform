#!/bin/sh
set -eu
umask 077

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

cleanup() {
  curl -sS -b "$COOKIE_JAR" -H "Origin: $BASE_URL" -X POST "$BASE_URL/api/auth/logout" >/dev/null 2>&1 || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT INT TERM

status="$(curl -sS -o "$WORK_DIR/anonymous-admin.json" -w '%{http_code}' "$BASE_URL/api/admin/payment-settings")"
test "$status" = "401"

python3 - "$ADMIN_PASSWORD_FILE" "$ADMIN_USERNAME" <<'PY' |
import json
import pathlib
import sys

text = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8").strip()
values = {}
for line in text.splitlines():
    if "=" in line:
        name, value = line.split("=", 1)
        values[name.strip()] = value.strip()
password = values.get("password", text if "\n" not in text else "")
if not password:
    raise SystemExit("管理员密码文件缺少 password")
print(json.dumps({"username": sys.argv[2], "password": password}))
PY
curl -fsS -c "$COOKIE_JAR" -H 'Content-Type: application/json' --data-binary @- "$BASE_URL/api/auth/login" >"$WORK_DIR/login.json"

for endpoint in /api/payment-settings /api/admin/payment-settings; do
  status="$(curl -sS -o "$WORK_DIR/setting.json" -w '%{http_code}' -b "$COOKIE_JAR" "$BASE_URL$endpoint")"
  if [ "$status" != "200" ] && [ "$status" != "404" ]; then
    echo "$endpoint 状态码为 $status，预期 200 或 404" >&2
    exit 1
  fi
done

status="$(curl -sS -o "$WORK_DIR/invalid-notify.txt" -w '%{http_code}' "$BASE_URL/api/payments/easypay/notify?sign=invalid")"
test "$status" = "400"

return_page="$(curl -fsS "$BASE_URL/payments/return?out_trade_no=contract-missing")"
printf '%s' "$return_page" | grep -q '<title>ModelRoute'

echo "易支付充值契约验证通过：权限、返回页和无效回调拦截正常。"
