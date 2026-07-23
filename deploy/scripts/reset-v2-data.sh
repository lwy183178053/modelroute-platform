#!/bin/sh
set -eu
umask 077

if [ "${CONFIRM_V2_RESET:-}" != "MODELROUTE_V2_RESET_103" ]; then
  echo "必须设置 CONFIRM_V2_RESET=MODELROUTE_V2_RESET_103" >&2
  exit 1
fi

TARGET_IP=38.34.191.103
if ! hostname -I | tr ' ' '\n' | grep -Fxq "$TARGET_IP"; then
  echo "当前主机未绑定目标地址 $TARGET_IP" >&2
  exit 1
fi

ROOT_DIR=$(CDPATH= cd -P -- "$(dirname -- "$0")/../.." && pwd -P)
case "$ROOT_DIR" in
  /opt/modelroute-platform/releases/*) ;;
  *) echo "发布目录不在 ModelRoute releases 范围内" >&2; exit 1 ;;
esac
RELEASE_NAME=$(basename "$ROOT_DIR")
if ! printf '%s' "$RELEASE_NAME" | grep -Eq '^[0-9a-f]{7,40}$'; then
  echo "发布目录名不是 Git 哈希" >&2
  exit 1
fi

cd "$ROOT_DIR"
test -f .env
test "$(readlink -f .env)" = "/opt/modelroute-platform/shared/.env"
test "$(readlink -f data)" = "/opt/modelroute-platform/shared/data"
test "$(readlink -f backups)" = "/opt/modelroute-platform/shared/backups"

set -a
. ./.env
set +a
test "$COMPOSE_PROJECT_NAME" = "modelroute"
APP_VERSION="$RELEASE_NAME"
export APP_VERSION

BACKUP_DIR=$(BACKUP_ROOT=/opt/modelroute-platform/shared/backups deploy/scripts/backup.sh)
BACKUP_DIR=$(readlink -f "$BACKUP_DIR")
case "$BACKUP_DIR" in
  /opt/modelroute-platform/shared/backups/*) ;;
  *) echo "备份目录超出 ModelRoute shared/backups" >&2; exit 1 ;;
esac
(cd "$BACKUP_DIR" && sha256sum -c SHA256SUMS)
CONFIRM_RESTORE_DRILL=MODELROUTE_RESTORE_DRILL deploy/scripts/verify-backup-restore.sh "$BACKUP_DIR"

docker compose --env-file .env down --remove-orphans

DATA_ROOT=/opt/modelroute-platform/shared/data
for name in postgres redis new-api new-api-logs; do
  target="$DATA_ROOT/$name"
  test "$(readlink -f "$target")" = "$target"
  find "$target" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
done

docker compose --env-file .env --profile tools config --quiet
docker compose --env-file .env --profile tools build
docker compose --env-file .env up -d --wait postgres redis new-api

NEW_API_CONTAINER=$(docker compose --env-file .env ps -q new-api)
NEW_API_IP=$(docker inspect -f '{{with index .NetworkSettings.Networks "modelroute_backend"}}{{.IPAddress}}{{end}}' "$NEW_API_CONTAINER")
test -n "$NEW_API_IP"
python3 deploy/scripts/bootstrap-newapi.py "http://$NEW_API_IP:3000" "$BACKUP_DIR/newapi-smtp-options.json"

docker compose --env-file .env --profile tools run --rm model-seed
docker compose --env-file .env up -d --wait --remove-orphans
deploy/scripts/verify.sh http://127.0.0.1:28100

MIGRATIONS=$(docker compose --env-file .env exec -T postgres psql -At -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_DB_NAME" -c "SELECT name FROM platform_schema_migrations ORDER BY name")
test "$MIGRATIONS" = "000001_v2_baseline.sql"
SEED_VERSION=$(docker compose --env-file .env exec -T postgres psql -At -U "$POSTGRES_SUPERUSER" -d "$PLATFORM_DB_NAME" -c "SELECT value FROM platform_runtime_metadata WHERE key='official_model_seed_version'")
test "$SEED_VERSION" = "2026-07-23-v2"

printf 'V2 数据重置与基础验收完成，回滚备份：%s\n' "$BACKUP_DIR"
