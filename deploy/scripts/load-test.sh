#!/bin/sh
set -eu

BASE_URL="${1:-http://127.0.0.1:28100}"
BASE_URL="${BASE_URL%/}"
REQUESTS=${REQUESTS:-200}
CONCURRENCY=${CONCURRENCY:-10}
MAX_REQUESTS=10000
MAX_CONCURRENCY=200
MAX_P95_SECONDS=${MAX_P95_SECONDS:-1.000}

case "$REQUESTS" in ''|*[!0-9]*) echo "REQUESTS 必须是正整数" >&2; exit 1;; esac
case "$CONCURRENCY" in ''|*[!0-9]*) echo "CONCURRENCY 必须是正整数" >&2; exit 1;; esac
test "$REQUESTS" -ge 1 && test "$REQUESTS" -le "$MAX_REQUESTS" || { echo "REQUESTS 范围为 1..$MAX_REQUESTS" >&2; exit 1; }
test "$CONCURRENCY" -ge 1 && test "$CONCURRENCY" -le "$MAX_CONCURRENCY" || { echo "CONCURRENCY 范围为 1..$MAX_CONCURRENCY" >&2; exit 1; }

RESULTS=$(mktemp)
trap 'rm -f "$RESULTS"' EXIT INT TERM
export BASE_URL
set +e
seq "$REQUESTS" | xargs -n 1 -P "$CONCURRENCY" sh -c '
  curl -sS -o /dev/null --connect-timeout 3 --max-time 10 \
    -w "%{http_code} %{time_total}\n" "$BASE_URL/api/health/ready"
' _ >"$RESULTS"
xargs_status=$?
set -e

count=$(wc -l <"$RESULTS" | tr -d ' ')
success=$(awk '$1 == 200 { count++ } END { print count + 0 }' "$RESULTS")
failed=$((REQUESTS - success))
average=$(awk '{ total += $2 } END { if (NR) printf "%.4f", total / NR; else print "0" }' "$RESULTS")
p95_index=$(( (REQUESTS * 95 + 99) / 100 ))
p95=$(sort -k2,2n "$RESULTS" | awk -v target="$p95_index" 'NR == target { print $2 }')

test "$xargs_status" -eq 0
test "$count" -eq "$REQUESTS"
test "$failed" -eq 0
awk -v actual="$p95" -v maximum="$MAX_P95_SECONDS" 'BEGIN { exit !(actual <= maximum) }'
echo "只读压力测试通过：请求 $REQUESTS，并发 $CONCURRENCY，成功 $success，平均 ${average}s，P95 ${p95}s。"
