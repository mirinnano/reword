#!/usr/bin/env bash
# Runs on the VM as the forced command of the CI deploy key
# (authorized_keys: command="~/reword/deploy/staging/remote-deploy.sh",restrict).
# The only input is the commit SHA in SSH_ORIGINAL_COMMAND; anything else is refused.
set -euo pipefail

sha=${SSH_ORIGINAL_COMMAND:-}
if [[ ! $sha =~ ^[0-9a-f]{40}$ ]]; then
  echo "refused: expected a 40-char commit sha" >&2
  exit 2
fi

cd ~/reword
git fetch -q origin main
git checkout -q --detach "$sha"

env_file=deploy/staging/.env
export API_IMAGE=ghcr.io/mirinnano/reword-api:$sha
compose="docker compose -f deploy/staging/compose.yml --env-file $env_file"
$compose pull -q
$compose up -d --remove-orphans

# Readiness through Caddy on loopback, exactly as the Worker reaches it.
host=$(grep '^API_HOST=' "$env_file" | cut -d= -f2)
secret=$(grep '^ORIGIN_SECRET=' "$env_file" | cut -d= -f2)
for _ in $(seq 1 30); do
  if curl -fsS -o /dev/null --max-time 3 --resolve "$host:443:127.0.0.1" \
       -H "X-Origin-Auth: $secret" "https://$host/api/readyz"; then
    docker image prune -f >/dev/null
    echo "deployed $sha"
    exit 0
  fi
  sleep 2
done
echo "api did not become ready" >&2
$compose logs --tail 50 api migrate >&2
exit 1
