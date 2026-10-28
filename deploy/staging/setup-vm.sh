#!/usr/bin/env bash
# One-time setup for an Ubuntu 24.04 VM on Oracle Cloud (tested target:
# VM.Standard.E2.1.Micro, 1 GB RAM). Works on the full or Minimal image.
# Run on the VM:  curl -fsSL https://raw.githubusercontent.com/mirinnano/reword/main/deploy/staging/setup-vm.sh | bash
set -euo pipefail

REPO_URL=https://github.com/mirinnano/reword.git
APP_DIR=$HOME/reword
export DEBIAN_FRONTEND=noninteractive

echo "==> Swap (2 GB; the VM has only 1 GB RAM)"
if ! swapon --show | grep -q /swapfile; then
  sudo fallocate -l 2G /swapfile
  sudo chmod 600 /swapfile
  sudo mkswap /swapfile >/dev/null
  sudo swapon /swapfile
  grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab >/dev/null
fi
echo 'vm.swappiness=10' | sudo tee /etc/sysctl.d/99-reword.conf >/dev/null
sudo sysctl -q --system

echo "==> Remove packages this host does not need"
if command -v snap >/dev/null; then
  for s in $(snap list 2>/dev/null | awk 'NR>1 {print $1}' | grep -vE '^(core|snapd)' || true); do sudo snap remove --purge "$s" || true; done
  sudo apt-get purge -y snapd
  sudo apt-mark hold snapd
fi
sudo apt-get purge -y modemmanager multipath-tools udisks2 packagekit 2>/dev/null || true
sudo apt-get autoremove -y --purge

echo "==> Security updates only, applied automatically"
sudo apt-get update -y
sudo apt-get install -y unattended-upgrades
sudo tee /etc/apt/apt.conf.d/20auto-upgrades >/dev/null <<'EOF'
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
EOF

echo "==> SSH: keys only, no root login"
sudo tee /etc/ssh/sshd_config.d/99-reword.conf >/dev/null <<'EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
sudo systemctl reload ssh

echo "==> Docker"
if ! command -v docker >/dev/null; then
  sudo apt-get install -y ca-certificates curl git
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update -y
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
  sudo usermod -aG docker "$USER"
fi
# Keep container logs from filling the small boot volume.
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{ "log-driver": "json-file", "log-opts": { "max-size": "10m", "max-file": "3" } }
EOF
sudo systemctl restart docker

echo "==> Firewall (Oracle Ubuntu images reject everything but SSH in iptables)"
for port in 80 443; do
  sudo iptables -C INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null \
    || sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport "$port" -j ACCEPT
done
if command -v netfilter-persistent >/dev/null; then
  sudo netfilter-persistent save
else
  sudo apt-get install -y iptables-persistent
  sudo netfilter-persistent save
fi

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
COMPOSE="sudo docker compose -f $APP_DIR/deploy/staging/compose.yml --env-file $ENV_FILE"
$COMPOSE pull
$COMPOSE up -d

echo
echo "Done. API host: $(grep API_HOST "$ENV_FILE" | cut -d= -f2)"
echo "ORIGIN_SECRET is in $ENV_FILE (needed for the Worker secret)."
free -h
