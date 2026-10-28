# staging 環境

| 役割 | 場所 |
|---|---|
| Web（Next.js） | https://reword-web-staging.ponkotu.workers.dev（Cloudflare Workers） |
| API / MariaDB | Oracle Cloud Always Free VM（`deploy/staging/compose.yml`） |
| API の公開ホスト | `<IP をハイフン区切り>.sslip.io`（Caddy、Let's Encrypt） |

ブラウザは Worker だけにアクセスし、`/api/*` は Worker が VM の Caddy へ転送する。
Caddy は `X-Origin-Auth` ヘッダー（共有シークレット）がないリクエストを 403 で拒否する。

## 初回構築

### 1. Oracle Cloud で VM を作る（コンソール操作）

1. Compute → Instances → Create instance
   - Image: **Canonical Ubuntu 24.04**
   - Shape: **VM.Standard.A1.Flex**（Ampere ARM）、2 OCPU / 12 GB 程度（Always Free は合計 4 OCPU / 24 GB まで）
   - Networking: パブリック IPv4 を割り当てる
   - SSH keys: 「Paste public keys」に `~/.ssh/reword_staging.pub` の内容を貼る
2. VCN → Subnet → Security List → Ingress Rules に追加
   - `0.0.0.0/0` TCP **80**
   - `0.0.0.0/0` TCP **443**
3. 東京リージョンで「Out of capacity」が出たら、時間を置くか可用性ドメインを変えて再試行する。

### 2. VM を初期化する（ローカルから）

```bash
ssh -i ~/.ssh/reword_staging ubuntu@<IP> 'curl -fsSL https://raw.githubusercontent.com/mirinnano/reword/main/deploy/staging/setup-vm.sh | bash'
```

Docker の導入、iptables の 80/443 開放、リポジトリの取得、シークレット生成（`deploy/staging/.env`）、起動までを行う。

### 3. Worker に API の接続先を設定する

```bash
cd web
ssh -i ~/.ssh/reword_staging ubuntu@<IP> 'grep -E "^(API_HOST|ORIGIN_SECRET)=" ~/reword/deploy/staging/.env'
pnpm exec wrangler secret put API_ORIGIN --env staging     # https://<API_HOST>
pnpm exec wrangler secret put ORIGIN_SECRET --env staging  # 上で表示された値
```

## デプロイ

```bash
STAGING_SSH=ubuntu@<IP> deploy/staging/deploy.sh   # API（VM 上で git pull → build → up）
cd web && pnpm deploy:staging                       # Web（Workers）
```

## テスト

```bash
STAGING_URL=https://reword-web-staging.ponkotu.workers.dev STAGING_SSH=ubuntu@<IP> deploy/staging/test.sh
```

1. 公開 URL 経由で `/api/healthz`・`/api/readyz` を確認（Worker → Caddy → API → MariaDB の疎通）
2. SSH トンネル経由で staging の MariaDB に Go の結合テストを実行

## 未対応

- MariaDB の日次バックアップ（R2 へ）
- GitHub Actions からの自動デプロイ
