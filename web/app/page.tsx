"use client";

import Image from "next/image";
import { useState } from "react";
import { Burst, ButtonLink, Mark } from "@/components/ui";
import { useMe } from "@/lib/auth";

// A real question from the 2級 set (content/g2/questions/g2-q1), answerable here without an account.
const SAMPLE = {
  before: "Because of the heavy snow, the school decided to ",
  after: " the sports festival until next week.",
  choices: ["preserve", "postpone", "reserve", "organize"],
  answer: 1,
  explanation: "大雪のため運動会を来週まで「延期する」。preserve は「保存する」、reserve は「予約する」、organize は「企画する」。",
};

// The real review intervals (api/internal/practice/select.go). Each panel's width is its number of days.
const GAPS = [1, 4, 10, 30, 90];

// Screenshots of the real app with real questions. They replace nothing a photograph would prove.
const SHOTS = [
  { src: "/shots/home.webp", caption: "今日の話が決まっている", alt: "ホーム画面。第12話、今日の12問と、復習、新規、長文の内訳" },
  { src: "/shots/question.webp", caption: "解いたらすぐ採点", alt: "語彙問題の採点画面。正解の描き文字と解説" },
  { src: "/shots/passage.webp", caption: "長文は本文を見ながら", alt: "長文問題の画面。本文の下に設問と選択肢" },
  { src: "/shots/summary.webp", caption: "間違いは、また出る", alt: "採点結果の画面。12問中3問正解と、問題ごとのふりかえり" },
];

export default function Home() {
  const me = useMe();
  return (
    <>
      <main id="main" className="mx-auto w-full max-w-5xl flex-1 px-4 pb-16">
        <header className="flex h-14 items-center justify-between">
          <span className="font-display text-xl">
            ReWord<span className="font-sans text-sm text-muted">（仮）</span>
          </span>
          {!me.data && (
            <a href="/login" className="font-medium underline decoration-2 underline-offset-4">
              ログイン
            </a>
          )}
        </header>

        <div className="mt-2 grid gap-1 lg:grid-cols-[minmax(0,5fr)_minmax(0,4fr)]">
          <section className="flex flex-col justify-between gap-8 border-3 border-ink bg-spot p-5 text-on-spot lg:p-8">
            <div>
              <h1 className="font-display text-5xl leading-tight lg:text-7xl">
                英検2級、
                <br />
                毎日12問。
              </h1>
              <p className="mt-4 text-lg font-medium leading-relaxed">
                語彙と長文を解いて、間違えた問題は日をあけて復習する。今日やることは、開けば決まっています。
              </p>
            </div>
            <div className="flex flex-col gap-1 sm:flex-row">
              {me.data ? (
                <ButtonLink href="/app" variant="onSpot">
                  今日の日課へ
                </ButtonLink>
              ) : (
                <ButtonLink href="/signup" variant="onSpot">
                  無料で登録する
                </ButtonLink>
              )}
            </div>
          </section>
          <Sample />
        </div>

        <section className="mt-16">
          <h2 className="font-display text-3xl leading-snug">開いて、解いて、おわり</h2>
          <p className="mt-2 text-sm text-muted">開発中の画面です。話数や成績はサンプルの表示です。</p>
          <ul className="mt-5 grid grid-cols-2 gap-1 lg:grid-cols-4">
            {SHOTS.map((s) => (
              <li key={s.src} className="rise">
                <figure className="border-3 border-ink">
                  <figcaption className="border-b-3 border-ink p-2 text-sm font-bold">{s.caption}</figcaption>
                  <Image src={s.src} alt={s.alt} width={750} height={1624} unoptimized className="block h-auto w-full" />
                </figure>
              </li>
            ))}
          </ul>
        </section>

        <section className="mt-16">
          <h2 className="font-display text-3xl leading-snug">
            間違えた問題は、
            <br className="sm:hidden" />
            忘れたころに戻ってくる
          </h2>
          <p className="mt-3 leading-relaxed">
            正解した問題は、次に出るまでの間隔が 1日、4日、10日、30日、90日と延びていきます。初めて解いて正解した問題は7日後に確かめ、間違えた問題は次の回で復習します。
          </p>
          <ul aria-label="復習の間隔" className="mt-5 space-y-1">
            {GAPS.map((days) => (
              <li key={days} style={{ width: `${(days / 90) * 100}%` }} className="rise min-w-20 border-3 border-ink p-2 whitespace-nowrap">
                <span className="font-display text-xl tabular-nums">{days}</span>
                <span className="ml-1 text-sm font-bold">日後</span>
              </li>
            ))}
          </ul>
        </section>

        <div className="mt-16 grid gap-1 lg:grid-cols-2">
          <section className="border-3 border-ink p-5">
            <h2 className="font-display text-2xl">3回続けて正解して、習得</h2>
            <p className="mt-3 leading-relaxed">
              1回の正解は、まぐれかもしれません。日をあけて3回続けて正解した問題だけを習得と数えるので、数字は本当に覚えた量を示します。
            </p>
          </section>
          <section className="tone border-3 border-ink p-5">
            <div className="bg-surface p-3">
              <h2 className="font-display text-2xl">話数は減らない</h2>
              <p className="mt-3 leading-relaxed">
                毎日の日課は1話ずつ積み上がります。連続記録はありません。休んでも、次の日は次の話から続けられます。
              </p>
            </div>
          </section>
        </div>

        <section className="mt-16">
          <h2 className="font-display text-3xl">公開中の問題</h2>
          <dl className="mt-5 grid grid-cols-3 gap-1">
            <Figure value="150問" label="語彙・熟語" />
            <Figure value="12題" label="長文" />
            <Figure value="1,585語" label="単語" />
          </dl>
          <p className="mt-3 text-sm leading-relaxed">
            すべて英検2級向けのオリジナル問題です。解説は日本語で読めます。実力診断と基礎固めのために、準2級の語彙問題も36問あります。
          </p>
        </section>
      </main>

      <footer className="border-t-3 border-ink px-4 py-8">
        <div className="mx-auto max-w-5xl">
          <p className="font-display text-5xl lg:text-8xl">ReWord</p>
          <p className="mt-4 text-sm text-muted">ReWord は仮称です。開発中の検証環境のため、データは予告なく削除されることがあります。</p>
          <p className="mt-2 flex gap-4 text-sm">
            <a href="/terms" className="underline underline-offset-4">
              利用規約
            </a>
            <a href="/privacy" className="underline underline-offset-4">
              プライバシーポリシー
            </a>
          </p>
        </div>
      </footer>
    </>
  );
}

function Figure({ value, label }: { value: string; label: string }) {
  return (
    <div className="flex flex-col-reverse border-3 border-ink p-3">
      <dt className="mt-1 text-sm font-bold">{label}</dt>
      <dd className="font-display text-lg tabular-nums whitespace-nowrap sm:text-xl lg:text-4xl">{value}</dd>
    </div>
  );
}

/** The live thing: the first question of the product, graded on the spot. */
function Sample() {
  const [picked, setPicked] = useState<number | null>(null);
  const done = picked !== null;
  const correct = picked === SAMPLE.answer;
  return (
    <section aria-labelledby="sample" className="border-3 border-ink p-5">
      <h2 id="sample" className="font-bold">
        1問、解いてみる
      </h2>
      <p lang="en" className="mt-3 font-en text-xl leading-relaxed">
        {SAMPLE.before}
        {done ? (
          <ins className="font-bold no-underline">{SAMPLE.choices[SAMPLE.answer]}</ins>
        ) : (
          <span className="mx-1 inline-block w-20 border-b-3 border-ink">
            <span className="sr-only">（空所）</span>&nbsp;
          </span>
        )}
        {SAMPLE.after}
      </p>
      <ul className="mt-4 space-y-1.5" aria-label="選択肢">
        {SAMPLE.choices.map((c, i) => {
          const isAnswer = done && i === SAMPLE.answer;
          const isWrong = done && i === picked && !correct;
          return (
            <li key={c}>
              <button
                onClick={() => setPicked(i)}
                disabled={done}
                lang="en"
                className={`press flex min-h-12 w-full items-center justify-between gap-2 border-3 px-3 text-left font-en text-lg disabled:cursor-default ${
                  isAnswer ? "border-ink bg-spot text-on-spot" : isWrong ? "border-ng" : done ? "border-hairline text-muted" : "border-ink hover:bg-tint"
                }`}
              >
                {c}
                {isAnswer && <Mark correct className="size-6" />}
                {isWrong && <Mark correct={false} className="size-6" />}
              </button>
            </li>
          );
        })}
      </ul>
      {done && (
        <div className="mt-3" role="status">
          {correct ? <Burst>正解</Burst> : <p className="stamp font-display text-xl text-ng">不正解</p>}
          <p className="mt-2 text-sm leading-relaxed">{SAMPLE.explanation}</p>
        </div>
      )}
    </section>
  );
}
