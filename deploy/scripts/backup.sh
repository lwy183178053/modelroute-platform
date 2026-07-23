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

DATA_ROOT=$(readlink -f "$ROOT_DIR/data")
ENV_FILE=$(readlink -f "$ROOT_DIR/.env")
SHARED_ROOT=$(dirname "$ENV_FILE")

docker compose --env-file .env exec -T postgres pg_dump -Fc -U "$POSTGRES_SUPERUSER" "$PLATFORM_DB_NAME" > "$TARGET/platform.dump"
docker compose --env-file .env exec -T postgres pg_dump -Fc -U "$POSTGRES_SUPERUSER" "$NEWAPI_DB_NAME" > "$TARGET/newapi.dump"
docker compose --env-file .env exec -T -e REDISCLI_AUTH="$REDIS_PASSWORD" redis redis-cli SAVE >/dev/null
docker compose --env-file .env cp redis:/data/dump.rdb "$TARGET/redis.rdb" >/dev/null
docker compose --env-file .env exec -T postgres psql -At -U "$POSTGRES_SUPERUSER" -d "$NEWAPI_DB_NAME" -c "SELECT COALESCE(json_object_agg(key, value)::text, '{}') FROM options WHERE key IN ('EmailVerificationEnabled','SMTPAccount','SMTPForceAuthLogin','SMTPFrom','SMTPInsecureSkipVerify','SMTPPort','SMTPSSLEnabled','SMTPServer','SMTPStartTLSEnabled','SMTPToken')" > "$TARGET/newapi-smtp-options.json"
tar -C "$DATA_ROOT" -czf "$TARGET/newapi-data.tar.gz" new-api
cp "$ENV_FILE" "$TARGET/env.backup"
if [ -f "$SHARED_ROOT/bootstrap-admin.txt" ]; then
  cp "$SHARED_ROOT/bootstrap-admin.txt" "$TARGET/bootstrap-admin.txt"
fi
docker compose --env-file .env config > "$TARGET/compose.resolved.yaml"
printf '%s\n' "${APP_VERSION:-unknown}" > "$TARGET/app-version.txt"
test -s "$TARGET/platform.dump"
test -s "$TARGET/newapi.dump"
test -s "$TARGET/redis.rdb"
test -s "$TARGET/newapi-smtp-options.json"
test -s "$TARGET/env.backup"
sha256sum "$TARGET"/* > "$TARGET/SHA256SUMS"
chmod 600 "$TARGET"/*

echo "$TARGET"
