#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -P -- "$(dirname -- "$0")/../.." && pwd -P)
cd "$ROOT_DIR"

if [ ! -f .env ]; then
  echo "缺少生产环境文件 .env" >&2
  exit 1
fi

APP_VERSION=$(basename "$ROOT_DIR")
export APP_VERSION

docker compose --env-file .env --profile tools config --quiet

if [ -f release-images.env ]; then
  test "$(wc -l < release-images.env | tr -d ' ')" = "3"
  grep -Eq '^RELEASE_VERSION=[0-9a-f]{12}$' release-images.env
  grep -Eq '^PLATFORM_IMAGE=ghcr\.io/lwy183178053/modelroute-platform-api@sha256:[0-9a-f]{64}$' release-images.env
  grep -Eq '^GATEWAY_IMAGE=ghcr\.io/lwy183178053/modelroute-platform-gateway@sha256:[0-9a-f]{64}$' release-images.env
  . ./release-images.env
  test "$RELEASE_VERSION" = "$APP_VERSION"
  export PLATFORM_IMAGE GATEWAY_IMAGE
  docker compose --env-file .env --profile tools config --quiet
  docker compose --env-file .env --profile tools pull platform-api platform-worker model-seed gateway
else
  docker compose --env-file .env --profile tools build
fi
docker compose --env-file .env up -d postgres redis new-api

HAS_PLATFORM_MIGRATIONS=$(docker compose --env-file .env exec -T postgres sh -c "psql -At -U \"\$POSTGRES_USER\" -d \"\$PLATFORM_DB_NAME\" -c \"SELECT CASE WHEN to_regclass('public.platform_schema_migrations') IS NULL THEN 0 ELSE 1 END\"")
if [ "$HAS_PLATFORM_MIGRATIONS" = "1" ]; then
  LEGACY_TOKEN_IDS=$(docker compose --env-file .env exec -T postgres sh -c "psql -At -U \"\$POSTGRES_USER\" -d \"\$PLATFORM_DB_NAME\" -c \"SELECT CASE WHEN EXISTS (SELECT 1 FROM platform_schema_migrations WHERE version = 11) THEN '__MIGRATED__' ELSE COALESCE((SELECT string_agg(newapi_token_id::text, ',') FROM token_bindings), '') END\"")
  case "$LEGACY_TOKEN_IDS" in
    __MIGRATED__|'') ;;
    *[!0-9,]*) echo "旧用户密钥编号格式无效" >&2; exit 1 ;;
    *)
      docker compose --env-file .env exec -T postgres sh -c "psql -v ON_ERROR_STOP=1 -U \"\$POSTGRES_USER\" -d \"\$NEWAPI_DB_NAME\" -c \"DELETE FROM tokens WHERE id IN ($LEGACY_TOKEN_IDS)\""
      ;;
  esac
fi

docker compose --env-file .env run --rm model-seed
docker compose --env-file .env up -d --remove-orphans
docker compose --env-file .env ps
