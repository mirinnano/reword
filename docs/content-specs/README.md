# コンテンツ形式

問題と単語は `content/` 以下の YAML で管理し、git の履歴と PR でレビューする。
デプロイ時に `content import` が DB へ同期する（[api/internal/content](../../api/internal/content)）。

```
content/
└── g2/
    ├── vocab/*.yaml           # 単語
    └── questions/
        └── g2-q1/*.yaml       # 大問1（語彙・熟語の空所補充）
```

## 原則

- **すべてオリジナル。** 過去問・市販教材の文章や例文を写さない。単語の選定は自由だが、例文と問題文は自作する。
- **key は一度付けたら変えない。** 学習履歴と結び付くため。削除したい場合はファイルから消す（DB では retired になる）。
- 1 ファイルに複数件を書いてよい。ファイル名は自由（例: `0001-0050.yaml`）。

## 単語（`vocab/*.yaml`）

```yaml
grade: g2
items:
  - key: g2-v-abandon          # <grade>-v-<見出し語>。英小文字・数字・ハイフンのみ
    headword: abandon
    pos: verb                  # noun | verb | adjective | adverb | phrase | other
    meanings_ja: ["（計画などを）断念する", "見捨てる"]
    example_en: The team had to abandon the climb because of the storm.
    example_ja: 嵐のため、チームは登頂を断念しなければならなかった。
```

- 内容を直しても学習履歴（復習カード）はそのまま引き継がれる。

## 大問1: 語彙・熟語の空所補充（`questions/g2-q1/*.yaml`）

```yaml
grade: g2
type: g2-q1
items:
  - key: g2-q1-0001            # <type>-<4桁以上の連番>
    stem: The company decided to ( ) the project after losing its main sponsor.
    choices: [abandon, acquire, approve, assemble]
    answer: 0                  # choices の添字（0 始まり）
    explanation_ja: 主要スポンサーを失ったので計画を「断念した」。acquire は「獲得する」…
    vocab: [g2-v-abandon]      # 任意。関連する単語の key
    difficulty: 2              # 1（易）〜 3（難）
```

- `stem` には空所 `( )` をちょうど 1 つ含める。
- `choices` は 4 つ、重複なし。
- **内容を変えると新しい版になる。** 解答記録は解いた時点の版を指すので、正解を直しても過去の採点は変わらない。

## 検証と取り込み

```bash
cd api
go run ./cmd/content validate ../content   # CI でも実行
go run ./cmd/content import ../content     # DATABASE_DSN が必要。--dry-run で差分のみ表示
```
