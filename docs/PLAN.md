# ReWord（仮称）開発計画

英検 準2級〜1級 向けのデジタル学習サービス。1 名開発、製品リリース前提、初期は無料枠中心で運用する。

## 前提

| 項目 | 決定 |
|---|---|
| 対象 | 個人向け（B2C）。保護者アカウントの下に子どもの学習者を作れる |
| 対象級 | 準2級・準2級プラス・2級・準1級・1級（スキーマは5級〜1級まで対応） |
| 開発体制 | 1 名 |
| 問題の調達 | LLM で下書き → 監修 → TTS で音声化（オリジナル問題のみ。過去問は使わない） |
| ホスティング | 無料枠中心 → 負荷に応じて AWS へ（[ADR 0003](adr/0003-free-tier-hosting.md)） |
| 名称 | 「ReWord」は仮称（Key 作品『Rewrite』から着想）。商標調査後に確定 |

## 1 名体制による優先順位

- **コンテンツが最大のボトルネック。** 5 級分を一度に揃えるのは非現実的なので、リリースは級ごとに段階的に行う。
  - 順序案: 2級 → 準2級／準2級プラス → 準1級 → 1級（1級は語彙・作文とも監修の難度が最も高い）
- 自分で運用しなくて済む部分（CDN・ストレージ・メール・エラー監視）は外部サービスに任せる。
- 自前で運用するのは VM 1 台（API・Worker・MariaDB）だけに絞る。

## アーキテクチャ

```
ブラウザ ── Cloudflare（DNS / CDN / WAF）
              ├─ /api/*  → Cloudflare Tunnel → VM: Go API ─┐
              └─ その他  → Workers: Next.js（OpenNext）     │
                                                           ├─ MariaDB（同 VM）
                          VM: Go Worker（TTS・メール・集計）─┘
              R2: 音声・画像・DB バックアップ
```

## 技術選定

| 領域 | 採用 |
|---|---|
| API | Go 1.27、chi、slog、機能ごとのパッケージ |
| DB アクセス | sqlc（制約付き、[ADR 0002](adr/0002-sqlc-with-mariadb.md)）、golang-migrate |
| 認証 | MariaDB セッション＋httpOnly Cookie＋Origin 検証（[ADR 0001](adr/0001-session-auth.md)） |
| ジョブ | MariaDB テーブル＋`FOR UPDATE SKIP LOCKED` |
| Web | Next.js 16（App Router）。紹介ページは静的生成、学習画面はクライアント側＋TanStack Query |
| API 契約 | OpenAPI。TS 側の型のみ生成（openapi-typescript） |
| 単語学習 | FSRS＋復習履歴の保存 |
| 監視 | Sentry、構造化ログ、外形監視 |

## データモデル（主要部分）

- **アカウント**: `users`、`learner_profiles`、`sessions`、`consents`
- **試験構造**: `grades`（作成済み）、`question_types`
- **コンテンツ**（公開後は編集せず、新しい版を作る）: `questions`（`stable_key`＋`version`、`format`、`payload_json`、`answer_json`、`status`）、`passages`、`assets`、`content_reviews`
- **学習**: `attempts`（サーバー側の締切時刻）、`attempt_items`（問題の版を固定）、`responses`
- **単語**: `vocab_items`、`vocab_cards`、`review_logs`
- **運用**: `jobs`（作成済み）、`audit_logs`、`subscriptions`、`stripe_events`

スコアは正答率と合格ラインの目安のみ表示し、CSE スコアの換算は出さない。

## フェーズ

| # | 内容 | 完了条件 |
|---|---|---|
| 0 | 雛形、CI、sqlc 検証、ADR、無料枠 VM＋Cloudflare に staging を構築 | main へのマージで staging に自動デプロイされる |
| 1 | アカウント（登録・メール確認・セッション・保護者と子ども・パスワード再設定・同意） | 子どもの学習者を作り、ログインできる |
| 2 | コンテンツ基盤（版管理・管理画面・レビュー・TTS ジョブ・R2 署名付き URL） | 1 つの大問形式で生成から公開まで一通り動く |
| 2' | **並行作業**: 2級のコンテンツ制作 | 目標問題数に到達 |
| 3 | 演習・解説・リスニング・FSRS 単語学習 | 主要な流れの E2E テストが CI で通る |
| 4 | 模試・技能別の結果・苦手分析 | 2級の模試を最初から最後まで通して受けられる |
| 5 | Stripe の定期課金、無料プランの制限、法務文書 | テストモードで課金が一通り動く |
| 6 | リリース準備（負荷テスト・脆弱性診断・復元訓練・runbook） | リリース判定チェックリストをすべて満たす |
| 7 | 2級で β 版 → 正式版 → 他の級を順に追加 | — |

期間の見積もりは、フェーズ1の実績（速度）をもとに出す。

## フェーズ0 の進捗

- [x] モノレポ・Go API 雛形（`/api/healthz`、`/api/readyz`、正常終了処理）
- [x] マイグレーション（`grades`、`jobs`）と migrate コマンド
- [x] sqlc×MariaDB の検証 → [ADR 0002](adr/0002-sqlc-with-mariadb.md)
- [x] Next.js 雛形（開発時は `/api` を Go に転送）
- [x] Makefile、GitHub Actions、API の Dockerfile
- [x] CI（GitHub Actions）上の MariaDB で結合テスト
- [x] GitHub リポジトリ作成（公開）と CI の初回実行
- [x] Web を Cloudflare Workers の staging にデプロイ
- [ ] Oracle VM に API と MariaDB を構築（[runbook](runbooks/staging.md)）
