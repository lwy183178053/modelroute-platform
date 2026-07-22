#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ROOT_DIR"

if [ ! -f .env ]; then
  echo "缺少 .env" >&2
  exit 1
fi

set -a
. ./.env
set +a

BACKUP_ROOT=${BACKUP_ROOT:-"$ROOT_DIR/backups"}
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
TARGET="$BACKUP_ROOT/$STAMP"
umask 077
mkdir -p "$TARGET"

docker compose exec -T postgres pg_dump -Fc -U "$POSTGRES_SUPERUSER" "$PLATFORM_DB_NAME" > "$TARGET/platform.dump"
docker compose exec -T postgres pg_dump -Fc -U "$POSTGRES_SUPERUSER" "$NEWAPI_DB_NAME" > "$TARGET/newapi.dump"
docker compose config > "$TARGET/compose.resolved.yaml"
printf '%s\n' "$APP_VERSION" > "$TARGET/app-version.txt"
sha256sum "$TARGET"/* > "$TARGET/SHA256SUMS"

echo "$TARGET"
