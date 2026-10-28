#!/usr/bin/env bash
# Test against staging:
#   1. smoke-test the public URL (Worker -> VM API -> MariaDB)
#   2. run Go integration tests against the staging DB over an SSH tunnel
#
#   STAGING_URL=https://reword-web.<sub>.workers.dev STAGING_SSH=ubuntu@<ip> deploy/staging/test.sh
set -euo pipefail
: "${STAGING_URL:?}" "${STAGING_SSH:?}"
SSH_KEY=${STAGING_SSH_KEY:-$HOME/.ssh/reword_staging}
ROOT=$(cd "$(dirname "$0")/../.." && pwd)

echo "==> Smoke: $STAGING_URL"
for path in /api/healthz /api/readyz; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$STAGING_URL$path")
  echo "$path -> $code"
  [ "$code" = 200 ] || { echo "FAIL"; exit 1; }
done

echo "==> Integration tests over SSH tunnel"
db_pass=$(ssh -i "$SSH_KEY" "$STAGING_SSH" "grep ^MARIADB_PASSWORD= ~/reword/deploy/staging/.env | cut -d= -f2")
ssh -i "$SSH_KEY" -o ExitOnForwardFailure=yes -f -N -L 13306:127.0.0.1:3306 "$STAGING_SSH"
trap 'pkill -f "13306:127.0.0.1:3306" || true' EXIT
cd "$ROOT/api"
DATABASE_DSN="reword:${db_pass}@tcp(127.0.0.1:13306)/reword" go test -count=1 -v ./... 2>&1 | grep -E '^(=== RUN|--- |ok|FAIL|PASS)'
