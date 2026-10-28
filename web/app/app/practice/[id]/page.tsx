"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useId, useRef, useState } from "react";
import { Button, Loading, Mark, Notice, Page, Sheet } from "@/components/ui";
import { api, errorMessage, type Feedback, type Format, type Item, type Session, type Vocab } from "@/lib/api";

export default function PracticePage() {
  const id = Number(useParams<{ id: string }>().id);
  const qc = useQueryClient();
  const key = ["session", id];
  const session = useQuery({ queryKey: key, queryFn: () => api.session(id), enabled: Number.isFinite(id) });
  // Position whose feedback is on screen; null means "show the next unanswered".
  const [showing, setShowing] = useState<number | null>(null);

  if (session.isPending) return <Loading />;
  if (session.isError) {
    return (
      <Page narrow>
        <Notice>{errorMessage(session.error)}</Notice>
        <p className="mt-4 text-center">
          <Link href="/app" className="text-accent underline underline-offset-4">
            今日の演習に戻る
          </Link>
        </p>
      </Page>
    );
  }
  const s = session.data;
  const next = s.items.find((it) => !it.feedback);
  const current = showing !== null ? s.items.find((it) => it.position === showing) : next;

  if (!current || (s.progress.finished && showing === null)) return <Summary session={s} />;

  const update = (position: number, feedback: Feedback, progress: Session["progress"]) =>
    qc.setQueryData<Session>(key, (old) =>
      old && {
        ...old,
        progress,
        items: old.items.map((it) => (it.position === position ? { ...it, feedback } : it)),
      },
    );

  return (
    <Page narrow>
      <ProgressBar session={s} position={current.position} />
      <Question
        key={current.position}
        sessionId={s.id}
        format={s.format}
        item={current}
        isLast={!s.items.some((it) => !it.feedback && it.position !== current.position)}
        onAnswered={(fb, progress) => {
          update(current.position, fb, progress);
          setShowing(current.position);
        }}
        onNext={() => setShowing(null)}
      />
      <QuitButton session={s} onDone={(fin) => qc.setQueryData(key, fin)} />
    </Page>
  );
}

function ProgressBar({ session, position }: { session: Session; position: number }) {
  const { total, answered } = session.progress;
  return (
    <div className="mb-4">
      <div className="mb-1.5 flex justify-between text-sm text-muted">
        <span className="truncate">{session.type_name_ja}</span>
        <span className="shrink-0 tabular-nums">
          第{position}問／{total}問
        </span>
      </div>
      <div
        role="progressbar"
        aria-label="回答済みの問題"
        aria-valuemin={0}
        aria-valuemax={total}
        aria-valuenow={answered}
        className="h-1.5 overflow-hidden rounded-full bg-hairline"
      >
        <div className="h-full bg-accent transition-[width]" style={{ width: `${(answered / total) * 100}%` }} />
      </div>
    </div>
  );
}

/**
 * The stem as a line on an answer sheet. Unanswered: an empty blank.
 * Answered: the correct word written in blue ink, the learner's wrong word
 * struck through in red beside it.
 */
function Stem({ stem, item, format }: { stem: string; item: Item; format: Format }) {
  const fb = item.feedback;
  // Drills show a single prompt: an English headword, or Japanese meanings.
  if (format === "vocab_meaning") {
    return (
      <p lang="en" className="font-sheet text-3xl font-semibold">
        {stem}
      </p>
    );
  }
  if (format === "vocab_word") {
    return <p className="text-xl leading-relaxed">{stem}</p>;
  }
  const [before, after] = stem.split(/\(\s*\)/);
  return (
    <p lang="en" className="font-sheet text-xl leading-[1.7]">
      {before}
      {!fb ? (
        <span className="mx-1 inline-block w-20 border-b-2 border-line align-baseline" aria-label="空所">
          &nbsp;
        </span>
      ) : (
        <span className="mx-1">
          {!fb.correct && (
            <>
              <del className="text-ng decoration-2">{item.choices[fb.selected]}</del>{" "}
            </>
          )}
          <ins className="font-semibold text-accent no-underline">{item.choices[fb.correct_choice]}</ins>
        </span>
      )}
      {after}
    </p>
  );
}

function Question({
  sessionId,
  format,
  item,
  isLast,
  onAnswered,
  onNext,
}: {
  sessionId: number;
  format: Format;
  item: Item;
  isLast: boolean;
  onAnswered: (fb: Feedback, progress: Session["progress"]) => void;
  onNext: () => void;
}) {
  const shownAt = useRef(0);
  const sheet = useRef<HTMLDivElement>(null);
  const result = useRef<HTMLHeadingElement>(null);
  const hintId = useId();
  const fb = item.feedback;

  // A new question starts at the top with focus on its sheet, so screen
  // readers announce it and the 1–4 shortcuts (scoped to this region) work.
  useEffect(() => {
    shownAt.current = Date.now();
    window.scrollTo(0, 0);
    sheet.current?.focus({ preventScroll: true });
  }, []);

  const answer = useMutation({
    mutationFn: (choice: number) => api.answer(sessionId, item.position, choice, Date.now() - shownAt.current),
    onSuccess: (r) => onAnswered(r.feedback, r.progress),
  });

  // After answering, move focus to the result so it is read out first.
  useEffect(() => {
    if (fb) result.current?.focus();
  }, [fb]);

  const choose = (i: number) => {
    if (!fb && !answer.isPending) answer.mutate(i);
  };

  // Single-key shortcuts only while focus is inside this question (WCAG 2.1.4).
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.altKey || e.ctrlKey || e.metaKey) return;
    const n = Number(e.key);
    if (!fb && n >= 1 && n <= item.choices.length) {
      e.preventDefault();
      choose(n - 1);
    } else if (fb && e.key === "Enter" && e.target === result.current) {
      e.preventDefault();
      onNext();
    }
  };

  return (
    <div onKeyDown={onKeyDown}>
      <div ref={sheet} tabIndex={-1} aria-describedby={hintId} className="rounded-[var(--radius-sheet)] outline-offset-4">
        {/* Answer-sheet margin: the question number, graded with a red-pen mark. */}
        <Sheet className="flex gap-3 pl-3">
          <div className="w-9 shrink-0 text-center">
            {fb ? (
              <Mark correct={fb.correct} className="size-9 text-3xl" />
            ) : (
              <span aria-hidden="true" className="inline-block pt-1 text-sm text-muted tabular-nums">
                ({item.position})
              </span>
            )}
          </div>
          <div className="min-w-0 flex-1">
            <h1 className="sr-only">第{item.position}問</h1>
            <Stem stem={item.stem} item={item} format={format} />
          </div>
        </Sheet>
      </div>
      <p id={hintId} className="mt-2 hidden text-sm text-muted [@media(pointer:fine)]:block">
        キーボードでは 1〜4 で選べます
      </p>

      <ul className="mt-4 space-y-2" aria-label="選択肢">
        {item.choices.map((c, i) => {
          const isCorrect = fb && i === fb.correct_choice;
          const isWrongPick = fb && !fb.correct && i === fb.selected;
          let style = "border-line bg-surface hover:border-accent";
          if (fb) style = isCorrect ? "border-ok bg-ok-soft" : isWrongPick ? "border-ng bg-ng-soft" : "border-hairline bg-surface text-muted";
          return (
            <li key={i}>
              <button
                onClick={() => choose(i)}
                disabled={!!fb || answer.isPending}
                className={`flex min-h-12 w-full items-center gap-3 rounded-[var(--radius-control)] border-2 px-3 text-left transition-colors disabled:cursor-default ${style}`}
              >
                <span aria-hidden="true" className="w-4 shrink-0 text-center text-sm text-muted tabular-nums">
                  {i + 1}
                </span>
                <span lang={format === "vocab_meaning" ? "ja" : "en"} className="flex-1 text-lg">
                  {c}
                </span>
                {isCorrect && <span className="text-sm font-medium text-ok">正解</span>}
                {isWrongPick && <span className="text-sm font-medium text-ng">あなたの回答</span>}
              </button>
            </li>
          );
        })}
      </ul>

      {answer.isError && (
        <div className="mt-3">
          <Notice>{errorMessage(answer.error)}</Notice>
        </div>
      )}

      {fb && (
        <div className="mt-6 space-y-3">
          <h2 ref={result} tabIndex={-1} className={`scroll-mt-20 text-lg font-bold ${fb.correct ? "text-ok" : "text-ng"}`}>
            {fb.correct ? (
              "正解"
            ) : (
              <>
                不正解。正解は <span lang={format === "vocab_meaning" ? "ja" : "en"}>{item.choices[fb.correct_choice]}</span>
              </>
            )}
          </h2>
          <Explanation fb={fb} />
          <Button className="w-full" onClick={onNext}>
            {isLast ? "採点結果を見る" : "次の問題へ"}
          </Button>
        </div>
      )}
    </div>
  );
}

function Explanation({ fb }: { fb: Feedback }) {
  return (
    <>
      <Sheet>
        <h3 className="mb-1.5 font-bold">解説</h3>
        <p className="leading-relaxed">{fb.explanation_ja}</p>
      </Sheet>
      {fb.vocab.map((v) => (
        <VocabSheet key={v.key} v={v} />
      ))}
    </>
  );
}

const posJa: Record<string, string> = {
  noun: "名詞",
  verb: "動詞",
  adjective: "形容詞",
  adverb: "副詞",
  phrase: "熟語",
  other: "その他",
};

function VocabSheet({ v }: { v: Vocab }) {
  return (
    <Sheet>
      <h3 className="flex items-baseline gap-2">
        <span lang="en" className="font-sheet text-xl font-semibold">
          {v.headword}
        </span>
        <span className="text-sm text-muted">{posJa[v.pos] ?? v.pos}</span>
      </h3>
      <p className="mt-1">{v.meanings_ja.join("／")}</p>
      <p lang="en" className="mt-3 font-sheet text-lg leading-relaxed">
        {v.example_en}
      </p>
      <p className="mt-0.5 text-sm text-muted">{v.example_ja}</p>
    </Sheet>
  );
}

function QuitButton({ session, onDone }: { session: Session; onDone: (s: Session) => void }) {
  const quit = useMutation({ mutationFn: () => api.finish(session.id), onSuccess: onDone });
  const { answered, total, finished } = session.progress;
  if (finished) return null;
  const left = total - answered;
  return (
    <div className="mt-10 text-center">
      <button
        onClick={() => {
          if (confirm(`残りの${left}問を解かずに終えますか？解いた${answered}問の結果は記録されます。`)) quit.mutate();
        }}
        className="min-h-11 px-3 text-sm text-muted underline underline-offset-4"
      >
        ここで終える
      </button>
    </div>
  );
}

/** One line per question on the graded sheet: the sentence or prompt with its answer. */
function reviewLine(format: Format, it: Item): string {
  const fb = it.feedback;
  if (!fb) return it.stem;
  const answer = it.choices[fb.correct_choice];
  if (format === "mcq") return it.stem.replace(/\(\s*\)/, `(${answer})`);
  return `${it.stem} — ${answer}`;
}

/** The graded answer sheet. */
function Summary({ session }: { session: Session }) {
  const router = useRouter();
  const qc = useQueryClient();
  const [open, setOpen] = useState<number | null>(null);
  useEffect(() => {
    window.scrollTo(0, 0);
  }, []);
  const { correct, answered, total } = session.progress;
  const again = useMutation({
    mutationFn: () => api.startSession(session.type_code, total),
    onSuccess: (s) => {
      qc.invalidateQueries({ queryKey: ["recent-sessions"] });
      qc.invalidateQueries({ queryKey: ["progress"] });
      router.push(`/app/practice/${s.id}`);
    },
  });
  const missed = answered - correct;

  return (
    <Page narrow>
      <Sheet>
        <h1 className="text-sm text-muted">{session.type_name_ja}　採点結果</h1>
        <p className="mt-2 flex items-baseline gap-2">
          <span className="text-6xl font-bold text-ng tabular-nums">{correct}</span>
          <span className="text-2xl text-muted tabular-nums">／{answered}</span>
        </p>
        <p className="mt-2 leading-relaxed">
          {answered === 0
            ? "1問も解かずに終えました。"
            : missed === 0
              ? "全問正解です。次の回は新しい問題を中心に出します。"
              : `間違えた${missed}問は、次からの演習で優先して復習します。`}
          {answered < total && `（${total - answered}問は未回答）`}
        </p>
      </Sheet>

      <h2 className="mt-8 mb-2 font-bold">ふりかえり</h2>
      <ol className="space-y-2">
        {session.items.map((it) => {
          const fb = it.feedback;
          const panelId = `review-${it.position}`;
          return (
            <li key={it.position} className="rounded-[var(--radius-sheet)] bg-surface">
              <button
                onClick={() => setOpen(open === it.position ? null : it.position)}
                disabled={!fb}
                aria-expanded={fb ? open === it.position : undefined}
                aria-controls={fb ? panelId : undefined}
                className="flex w-full items-start gap-3 p-4 text-left"
              >
                {fb ? (
                  <Mark correct={fb.correct} className="w-6 shrink-0 text-2xl" />
                ) : (
                  <span aria-hidden="true" className="w-6 shrink-0 text-center text-2xl text-muted">
                    －
                  </span>
                )}
                <span className="sr-only">{!fb ? "未回答" : fb.correct ? "正解" : "不正解"}</span>
                <span lang={session.format === "vocab_word" ? "ja" : "en"} className="font-sheet text-lg leading-relaxed">
                  {reviewLine(session.format, it)}
                </span>
              </button>
              {open === it.position && fb && (
                <div id={panelId} className="space-y-2 px-4 pb-4">
                  {!fb.correct && (
                    <p className="text-sm">
                      あなたの回答：
                      <span lang="en" className="text-ng line-through">
                        {it.choices[fb.selected]}
                      </span>
                    </p>
                  )}
                  <p className="leading-relaxed">{fb.explanation_ja}</p>
                  {fb.vocab.map((v) => (
                    <VocabSheet key={v.key} v={v} />
                  ))}
                </div>
              )}
            </li>
          );
        })}
      </ol>

      <div className="mt-8 space-y-2">
        {again.isError && <Notice>{errorMessage(again.error)}</Notice>}
        <Button className="w-full" onClick={() => again.mutate()} disabled={again.isPending}>
          {again.isPending ? "問題を用意しています…" : `続けて${total}問を解く`}
        </Button>
        <Button
          variant="secondary"
          className="w-full"
          onClick={() => {
            qc.invalidateQueries({ queryKey: ["recent-sessions"] });
      qc.invalidateQueries({ queryKey: ["progress"] });
            router.push("/app");
          }}
        >
          今日の演習に戻る
        </Button>
      </div>
    </Page>
  );
}
