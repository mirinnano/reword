# ReWord（仮称）

英検 準2級〜1級 向けのデジタル学習サービス。計画は [docs/PLAN.md](docs/PLAN.md)、設計判断は [docs/adr/](docs/adr/)。

## 構成

- `api/` — Go API／Worker（chi, sqlc, golang-migrate）
- `web/` — Next.js（App Router）
- `docs/` — 計画、ADR、コンテンツ仕様、runbook

## ローカル開発

必要なもの: Go 1.27、Node 24、pnpm、Docker 互換のコンテナ実行環境

```bash
cp .env.example .env
make up        # MariaDB 起動
make migrate   # マイグレーション
make api       # :8080
make web       # :3000（/api は Go に転送）
```

`make check` で lint・テスト・ビルドをまとめて実行する。
