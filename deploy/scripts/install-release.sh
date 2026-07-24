#!/bin/sh
set -eu

if [ "$#" -ne 4 ]; then
  echo "用法：install-release.sh <12位版本> <归档> <镜像清单> <归档SHA256>" >&2
  exit 1
fi

VERSION=$1
ARCHIVE=$2
IMAGE_MANIFEST=$3
EXPECTED_SHA=$4
BASE=${MODELROUTE_RELEASE_BASE:-/opt/modelroute-platform}
if [ "$BASE" != "/opt/modelroute-platform" ] && [ "${MODELROUTE_RELEASE_TESTING:-}" != "1" ]; then
  echo "发布根目录只允许在测试模式覆盖" >&2
  exit 1
fi
RELEASES=$BASE/releases
TARGET=$RELEASES/$VERSION
CURRENT=$BASE/current
SHARED=$BASE/shared

case "$VERSION" in
  *[!0-9a-f]*|'') echo "发布版本格式无效" >&2; exit 1 ;;
esac
test "${#VERSION}" -eq 12
case "$EXPECTED_SHA" in
  *[!0-9a-f]*|'') echo "归档摘要格式无效" >&2; exit 1 ;;
esac
test "${#EXPECTED_SHA}" -eq 64
test -f "$ARCHIVE"
test -f "$IMAGE_MANIFEST"
test -d "$RELEASES"
test -f "$SHARED/.env"
test -d "$SHARED/data"
test -d "$SHARED/backups"
test ! -e "$TARGET"

ACTUAL_SHA=$(sha256sum "$ARCHIVE" | awk '{print $1}')
test "$ACTUAL_SHA" = "$EXPECTED_SHA"
if tar -tf "$ARCHIVE" | grep -Eq '(^/|(^|/)\.\.(/|$))'; then
  echo "发布归档包含越界路径" >&2
  exit 1
fi

PREVIOUS=$(readlink -f "$CURRENT")
case "$PREVIOUS" in
  "$RELEASES"/*) ;;
  *) echo "当前发布指针无效" >&2; exit 1 ;;
esac

SUCCESS=0
cleanup() {
  status=$?
  trap - 0 INT TERM
  rm -f "$ARCHIVE" "$IMAGE_MANIFEST" 2>/dev/null || true
  if [ "$SUCCESS" -ne 1 ]; then
    echo "新版本验收失败，恢复上一发布：$PREVIOUS" >&2
    ln -s "$PREVIOUS" "$BASE/current.rollback-$VERSION" 2>/dev/null || true
    mv -Tf "$BASE/current.rollback-$VERSION" "$CURRENT" 2>/dev/null || true
    if [ -x "$PREVIOUS/deploy/scripts/deploy.sh" ]; then
      (cd "$PREVIOUS" && ./deploy/scripts/deploy.sh) || true
    fi
  fi
  exit "$status"
}
trap cleanup 0
trap 'exit 130' INT TERM

mkdir "$TARGET"
tar -xf "$ARCHIVE" -C "$TARGET"
cp "$IMAGE_MANIFEST" "$TARGET/release-images.env"
ln -s "$SHARED/.env" "$TARGET/.env"
ln -s "$SHARED/data" "$TARGET/data"
ln -s "$SHARED/backups" "$TARGET/backups"

cd "$TARGET"
./deploy/scripts/deploy.sh
./deploy/scripts/verify.sh http://127.0.0.1:28100

ln -s "$TARGET" "$BASE/current.next-$VERSION"
mv -Tf "$BASE/current.next-$VERSION" "$CURRENT"
./deploy/scripts/verify.sh https://aggregate.lwylink.xyz

export ADMIN_PASSWORD_FILE="$SHARED/bootstrap-admin.txt"
./deploy/scripts/verify-accounts.sh https://aggregate.lwylink.xyz
./deploy/scripts/verify-merchants.sh https://aggregate.lwylink.xyz
./deploy/scripts/verify-routing.sh https://aggregate.lwylink.xyz
./deploy/scripts/verify-payments.sh https://aggregate.lwylink.xyz
./deploy/scripts/verify-withdrawals.sh https://aggregate.lwylink.xyz
./deploy/scripts/verify-operations.sh https://aggregate.lwylink.xyz
REQUESTS=200 CONCURRENCY=10 ./deploy/scripts/load-test.sh http://127.0.0.1:28100

SUCCESS=1
echo "ModelRoute 发布完成：$VERSION，上一版本：$(basename "$PREVIOUS")"
