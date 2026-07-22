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
docker compose --env-file .env --profile tools build
docker compose --env-file .env up -d postgres redis new-api
docker compose --env-file .env run --rm model-seed
docker compose --env-file .env up -d --remove-orphans
docker compose --env-file .env ps
