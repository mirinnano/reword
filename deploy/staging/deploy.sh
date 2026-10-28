#!/usr/bin/env bash
# Manual deploy of the current origin/main to the staging VM. CI does the same
# automatically (.github/workflows/deploy.yml); the image for the commit must exist.
#   STAGING_SSH=ubuntu@<ip> deploy/staging/deploy.sh
set -euo pipefail
: "${STAGING_SSH:?set STAGING_SSH=user@host}"
SSH_KEY=${STAGING_SSH_KEY:-$HOME/.ssh/reword_staging}

git fetch -q origin main
sha=$(git rev-parse origin/main)
ssh -i "$SSH_KEY" "$STAGING_SSH" \
  "cd ~/reword && git fetch -q origin main && git checkout -q --detach $sha && SSH_ORIGINAL_COMMAND=$sha deploy/staging/remote-deploy.sh"
