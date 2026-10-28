#!/usr/bin/env bash
# Deploy main to the staging VM (pulls the image built by .github/workflows/image.yml):  STAGING_SSH=ubuntu@<ip> deploy/staging/deploy.sh
set -euo pipefail
: "${STAGING_SSH:?set STAGING_SSH=user@host}"
SSH_KEY=${STAGING_SSH_KEY:-$HOME/.ssh/reword_staging}

ssh -i "$SSH_KEY" "$STAGING_SSH" bash -s <<'EOF'
set -euo pipefail
cd ~/reword
git fetch -q origin main && git reset -q --hard origin/main
docker compose -f deploy/staging/compose.yml --env-file deploy/staging/.env pull
docker compose -f deploy/staging/compose.yml --env-file deploy/staging/.env up -d --remove-orphans
docker image prune -f >/dev/null
docker compose -f deploy/staging/compose.yml --env-file deploy/staging/.env ps
EOF
