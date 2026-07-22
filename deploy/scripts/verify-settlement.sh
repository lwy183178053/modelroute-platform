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

status="$(curl -sS -o "$WORK_DIR/anonymous.json" -w '%{http_code}' "$BASE_URL/api/admin/withdrawals")"
test "$status" = "401"

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

curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/admin/withdrawals?page_size=10" >"$WORK_DIR/withdrawals.json"
curl -fsS -b "$COOKIE_JAR" "$BASE_URL/api/admin/settlement-reconciliation?status=open&page_size=10" >"$WORK_DIR/reconciliation.json"

python3 - "$WORK_DIR/withdrawals.json" "$WORK_DIR/reconciliation.json" <<'PY'
import json
import sys

for path in sys.argv[1:]:
    payload = json.load(open(path, encoding="utf-8"))
    assert payload.get("success") is True
    data = payload.get("data")
    assert isinstance(data, dict) and isinstance(data.get("items"), list)
    assert "encrypted_recipient_account" not in json.dumps(payload, ensure_ascii=False)
PY

status="$(curl -sS -o "$WORK_DIR/balance.json" -w '%{http_code}' -b "$COOKIE_JAR" "$BASE_URL/api/merchant/balance")"
if [ "$status" != "200" ] && [ "$status" != "404" ]; then
  echo "管理员商户余额接口状态码为 $status，预期 200 或 404" >&2
  exit 1
fi

status="$(printf '{"reason":"contract csrf check"}' |
  curl -sS -o "$WORK_DIR/csrf.json" -w '%{http_code}' -b "$COOKIE_JAR" -H 'Content-Type: application/json' --data-binary @- -X POST "$BASE_URL/api/admin/withdrawals/1/approve")"
test "$status" = "403"

echo "结算与人工提现契约验证通过：权限、只读列表、敏感字段和同源拦截正常。"
