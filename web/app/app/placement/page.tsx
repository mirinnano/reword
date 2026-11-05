"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { Button, ButtonLink, Loading, Notice, Page, Sheet } from "@/components/ui";
import { api, errorMessage, type BlockScore, type Item, type Session } from "@/lib/api";
import { BLOCKS, STAGES } from "@/lib/placement";
import { useTitle } from "@/lib/title";

const BLOCK_SIZE = 10;

/** Intro, then the test, then where the learner stands. Answers are not marked one by one. */
export default function PlacementPage() {
  useTitle("実力診断");
  const qc = useQueryClient();
  const status = useQuery({ queryKey: ["placement"], queryFn: api.placement });
  const [session, setSession] = useState<Session | null>(null);
  const begin = useMutation({ mutationFn: api.startPlacement, onSuccess: setSession });

  const done = () => {
    qc.invalidateQueries({ queryKey: ["placement"] });
    qc.invalidateQueries({ queryKey: ["daily"] });
  };

  if (status.isPending) return <Loading />;
  if (status.isError) {
    return (
      <Page narrow>
        <h1 className="mb-3 font-display text-2xl">実力診断を開けませんでした</h1>
        <Notice>{errorMessage(status.error)}</Notice>
      </Page>
    );
  }
  if (session && !session.progress.finished) {
    return <Test session={session} onChange={setSession} onFinished={done} />;
  }
  const stage = session?.placement?.stage ?? status.data.stage;
  const start = (
    <>
      {begin.isError && (
        <div className="mb-2">
          <Notice>{errorMessage(begin.error)}</Notice>
        </div>
      )}
      <Button className="min-h-14 w-full" variant={stage === null ? "primary" : "secondary"} disabled={begin.isPending} onClick={() => begin.mutate()}>
        {begin.isPending ? "用意しています…" : status.data.session_id ? "つづきから" : stage === null ? "診断をはじめる" : "もう一度受ける"}
      </Button>
    </>
  );
  return stage !== null ? (
    <Result stage={stage} blocks={session?.placement?.blocks ?? null} start={start} />
  ) : (
    <Intro start={start} />
  );
}

function Intro({ start }: { start: React.ReactNode }) {
  return (
    <Page>
      <h1 className="font-display text-4xl">実力診断</h1>
      <p className="mt-3 leading-relaxed">語彙の問題を解いて、日課をどこから始めるかを決めます。</p>
      <ul className="mt-4 space-y-2 leading-relaxed">
        <li>10問ずつ、3段に分かれています。問題は段ごとに難しくなります。</li>
        <li>段に届かなかったら、そこで終わります。最大30問です。</li>
        <li>分からない問題は「分からない」を選びます。当てずっぽうで答えるより、結果が正確になります。</li>
        <li>途中でやめても、つづきから受けられます。この解答は、復習の予定には入りません。</li>
      </ul>
      <div className="mt-6">{start}</div>
    </Page>
  );
}

function Test({
  session,
  onChange,
  onFinished,
}: {
  session: Session;
  onChange: (s: Session) => void;
  onFinished: () => void;
}) {
  const item = session.items[session.progress.answered];
  const answer = useMutation({
    mutationFn: (choice: number | null) => api.answerPlacement(session.id, item.position, choice, Date.now() - shownAt.current),
    onSuccess: (r) => {
      onChange({ ...session, progress: r.progress, placement: r.placement });
      if (r.progress.finished) onFinished();
    },
  });
  const shownAt = useRef(0);
  const sheet = useRef<HTMLDivElement>(null);

  // A new question starts at the top with focus on it, so screen readers announce it.
  useEffect(() => {
    shownAt.current = Date.now();
    window.scrollTo(0, 0);
    sheet.current?.focus({ preventScroll: true });
  }, [item?.position]);

  if (!item) return <Loading />;
  const index = item.position - 1;
  const block = Math.floor(index / BLOCK_SIZE);
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.metaKey || e.ctrlKey || e.altKey || answer.isPending) return;
    const n = Number(e.key);
    if (n >= 1 && n <= 4) answer.mutate(n - 1);
    if (n === 5) answer.mutate(null);
  };

  return (
    <Page>
      <p className="mb-2 flex items-baseline justify-between font-bold tabular-nums">
        <span>
          第{block + 1}段 {BLOCKS[block]}
        </span>
        <span>
          {index % BLOCK_SIZE + 1}／{BLOCK_SIZE}問
        </span>
      </p>
      <div ref={sheet} tabIndex={-1} onKeyDown={onKeyDown} className="outline-none">
        <Sheet>
          <Stem item={item} />
          <ul className="mt-5 space-y-1.5" aria-label="選択肢">
            {item.choices.map((c, i) => (
              <li key={c}>
                <button
                  onClick={() => answer.mutate(i)}
                  disabled={answer.isPending}
                  lang="en"
                  className="press flex min-h-12 w-full items-center border-3 border-ink px-3 text-left font-en text-lg hover:bg-tint disabled:cursor-default"
                >
                  {c}
                </button>
              </li>
            ))}
          </ul>
          <button
            onClick={() => answer.mutate(null)}
            disabled={answer.isPending}
            className="press mt-3 min-h-12 w-full border-3 border-hairline px-3 font-medium hover:bg-tint disabled:cursor-default"
          >
            分からない
          </button>
          {answer.isError && (
            <div className="mt-3">
              <Notice>{errorMessage(answer.error)}</Notice>
            </div>
          )}
        </Sheet>
      </div>
      <p className="mt-3 text-sm text-muted">
        途中でやめても、つづきから受けられます。
        <a href="/app" className="font-medium text-ink underline underline-offset-4">
          ホームに戻る
        </a>
      </p>
    </Page>
  );
}

function Stem({ item }: { item: Item }) {
  const [before, after] = item.stem.split(/\(\s*\)/);
  return (
    <p lang="en" className="font-en text-xl leading-relaxed">
      {before}
      <span className="mx-1 inline-block w-20 border-b-3 border-ink align-baseline">
        <span className="sr-only">（空所）</span>&nbsp;
      </span>
      {after}
    </p>
  );
}

function Result({ stage, blocks, start }: { stage: number; blocks: BlockScore[] | null; start: React.ReactNode }) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    window.scrollTo(0, 0);
    heading.current?.focus({ preventScroll: true });
  }, []);
  const s = STAGES[stage];
  return (
    <Page>
      <div className="border-3 border-ink bg-spot p-5 pb-6 text-on-spot">
        <h1 ref={heading} tabIndex={-1} className="font-display text-3xl leading-tight outline-none sm:text-4xl">
          <span className="block text-xl">実力診断の結果</span>
          {s.title}
        </h1>
        <p className="mt-2 font-medium leading-relaxed">{s.body}</p>
      </div>
      <ol aria-label="各段の結果" className="mt-1 grid grid-cols-3 gap-1">
        {BLOCKS.map((name, i) => {
          const b = blocks?.[i];
          const passed = i < stage;
          return (
            <li key={name} className={`border-3 border-ink p-2 ${passed ? "bg-surface" : "tone bg-surface"}`}>
              <span className="inline-block bg-surface px-1 text-sm font-bold">{name}</span>
              <br />
              <span className="inline-block bg-surface px-1 font-bold">{passed ? "通過" : i === stage ? "ここから" : "この先"}</span>
              {b && (
                <span className="block bg-surface px-1 text-sm tabular-nums">
                  {b.correct}／{b.total}問正解
                </span>
              )}
            </li>
          );
        })}
      </ol>
      <ButtonLink href="/app" className="mt-1 min-h-16 w-full text-xl">
        今日の日課へ
      </ButtonLink>
      <p className="mt-4 text-sm leading-relaxed text-muted">
        この結果は語彙だけを測った目安で、合否の予測ではありません。文法と長文は測っていません。「形式を選んで解く」からは、結果にかかわらずすべての問題を解けます。
      </p>
      <div className="mt-4">{start}</div>
    </Page>
  );
}
