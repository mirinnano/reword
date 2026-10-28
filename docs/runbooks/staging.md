# staging 環境

| 役割 | 場所 |
|---|---|
| Web（Next.js） | https://reword-web-staging.ponkotu.workers.dev（Cloudflare Workers） |
| API / MariaDB | Oracle Cloud Always Free VM: E2.1.Micro（x86, 1/8 OCPU, 1 GB RAM＋スワップ 2 GB）（`deploy/staging/compose.yml`） |
| API イメージ | `ghcr.io/mirinnano/reword-api:staging`（`.github/workflows/image.yml` が main への push でビルド。VM 上ではビルドしない） |
| API の公開ホスト | `<IP をハイフン区切り>.sslip.io`（Caddy、Let's Encrypt） |

ブラウザは Worker だけにアクセスし、`/api/*` は Worker が VM の Caddy へ転送する。
Caddy は `X-Origin-Auth` ヘッダー（共有シークレット）がないリクエストを 403 で拒否する。

## 初回構築

### 1. Oracle Cloud で VM を作る（コンソール操作）

1. Compute → Instances → Create instance
   - Image: **Canonical Ubuntu 24.04 Minimal**（通常版でも動く）
   - Shape: **VM.Standard.E2.1.Micro**（A1.Flex が取れればそちらが望ましい。compose の MariaDB メモリ設定を引き上げる）
   - Networking: 新規 VCN＋パブリック・サブネット、「パブリック IPv4 を自動割当て」をオン
   - 作成後、パブリック IP を予約済み IP に切り替える（API_HOST が IP から決まるため）
   - SSH keys: 「Paste public keys」に `~/.ssh/reword_staging.pub` の内容を貼る
2. VCN → Subnet → Security List → Ingress Rules に追加
   - `0.0.0.0/0` TCP **80**
   - `0.0.0.0/0` TCP **443**
3. 東京リージョンで「Out of capacity」が出たら、時間を置くか可用性ドメインを変えて再試行する。

### 2. VM を初期化する（ローカルから）

```bash
ssh -i ~/.ssh/reword_staging ubuntu@<IP> 'bash -s' < deploy/staging/setup-vm.sh
```

スワップ作成、不要パッケージ（snapd など）の削除、セキュリティ更新の自動適用、SSH の鍵認証限定、
Docker の導入、iptables の 80/443 開放（イメージ既定の REJECT ルールの手前に挿入）、リポジトリの取得、シークレット生成（`deploy/staging/.env`）、
GHCR からのイメージ取得と起動までを行う。何度実行しても同じ状態になる。

> raw.githubusercontent.com 経由の `curl | bash` は使わない（数分キャッシュされ古い版が実行されることがある）。

### 3. Worker に API の接続先を設定する

```bash
cd web
ssh -i ~/.ssh/reword_staging ubuntu@<IP> 'grep -E "^(API_HOST|ORIGIN_SECRET)=" ~/reword/deploy/staging/.env'
pnpm exec wrangler secret put API_ORIGIN --env staging     # https://<API_HOST>
pnpm exec wrangler secret put ORIGIN_SECRET --env staging  # 上で表示された値
```

## デプロイ

```bash
STAGING_SSH=ubuntu@<IP> deploy/staging/deploy.sh   # API（main の image ワークフロー完了後。VM 上で git pull → pull → up）
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
