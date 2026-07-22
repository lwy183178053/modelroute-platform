#!/bin/sh
set -eu
umask 077

if [ "${CONFIRM_RESTORE_DRILL:-}" != "MODELROUTE_RESTORE_DRILL" ]; then
  echo "必须设置 CONFIRM_RESTORE_DRILL=MODELROUTE_RESTORE_DRILL" >&2
  exit 1
fi
if [ "$#" -ne 1 ] || [ ! -d "$1" ]; then
  echo "用法：verify-backup-restore.sh <备份目录>" >&2
  exit 1
fi

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ROOT_DIR"
if [ ! -r .env ]; then
  echo "缺少生产环境文件 .env" >&2
  exit 1
fi
set -a
. ./.env
set +a

BACKUP_DIR=$(CDPATH= cd -- "$1" && pwd -P)
for file in platform.dump newapi.dump SHA256SUMS; do
  test -r "$BACKUP_DIR/$file" || { echo "备份缺少 $file" >&2; exit 1; }
done
sha256sum -c "$BACKUP_DIR/SHA256SUMS"

SUFFIX="$(date -u +%Y%m%d%H%M%S)_$$"
PLATFORM_TEST_DB="modelroute_restore_check_$SUFFIX"
NEWAPI_TEST_DB="newapi_restore_check_$SUFFIX"
test "$PLATFORM_TEST_DB" != "$PLATFORM_DB_NAME"
test "$NEWAPI_TEST_DB" != "$NEWAPI_DB_NAME"

dc() {
  docker compose --env-file .env "$@"
}
cleanup() {
  dc exec -T postgres dropdb --if-exists --force -U "$POSTGRES_SUPERUSER" "$PLATFORM_TEST_DB" >/dev/null 2>&1 || true
  dc exec -T postgres dropdb --if-exists --force -U "$POSTGRES_SUPERUSER" "$NEWAPI_TEST_DB" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

dc exec -T postgres createdb -U "$POSTGRES_SUPERUSER" "$PLATFORM_TEST_DB"
dc exec -T postgres createdb -U "$POSTGRES_SUPERUSER" "$NEWAPI_TEST_DB"
dc exec -T postgres pg_restore --exit-on-error --no-owner -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_TEST_DB" < "$BACKUP_DIR/platform.dump"
dc exec -T postgres pg_restore --exit-on-error --no-owner -U "$POSTGRES_SUPERUSER" -d "$NEWAPI_TEST_DB" < "$BACKUP_DIR/newapi.dump"

platform_tables=$(dc exec -T postgres psql -Atq -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_TEST_DB" -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")
newapi_tables=$(dc exec -T postgres psql -Atq -U "$POSTGRES_SUPERUSER" -d "$NEWAPI_TEST_DB" -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'")
platform_migrations=$(dc exec -T postgres psql -Atq -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_TEST_DB" -c "SELECT count(*) FROM platform_schema_migrations")
test "$platform_tables" -gt 0
test "$newapi_tables" -gt 0
test "$platform_migrations" -gt 0

echo "备份恢复演练通过：平台库 $platform_tables 张表，New API 库 $newapi_tables 张表，生产数据库未作为恢复目标。"
