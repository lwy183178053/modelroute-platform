#!/bin/sh
set -eu

if [ "${CONFIRM_RESTORE:-}" != "MODELROUTE_RESTORE" ]; then
  echo "必须设置 CONFIRM_RESTORE=MODELROUTE_RESTORE" >&2
  exit 1
fi
if [ "$#" -ne 1 ] || [ ! -d "$1" ]; then
  echo "用法：restore.sh <备份目录>" >&2
  exit 1
fi

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ROOT_DIR"
set -a
. ./.env
set +a

sha256sum -c "$1/SHA256SUMS"
docker compose stop platform-api platform-worker new-api
docker compose exec -T postgres pg_restore --clean --if-exists --no-owner -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_DB_NAME" < "$1/platform.dump"
docker compose exec -T postgres pg_restore --clean --if-exists --no-owner -U "$POSTGRES_SUPERUSER" -d "$NEWAPI_DB_NAME" < "$1/newapi.dump"
docker compose start new-api platform-api platform-worker
