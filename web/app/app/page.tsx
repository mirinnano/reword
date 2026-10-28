"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button, Loading, Notice, Page, Sheet } from "@/components/ui";
import { api, errorMessage, type PracticeType, type SessionSummary, type TypeProgress } from "@/lib/api";

const COUNTS = [5, 10, 20];

const howTo: Record<PracticeType["format"], string> = {
  mcq: "英文の空所に入る語を4つから選びます。",
  vocab_meaning: "英単語を見て、意味を4つから選びます。",
  vocab_word: "意味を見て、英単語を4つから選びます。",
};

const dateFmt = new Intl.DateTimeFormat("ja-JP", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" });

/** The screen answers one question: what do I solve today? */
export default function Dashboard() {
  const types = useQuery({ queryKey: ["practice-types"], queryFn: api.practiceTypes });
  const recent = useQuery({ queryKey: ["recent-sessions"], queryFn: api.recentSessions });
  const progress = useQuery({ queryKey: ["progress"], queryFn: api.progress });

  if (types.isPending || recent.isPending) return <Loading />;

  const sessions = recent.data?.sessions ?? [];
  return (
    <Page>
      <h1 className="text-2xl font-bold">今日の演習</h1>
      <p className="mt-2 leading-relaxed text-muted">{finding(sessions)}</p>

      <div className="mt-6 space-y-3">
        {types.isError ? (
          <Notice>{errorMessage(types.error)}</Notice>
        ) : types.data.types.length === 0 ? (
          <Notice kind="info">公開中の問題はまだありません。</Notice>
        ) : (
          types.data.types.map((t) => (
            <StartSheet key={t.code} type={t} progress={progress.data?.progress.find((p) => p.type_code === t.code)} />
          ))
        )}
      </div>

      <p className="mt-3 text-sm leading-relaxed text-muted">
        「習得」は、日をあけて3回続けて正解した問題の数です。正解した問題は1日後、3日後、7日後…と間隔をあけてもう一度出題し、間違えた問題は次の演習で復習します。
      </p>

      <h2 className="mt-12 mb-3 text-lg font-bold">これまでの演習</h2>
      {recent.isError ? (
        <Notice>{errorMessage(recent.error)}</Notice>
      ) : sessions.length === 0 ? (
        <p className="text-muted">まだ記録はありません。1回解くと、ここに採点結果が残ります。</p>
      ) : (
        <History sessions={sessions} />
      )}
    </Page>
  );
}

/** One honest line drawn from the latest finished session. */
function finding(sessions: SessionSummary[]): string {
  const last = sessions.find((s) => s.progress.finished && s.progress.answered > 0);
  if (!last) return "1回10問、5分ほどで解けます。間違えた問題は次の回にもう一度出ます。";
  const { answered, correct } = last.progress;
  const missed = answered - correct;
  if (missed === 0) return `前回は${answered}問すべて正解。今回は新しい問題を中心に出します。`;
  return `前回は${answered}問中${correct}問正解。間違えた${missed}問は、次からの演習で優先して復習します。`;
}

function StartSheet({ type, progress }: { type: PracticeType; progress?: TypeProgress }) {
  const router = useRouter();
  const [count, setCount] = useState(10);
  const start = useMutation({
    mutationFn: () => api.startSession(type.code, count),
    onSuccess: (s) => router.push(`/app/practice/${s.id}`),
  });
  return (
    <Sheet>
      <h2 className="text-lg font-bold">
        {type.grade_name_ja}　{type.name_ja}
      </h2>
      <p className="mt-1 text-sm text-muted">
        全{type.question_count}問。{howTo[type.format]}
      </p>
      {progress && progress.seen > 0 && (
        <p className="mt-2 text-sm tabular-nums">
          習得 <span className="font-bold">{progress.mastered}</span>／{progress.total}問
          {progress.due > 0 && <span className="text-accent">・今日の復習 {progress.due}問</span>}
        </p>
      )}
      <fieldset className="mt-4">
        <legend className="sr-only">問題数</legend>
        <div className="grid grid-cols-3 gap-2">
          {COUNTS.map((n) => (
            <label
              key={n}
              className="flex min-h-11 cursor-pointer items-center justify-center rounded-[var(--radius-control)] border border-line text-base has-[:checked]:border-accent has-[:checked]:bg-accent has-[:checked]:text-on-accent has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent"
            >
              <input type="radio" name={`count-${type.code}`} value={n} checked={count === n} onChange={() => setCount(n)} className="sr-only" />
              {n}問
            </label>
          ))}
        </div>
      </fieldset>
      {start.isError && (
        <div className="mt-3">
          <Notice>{errorMessage(start.error)}</Notice>
        </div>
      )}
      <Button className="mt-3 w-full" onClick={() => start.mutate()} disabled={start.isPending}>
        {start.isPending ? "問題を用意しています…" : `${count}問を解く`}
      </Button>
    </Sheet>
  );
}

function History({ sessions }: { sessions: SessionSummary[] }) {
  return (
    <ul className="overflow-hidden rounded-[var(--radius-sheet)] bg-surface">
      {sessions.map((s, i) => {
        const p = s.progress;
        return (
          <li key={s.id} className={i > 0 ? "border-t border-hairline" : ""}>
            <Link href={`/app/practice/${s.id}`} className="flex min-h-16 items-center gap-3 px-4 py-3 hover:bg-ground">
              <div className="min-w-0 flex-1">
                <p className="truncate">{s.type_name_ja}</p>
                <p className="text-sm text-muted">
                  <time dateTime={s.started_at}>{dateFmt.format(new Date(s.started_at))}</time>
                </p>
              </div>
              {p.finished ? (
                <p className="text-right tabular-nums">
                  <span className="text-lg font-bold">{p.correct}</span>
                  <span className="text-muted">／{p.answered}問</span>
                </p>
              ) : (
                <p className="text-right text-sm text-accent">
                  続きから
                  <span className="block text-muted tabular-nums">
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
