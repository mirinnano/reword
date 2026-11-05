---
name: ReWord
description: 毎日の日課を今日の 1 話として解く、週刊マンガ誌の世界の英検学習アプリ
colors:
  paper: "#ffffff"
  ink: "#111111"
  muted: "#5c5c5a"
  tint: "#f0f0ee"
  hairline: "#d8d8d4"
  spot-yellow: "#ffd400"
  spot-red: "#c8002a"
  night-ground: "#121212"
  night-surface: "#1a1a1a"
  night-ink: "#f5f5f5"
  night-red: "#ff8393"
typography:
  display:
    fontFamily: "Dela Gothic One, Hiragino Sans, sans-serif"
    fontSize: "3rem"
    fontWeight: 400
    lineHeight: 1.25
  heading:
    fontFamily: "Dela Gothic One, Hiragino Sans, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 400
  body:
    fontFamily: "Zen Kaku Gothic New, Hiragino Sans, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.625
  english:
    fontFamily: "Atkinson Hyperlegible, Hiragino Sans, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 400
    lineHeight: 1.75
rounded:
  none: "0px"
spacing:
  gutter: "4px"
  panel: "20px"
  page: "16px"
components:
  button-primary:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
    rounded: "{rounded.none}"
    height: "48px"
  button-secondary:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    height: "48px"
  panel:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "20px"
  panel-spot:
    backgroundColor: "{colors.spot-yellow}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "20px"
---

## Overview

週刊少年誌のコマ割りと印刷を、学習アプリの作業画面に持ち込んだ世界。毎日の日課は「今日の 1 話」で、話数は累計なので休んでも減らない。世界が貸すのは書体、配色、密度、1 つの署名だけで、ナビゲーションと操作部品は標準のまま使う。

署名はコマ割りの幅である。ホームの内訳は、各コマの幅が所要時間に比例する。数字の時間は実測がないので画面に出さない。

## Colors

白い紙と黒インクに、特色の黄と赤を 1 色ずつ使う。黄は今日の話、選んだ選択肢、正解を担う。赤は不正解と誤答だけを担う。OS がダークのときは紙と墨を反転し、黄はそのまま使い、黄の上の文字は常に黒にする。

## Typography

Dela Gothic One は 24px 以上の見出しと数字に使う。それより小さいと日本語がつぶれるので、ボタンと小見出しには使わない。UI の本文は Zen Kaku Gothic New、英文は語の判別を優先して Atkinson Hyperlegible を使う。長文は 1 行 70 字以内、行間 1.75 とする。

## Layout

コマの間隔は 4px で、コマ同士を近づけて 1 ページに見せる。ページの最大幅は演習とホームで 1024px、認証と採点で 448px。長文は本文と設問を常に同時に見せる。PC は 2 カラム、スマホは本文を画面上部に固定し、横向きのスマホは 2 カラムにする。

## Elevation & Depth

影は使わない。面の区別は太い枠と網点の 2 つで行う。

## Shapes

角は常に直角。枠は 3px、操作部品は 2px か 3px。例外は採点の描き文字の星形だけである。

## Components

- コマ: 3px の黒枠。
- 網点（`.tone`）: 唯一の階調で、復習や読了を示す。上に文字を置くときは紙色の下地を敷く。
- 描き文字（`Burst`）: 正解のときだけ出す、1 画面で唯一の大きな表現。動きは 160ms で、動きを減らす設定では移動をなくす。
- 長文パネル: 枠と本文だけの静かな面。網点も特色も使わず、空所の現在地だけ黄にする。
- 途中で終える操作: 解答から離して置き、その場で確認する。

## Do's and Don'ts

- 間違いは責めず、また出題すると伝える。
- 連続日数、順位、確定していない見込みの数字を出さない。
- 正誤を色だけで伝えない。形と言葉を必ず添える。
- グラデーション、影、角丸を使わない。
- 長文の読解面に演出を持ち込まない。
