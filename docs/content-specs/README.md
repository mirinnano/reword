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

### 品質基準（大問1）

語彙問題が測るのは、正解の語を知っているかどうかである。誤答の選び方を誤ると、正解の語を知らない学習者でも消去法で正解できてしまい、問題がこの能力を測らなくなる。以下の基準は、2026年10月の監査（[docs/content-audits](../content-audits/)）で見つかった弱点を、作問の段階で防ぐためのものである。

#### 誤答の選び方

誤答は、正解と品詞・活用形をそろえる。句動詞の問題なら 4 つとも句動詞にし、単語と句を混ぜない。形がそろっていないと、意味を知らなくても形だけで誤答を外せる。

前置詞・自動詞と他動詞の違い・冠詞（"an" の直後の子音始まりの語など）といった語法だけで外せる語も、同じ理由で誤答に使わない。

誤答は、文の話題と同じ領域から選ぶ。"swallowed the exhibition" のように話題がまったく違う語は、その語の意味さえ知っていれば一目で外せる。4 つのうち 3 つがこの種類だと、問題は「正解の語を知っているか」ではなく「誤答 3 語のどれかの意味を知っているか」を測ることになる。監査で使った 3 段階の評価（0：話題違いで一目で外せる、1：少し考えれば外せる、2：迷う）でいえば、誤答の少なくとも 2 つを 1 以上にする。

誤答の難しさは、正解と同じ 2級相当にそろえる。3級以下の易しい語は意味を知られていて外されやすく、準1級を超える難語は学習者が知らないために「知らない語は正解ではないだろう」という推測を誘う。

ネイティブが受け入れる文になる誤答は使わない。正解が 2 つになるからである。

4 つの選択肢の頭文字がそろわないようにする。作問の担当を頭文字の範囲で分けると、担当者が誤答も自分の範囲から選びがちで、頭文字がそろいやすい（監査時点で 150 問中 9 問）。頭文字だけで正解が分かるわけではないが、選択肢の並びが不自然に見える。

#### 問題文

問題文は 8〜30 語とする。空所の直前の語との組み合わせだけで答えが決まる文より、文全体の意味から答えが決まる文のほうが、語の意味を理解しているかを測れる。状況や語句が既存の問題と重なる文は避ける（`content lint` の near-duplicate が検出する）。

#### 解説（explanation_ja）

解説は、文意と正解の意味を述べてから、各誤答の意味を述べる。学習者は誤答の意味も解説から覚えるので、誤答の訳も正確に書く。"conduct a survey"（調査を行う）のような、覚えておくと役に立つ組み合わせは明記する。

## 検証・監査・取り込み

```bash
cd api
go run ./cmd/content validate ../content    # 形式の誤り（CI でも実行）
go run ./cmd/content lint ../content        # 品質の警告（選択肢の形の不一致、重複など）
go run ./cmd/content blind -out <dir> ../content   # 独立解答者用（答えなし・選択肢を並べ替え）＋ key.json
go run ./cmd/content review -out <dir> ../content  # 査読者用（答え・解説・訳語つき）
go run ./cmd/content import ../content      # DATABASE_DSN が必要。--dry-run で差分のみ表示
```

監査の手順は [docs/content-audits/README.md](../content-audits/README.md)。
