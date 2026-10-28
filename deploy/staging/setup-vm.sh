#!/usr/bin/env bash
# One-time setup for an Ubuntu VM (Oracle Cloud Always Free, ARM).
# Run on the VM:  curl -fsSL https://raw.githubusercontent.com/mirinnano/reword/main/deploy/staging/setup-vm.sh | bash
set -euo pipefail

REPO_URL=https://github.com/mirinnano/reword.git
APP_DIR=$HOME/reword

echo "==> Docker"
if ! command -v docker >/dev/null; then
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update -y
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin git
  sudo usermod -aG docker "$USER"
fi

echo "==> Firewall (Oracle Ubuntu images reject everything but SSH in iptables)"
for port in 80 443; do
  sudo iptables -C INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null \
    || sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport "$port" -j ACCEPT
done
sudo netfilter-persistent save

echo "==> Code"
[ -d "$APP_DIR/.git" ] || git clone "$REPO_URL" "$APP_DIR"

echo "==> Secrets"
ENV_FILE=$APP_DIR/deploy/staging/.env
if [ ! -f "$ENV_FILE" ]; then
  ip=$(curl -fsS https://ifconfig.me)
  umask 077
  cat >"$ENV_FILE" <<EOF
MARIADB_ROOT_PASSWORD=$(openssl rand -hex 24)
MARIADB_PASSWORD=$(openssl rand -hex 24)
ORIGIN_SECRET=$(openssl rand -hex 32)
API_HOST=${ip//./-}.sslip.io
EOF
fi

echo "==> Start"
sudo docker compose -f "$APP_DIR/deploy/staging/compose.yml" --env-file "$ENV_FILE" up -d --build

echo
echo "Done. API host: $(grep API_HOST "$ENV_FILE" | cut -d= -f2)"
echo "ORIGIN_SECRET is in $ENV_FILE (needed for the Worker secret)."
