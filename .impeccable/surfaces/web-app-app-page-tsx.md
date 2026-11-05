---
version: 1
slug: "web-app-app-page-tsx"
primary_target: "web/app/app/page.tsx"
related_targets: ["web/app/app/practice/[id]/page.tsx","web/app/page.tsx"]
---

# ReWord 学習アプリ

Scope: 全画面（トップ、認証、ホーム、演習、採点）。Mode: Operate（トップのみ Persuade）。
Audience: 英検を受ける中高生本人。Job: 迷わず今日の日課を始め、解き終える。
Unresolved: 利用場面は未検証。新規が出ない日の割合は実データ待ち。

## Direction contract

THESIS: 毎日の日課は「今日の 1 話」。話数は累計で、休んでも減らない。連続日数や順位で煽る学習アプリの定番と、カードと進捗リングの標準形を拒む。

OWN-WORLD: 白い紙と黒インク、特色の黄 1 色と赤 1 色。夜に解く利用者のため、OS がダークのときは紙と墨を反転し、特色の黄はそのまま使う。太い黒枠のコマで面を分け、角は丸めない。階調はスクリーントーンの網点のみで、グラデーションと影は使わない。見出しと数字は極太ゴシック、本文は読みやすいゴシック、英文は判別しやすいサンセリフ。世界は書体、配色、密度、1 つの署名だけを貸し、ナビゲーションと操作部品は標準のまま。

STORY: 開くと今日の話数と内容が分かる。1 タップで始める。採点の瞬間だけ描き文字が出る。間違いは責めず、また出ると伝えて終わる。

FIRST VIEWPORT: スマホのホーム。上部に名前と学習者。黄色の大コマに話数と「今日の 12 問」を極太で置き、その下に内容をコマ割りで示す。コマの幅は所要時間に比例する。直下に黒の開始ボタン。形式別の演習と履歴はその下。

FORM: 週刊少年誌のコマ割りと印刷。候補リストの 3 番目。署名は「コマの大きさが時間を表すコマ割り」。長文の本文は例外で、枠と本文だけの静かな面にする。途中で終える操作は周囲から離して置く。seed:8ee4271402edc870

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
