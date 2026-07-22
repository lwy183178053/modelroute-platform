#!/bin/sh
set -eu

if [ "${CONFIRM_ROLLBACK:-}" != "MODELROUTE_ROLLBACK" ]; then
  echo "必须设置 CONFIRM_ROLLBACK=MODELROUTE_ROLLBACK" >&2
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "用法：rollback-release.sh <发布目录名>" >&2
  exit 1
fi
case "$1" in
  *[!0-9a-f]*|??????|?????|????|???|??|?|"")
    echo "发布目录名必须是 7 至 40 位小写 Git 十六进制哈希" >&2
    exit 1
    ;;
esac
if [ "${#1}" -gt 40 ]; then
  echo "发布目录名必须是 7 至 40 位小写 Git 十六进制哈希" >&2
  exit 1
fi

SCRIPT_ROOT=$(CDPATH= cd -P -- "$(dirname -- "$0")/../.." && pwd -P)
PLATFORM_ROOT=${MODELROUTE_ROOT:-$(CDPATH= cd -- "$SCRIPT_ROOT/../.." && pwd -P)}
RELEASES_DIR=$(CDPATH= cd -- "$PLATFORM_ROOT/releases" && pwd -P)
CURRENT_LINK="$PLATFORM_ROOT/current"
TARGET="$RELEASES_DIR/$1"
BASE_URL=${ROLLBACK_BASE_URL:-http://127.0.0.1:28100}

if [ ! -d "$TARGET" ] || [ -L "$TARGET" ]; then
  echo "目标发布目录不存在或不是实体目录" >&2
  exit 1
fi
TARGET_REAL=$(CDPATH= cd -P -- "$TARGET" && pwd -P)
if [ "$(dirname -- "$TARGET_REAL")" != "$RELEASES_DIR" ] || [ "$(basename -- "$TARGET_REAL")" != "$1" ]; then
  echo "目标发布目录未通过 releases 边界校验" >&2
  exit 1
fi
for file in .env compose.yaml deploy/scripts/deploy.sh deploy/scripts/verify.sh; do
  test -r "$TARGET_REAL/$file" || { echo "目标发布缺少 $file" >&2; exit 1; }
done
if [ ! -L "$CURRENT_LINK" ]; then
  echo "current 不是有效软链接" >&2
  exit 1
fi
PREVIOUS=$(readlink -f "$CURRENT_LINK")
test -d "$PREVIOUS" || { echo "当前发布目录不存在" >&2; exit 1; }

if ! "$TARGET_REAL/deploy/scripts/deploy.sh" || ! "$TARGET_REAL/deploy/scripts/verify.sh" "$BASE_URL"; then
  "$PREVIOUS/deploy/scripts/deploy.sh" >/dev/null 2>&1 || true
  echo "目标版本部署或验收失败，已尝试恢复当前发布" >&2
  exit 1
fi
version=$(curl -fsS "$BASE_URL/api/version")
printf '%s' "$version" | grep -q "\"version\":\"$1\"" || {
  "$PREVIOUS/deploy/scripts/deploy.sh" >/dev/null 2>&1 || true
  echo "线上版本与目标发布目录不一致，已尝试恢复当前发布" >&2
  exit 1
}

TEMP_LINK="$PLATFORM_ROOT/.current.rollback.$$"
trap 'rm -f "$TEMP_LINK"' EXIT INT TERM
ln -s "releases/$1" "$TEMP_LINK"
mv -Tf "$TEMP_LINK" "$CURRENT_LINK"
echo "版本回滚完成：$1"
