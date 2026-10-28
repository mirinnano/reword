# ADR 0003: 初期は無料枠中心でホスティングし、AWS へ移行できる形を保つ

- 状態: 採用（無料枠の条件は契約前に要確認）
- 日付: 2026-10-28

## 背景

開発者は 1 名、収益化前は固定費をほぼゼロにしたい。一方で製品としてリリースするため、
バックアップ・監視・HTTPS は最初から必要。DB は MariaDB 固定。

## 決定

| 役割 | 初期構成 | 移行先（負荷増加時） |
|---|---|---|
| DNS / CDN / WAF | Cloudflare（Free） | 継続 |
| Web（Next.js） | Cloudflare Workers（OpenNext アダプター） | 継続 or ECS |
| API / Worker（Go） | 無料枠 VM 1 台（例: Oracle Cloud Always Free ARM）で docker compose | AWS ECS Fargate |
| DB（MariaDB） | 同 VM 上のコンテナ | AWS RDS for MariaDB |
| 音声・画像 | Cloudflare R2（エグレス無料） | 継続（S3 互換 API） |
| DB バックアップ | `mariadb-backup` を日次で R2 へ | RDS 自動バックアップ |
| VM への経路 | Cloudflare Tunnel（VM はポートを公開しない） | ALB |
| メール | Resend 等の無料枠 | Amazon SES |
| エラー監視 | Sentry 無料枠 | 継続 |

同一オリジン: Cloudflare 上で `/api/*` を Tunnel 経由で VM へ、それ以外を Workers へ振り分ける。

## 移行しやすさのための制約

- アプリはすべてコンテナで動かし、設定は環境変数のみ。
- ストレージは S3 互換 API 経由でのみ扱う（R2 → S3 はエンドポイント変更のみ）。
- ジョブキューは DB テーブル（外部ブローカーに依存しない）。

## 注意点

- **Azure Database for MariaDB は 2025 年に提供終了済み**のため、Azure へ移る場合は自前運用か MySQL への移行が必要。
  マネージド MariaDB の移行先は AWS RDS を第一候補とする。
- 無料枠 VM は単一障害点。β〜初期の正式版までは許容し、復元手順（runbook）と復元訓練で補う。
- 無料枠の条件（休止アカウントの回収、容量不足など）は変わり得るため、契約時に確認する。
- 移行の目安: 有料会員の発生、または API の p95 レイテンシ・VM の CPU が継続的に閾値を超えたとき。
