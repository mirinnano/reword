#!/usr/bin/env bash
# Run the Go API locally against the staging database (no local MariaDB needed).
# Opens an SSH tunnel to the VM and reads the DB password from it at start-up,
# so no secret is stored in the repo.
#   STAGING_SSH=ubuntu@<ip> scripts/dev-api-staging-db.sh
set -euo pipefail
: "${STAGING_SSH:?set STAGING_SSH=user@host}"
KEY=${STAGING_SSH_KEY:-$HOME/.ssh/reword_staging}
PORT=13306

pkill -f "$PORT:127.0.0.1:3306" 2>/dev/null || true
ssh -i "$KEY" -o ExitOnForwardFailure=yes -f -N -L "$PORT:127.0.0.1:3306" "$STAGING_SSH"
trap 'pkill -f "$PORT:127.0.0.1:3306" || true' EXIT
pw=$(ssh -i "$KEY" "$STAGING_SSH" "grep ^MARIADB_PASSWORD= ~/reword/deploy/staging/.env | cut -d= -f2")

cd "$(dirname "$0")/../api"
export APP_ENV=local HTTP_ADDR=:8080 APP_ORIGIN=http://localhost:3000 COOKIE_SECURE=false MAIL_DRIVER=log
export DATABASE_DSN="reword:${pw}@tcp(127.0.0.1:${PORT})/reword"
exec go run ./cmd/api
