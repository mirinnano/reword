# ADR 0002: sqlc を MariaDB で制約付き採用する

- 状態: 採用
- 日付: 2026-10-28

## 背景

型安全な SQL アクセスのため sqlc（v1.31.1, `engine: mysql`）を検討した。sqlc の MySQL パーサーは
MariaDB 固有構文を完全には扱えないため、フェーズ0で実際のスキーマとクエリを通して検証した。

## 検証結果

| 項目 | 結果 | 対処 |
|---|---|---|
| `JSON` 列（`CHECK (JSON_VALID(...))` 付き） | ✅ | `json.RawMessage` にマッピング |
| `ENUM` 列 | ✅ | Go の型付き定数が生成される |
| `SELECT ... FOR UPDATE SKIP LOCKED` | ✅ | ジョブキューに使用 |
| `IF()` 関数 | ✅ | — |
| `COLLATE=utf8mb4_uca1400_ai_ci`（DDL） | ❌ 解析エラー | DDL に照合順序を書かず、サーバー既定値で設定 |
| `UTC_TIMESTAMP(6)` | ❌ 関数が未定義 | 現在時刻はアプリから引数で渡す |
| `INSERT ... RETURNING` | ❌ 構文エラー | `:execlastid` を使う |

## 決定

sqlc を採用する。ただし以下を規約とする。

1. 照合順序・文字コードは DDL に書かない。サーバー（`collation-server`）とデータベース既定値で設定する。
2. 現在時刻はアプリ側の `time.Now().UTC()` を引数で渡す。
3. MariaDB 固有構文が必要なクエリは sqlc の対象外とし、`internal/<feature>/` に手書きして結合テストで担保する。
4. CI で `sqlc diff` を実行し、生成コードの差分を検出する。

## 影響

- 照合順序がサーバー設定に依存するため、本番・CI・ローカルで同じ設定を必ず使う（docker-compose と CI に明記済み）。
