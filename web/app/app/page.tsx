"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button, Loading, Notice, Page, PlusMinus } from "@/components/ui";
import {
  api,
  errorMessage,
  isPassage,
  type Daily,
  type PracticeType,
  type SessionSummary,
  type TypeProgress,
} from "@/lib/api";
import { useTitle } from "@/lib/title";

const COUNTS = [5, 10, 20];
const PASSAGE_COUNTS = [1, 2, 3];

const howTo: Record<PracticeType["format"], string> = {
  mcq: "英文の空所に入る語を4つから選びます。",
  vocab_meaning: "英単語を見て、意味を4つから選びます。",
  vocab_word: "意味を見て、英単語を4つから選びます。",
  passage_blank: "長文を読み、空所に入る語句を4つから選びます。1題に数問あります。",
  passage_question: "長文を読み、内容に合う答えを4つから選びます。1題に数問あります。",
};

const dateFmt = new Intl.DateTimeFormat("ja-JP", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });

/** The screen answers one question: what do I solve today? */
export default function Home() {
  useTitle("今日の日課");
  const daily = useQuery({ queryKey: ["daily"], queryFn: api.daily });
  const types = useQuery({ queryKey: ["practice-types"], queryFn: api.practiceTypes });
  const recent = useQuery({ queryKey: ["recent-sessions"], queryFn: api.recentSessions });
  const progress = useQuery({ queryKey: ["progress"], queryFn: api.progress });

  if (daily.isPending || types.isPending || recent.isPending) return <Loading />;

  const sessions = recent.data?.sessions ?? [];
  const mastered = progress.data?.progress.reduce((sum, p) => sum + p.mastered, 0);
  return (
    <Page wide>
      <div className="lg:grid lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)] lg:items-start lg:gap-8">
        <div>
          {daily.isError ? <Notice>{errorMessage(daily.error)}</Notice> : <TodayPanel daily={daily.data} />}
          {progress.data && (
            <p className="mt-4 text-sm leading-relaxed">
              これまでに習得した問題は <span className="font-display text-base tabular-nums">{mastered}</span> 問です。
              <span className="text-muted">習得は、日をあけて3回続けて正解した問題の数です。</span>
            </p>
          )}
        </div>

        <div className="mt-10 lg:mt-0">
          <h2 className="mb-3 font-display text-2xl">形式を選んで解く</h2>
          {types.isError ? (
            <Notice>{errorMessage(types.error)}</Notice>
          ) : types.data.types.length === 0 ? (
            <Notice kind="info">公開中の問題はまだありません。</Notice>
          ) : (
            <div className="border-3 border-ink">
              {types.data.types.map((t, i) => (
                <FormatRow
                  key={t.code}
                  type={t}
                  first={i === 0}
                  progress={progress.data?.progress.find((p) => p.type_code === t.code)}
                />
              ))}
            </div>
          )}

          <h2 className="mt-10 mb-3 font-display text-2xl">これまでの演習</h2>
          {recent.isError ? (
            <Notice>{errorMessage(recent.error)}</Notice>
          ) : sessions.length === 0 ? (
            <p className="text-muted">まだ記録はありません。1回解くと、ここに採点結果が残ります。</p>
          ) : (
            <History sessions={sessions} />
          )}
        </div>
      </div>
    </Page>
  );
}

/** Rough seconds per question and per passage. They only set panel widths; they are not shown as a promise. */
const SECONDS_PER_QUESTION = 20;
const SECONDS_PER_PASSAGE = 240;

/** Today's episode. Each panel of the strip is as wide as its part of the routine is long. */
function TodayPanel({ daily }: { daily: Daily }) {
  const router = useRouter();
  const qc = useQueryClient();
  const start = useMutation({
    mutationFn: api.startDaily,
    onSuccess: (s) => {
      qc.invalidateQueries({ queryKey: ["daily"] });
      router.push(`/app/practice/${s.id}`);
    },
  });
  const p = daily.session;
  const done = p?.finished ?? false;
  const reviewOnly = !done && daily.new === 0 && daily.reviews > 0;

  if (daily.total === 0) return <Notice kind="info">今日の日課に出せる問題がまだありません。</Notice>;

  const label = start.isPending
    ? "用意しています…"
    : done
      ? "今日の結果を見る"
      : p && p.answered > 0
        ? `つづきから（${p.answered}／${p.total}問）`
        : "日課をはじめる";

  return (
    <section aria-labelledby="today">
      <div className={`border-3 border-ink p-5 pb-6 ${done ? "tone bg-surface" : "bg-spot text-on-spot"}`}>
        <div className={done ? "inline-block bg-surface px-2 py-1 text-ink" : ""}>
          <h1 id="today" className="font-display text-5xl leading-tight">
            <span className="block text-2xl">第{daily.number}話</span>
            {done ? "読了" : `今日の${daily.total}問`}
          </h1>
          <p className="mt-2 font-medium">
            {done ? "次の話は、明日の朝4時から読めます。" : reviewOnly ? "今日は復習の回です。" : "語彙と長文を、混ぜて出します。"}
          </p>
        </div>
      </div>

      <ul aria-label="今日の内容" className="mt-1 flex gap-1">
        {daily.reviews > 0 && <Strip seconds={daily.reviews * SECONDS_PER_QUESTION} name="復習" value={`${daily.reviews}問`} tone />}
        {daily.new > 0 && <Strip seconds={daily.new * SECONDS_PER_QUESTION} name="新規" value={`${daily.new}問`} />}
        {daily.passage && <Strip seconds={SECONDS_PER_PASSAGE} name="長文" value="1題" />}
      </ul>

      {start.isError && (
        <div className="mt-2">
          <Notice>{errorMessage(start.error)}</Notice>
        </div>
      )}
      <Button
        className="mt-1 min-h-16 w-full text-xl"
        disabled={start.isPending}
        onClick={() => (daily.session_id ? router.push(`/app/practice/${daily.session_id}`) : start.mutate())}
      >
        {label}
      </Button>
    </section>
  );
}

function Strip({ seconds, name, value, tone = false }: { seconds: number; name: string; value: string; tone?: boolean }) {
  return (
    <li style={{ flexGrow: seconds, flexBasis: 0 }} className={`min-w-14 border-3 border-ink bg-surface p-1.5 ${tone ? "tone" : ""}`}>
      <span className="inline-block bg-surface px-1 text-sm font-bold whitespace-nowrap">{name}</span>
      <br />
      <span className="inline-block bg-surface px-1 font-display text-base tabular-nums whitespace-nowrap">{value}</span>
    </li>
  );
}

function FormatRow({ type, first, progress }: { type: PracticeType; first: boolean; progress?: TypeProgress }) {
  const router = useRouter();
  const counts = isPassage(type.format) ? PASSAGE_COUNTS : COUNTS;
  const unit = isPassage(type.format) ? "題" : "問";
  const [count, setCount] = useState(counts[1]);
  const start = useMutation({
    mutationFn: () => api.startSession(type.code, count),
    onSuccess: (s) => router.push(`/app/practice/${s.id}`),
  });
  return (
    <details className={`group ${first ? "" : "border-t-3 border-ink"}`}>
      <summary className="flex min-h-14 cursor-pointer items-center justify-between gap-3 px-3 py-2 hover:bg-tint">
        <span className="min-w-0">
          <span className="block font-bold">{type.name_ja}</span>
          <span className="block text-sm text-muted tabular-nums">
            {type.grade_name_ja}・全{type.question_count}問
            {progress && progress.seen > 0 && `・習得 ${progress.mastered}問`}
          </span>
        </span>
        <span className="group-open:hidden">
          <PlusMinus open={false} />
        </span>
        <span className="hidden group-open:inline">
          <PlusMinus open />
        </span>
      </summary>
      <div className="px-3 pb-3">
        <p className="text-sm leading-relaxed text-muted">{howTo[type.format]}</p>
        <fieldset className="mt-3">
          <legend className="sr-only">問題数</legend>
          <div className="grid grid-cols-3 gap-1">
            {counts.map((n) => (
              <label
                key={n}
                className="flex min-h-11 cursor-pointer items-center justify-center border-2 border-ink font-bold has-[:checked]:bg-spot has-[:checked]:text-on-spot has-[:focus-visible]:outline-3 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-ink"
              >
                <input
                  type="radio"
                  name={`count-${type.code}`}
                  value={n}
                  checked={count === n}
                  onChange={() => setCount(n)}
                  className="sr-only"
                />
                {n}
                {unit}
              </label>
            ))}
          </div>
        </fieldset>
        {start.isError && (
          <div className="mt-2">
            <Notice>{errorMessage(start.error)}</Notice>
          </div>
        )}
        <Button className="mt-2 w-full" onClick={() => start.mutate()} disabled={start.isPending}>
          {start.isPending ? "用意しています…" : `${count}${unit}を解く`}
        </Button>
      </div>
    </details>
  );
}

function History({ sessions }: { sessions: SessionSummary[] }) {
  return (
    <ul className="border-3 border-ink">
      {sessions.map((s, i) => {
        const p = s.progress;
        return (
          <li key={s.id} className={i > 0 ? "border-t-2 border-hairline" : ""}>
            <Link href={`/app/practice/${s.id}`} className="flex min-h-14 items-center gap-3 px-3 py-2 hover:bg-tint">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{s.type_name_ja}</p>
                <p className="text-sm text-muted">
                  <time dateTime={s.started_at}>{dateFmt.format(new Date(s.started_at))}</time>
                </p>
              </div>
              {p.finished ? (
                <p className="text-right tabular-nums">
                  <span className="font-display text-lg">{p.correct}</span>
                  <span className="text-muted">／{p.answered}問</span>
                </p>
              ) : (
                <p className="text-right text-sm font-bold">
                  続きから
                  <span className="block font-normal text-muted tabular-nums">
                    {p.answered}／{p.total}問
                  </span>
                </p>
              )}
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
