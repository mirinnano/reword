"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useId, useRef, useState, type RefObject } from "react";
import { Burst, Button, ButtonLink, Loading, Mark, Notice, Page, Sheet } from "@/components/ui";
import { api, errorMessage, isPassage, type Feedback, type Format, type Item, type Session, type Vocab } from "@/lib/api";
import { useTitle } from "@/lib/title";

type Keys = (e: React.KeyboardEvent) => void;

export default function PracticePage() {
  const param = useParams<{ id: string }>().id;
  const id = Number(param);
  const valid = Number.isInteger(id) && id > 0;
  const qc = useQueryClient();
  const key = ["session", id];
  const session = useQuery({ queryKey: key, queryFn: () => api.session(id), enabled: valid });
  // Position whose feedback is on screen; null means "show the next unanswered".
  const [showing, setShowing] = useState<number | null>(null);
  // The current question's shortcuts, so they also work from the passage beside it.
  const questionKeys = useRef<Keys | null>(null);
  useTitle(session.data?.type_name_ja ?? "演習");

  if (valid && session.isPending) return <Loading />;
  if (!valid || session.isError) {
    return (
      <Page narrow>
        <h1 className="mb-3 font-display text-2xl">演習を開けませんでした</h1>
        <Notice>{valid ? errorMessage(session.error) : "この演習は見つかりませんでした。"}</Notice>
        <p className="mt-4 text-center">
          <Link href="/app" className="font-medium underline decoration-2 underline-offset-4">
            今日の日課に戻る
          </Link>
        </p>
      </Page>
    );
  }
  const s = session.data;
  if (!s) return <Loading />;
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

  const question = (
    <Question
      key={current.position}
      sessionId={s.id}
      item={current}
      keysRef={questionKeys}
      isLast={!s.items.some((it) => !it.feedback && it.position !== current.position)}
      onAnswered={(fb, progress) => {
        update(current.position, fb, progress);
        setShowing(current.position);
        qc.invalidateQueries({ queryKey: ["daily"] });
      }}
      onNext={() => setShowing(null)}
    />
  );

  return (
    <Page wide>
      <ProgressStrip session={s} current={current} />
      {isPassage(current.format) ? (
        <div className="lg:grid lg:grid-cols-[minmax(0,1fr)_24rem] lg:items-start lg:gap-6 [@media(min-width:640px)_and_(max-height:500px)]:grid [@media(min-width:640px)_and_(max-height:500px)]:grid-cols-2 [@media(min-width:640px)_and_(max-height:500px)]:items-start [@media(min-width:640px)_and_(max-height:500px)]:gap-4">
          <PassagePanel
            key={current.group}
            current={current}
            items={groupItems(s, current.group)}
            onKeyDown={(e) => questionKeys.current?.(e)}
          />
          <div className="mt-4 lg:mt-0 [@media(min-width:640px)_and_(max-height:500px)]:mt-0">{question}</div>
        </div>
      ) : (
        <div className="mx-auto max-w-xl">{question}</div>
      )}
      <QuitButton session={s} onDone={(fin) => qc.setQueryData(key, fin)} />
    </Page>
  );
}

function groupItems(session: Session, group?: string): Item[] {
  return session.items.filter((it) => it.group === group);
}

function sectionName(session: Session, item: Item): string {
  if (session.format !== "daily") return session.type_name_ja;
  if (isPassage(item.format)) return "長文";
  return item.review ? "復習" : "新しい問題";
}

/** One panel per question: a page of the episode filling up as it is solved. */
function ProgressStrip({ session, current }: { session: Session; current: Item }) {
  const { total, answered } = session.progress;
  const mode = session.format === "daily" ? null : isPassage(session.format) ? "passage" : null;
  const groups = mode === "passage" ? [...new Set(session.items.map((it) => it.group))] : [];
  return (
    <div className="mb-4">
      <div className="mb-1.5 flex justify-between gap-3 text-sm">
        <span className="truncate font-bold">{sectionName(session, current)}</span>
        <span className="shrink-0 tabular-nums">
          {mode === "passage"
            ? `${groups.indexOf(current.group) + 1}題目／${groups.length}題`
            : `第${current.position}問／${total}問`}
        </span>
      </div>
      <div
        role="progressbar"
        aria-label="回答済みの問題"
        aria-valuemin={0}
        aria-valuemax={total}
        aria-valuenow={answered}
        className="flex h-3 gap-0.5"
      >
        {session.items.map((it) => (
          <span
            key={it.position}
            className={`min-w-0 flex-1 border-2 border-ink ${
              it.position === current.position ? "bg-spot" : it.feedback ? "bg-ink" : "bg-surface"
            }`}
          />
        ))}
      </div>
    </div>
  );
}

/**
 * The passage stays on screen beside the question (above it on a phone, in its
 * own scroll area), so answering never means scrolling away from the text.
 * This is the quiet panel: reading needs nothing but a frame and the words.
 * It outlives the questions of its passage and keeps its scroll position.
 */
function PassagePanel({ current, items, onKeyDown }: { current: Item; items: Item[]; onKeyDown: Keys }) {
  const panel = useRef<HTMLElement>(null);
  const blank = useRef<HTMLSpanElement>(null);

  // Bring the blank being asked into view, inside the panel only.
  useEffect(() => {
    const [p, b] = [panel.current, blank.current];
    if (p && b) p.scrollTop = b.offsetTop - p.clientHeight / 3;
  }, [current.seq]);

  return (
    <section
      ref={panel}
      tabIndex={0}
      onKeyDown={onKeyDown}
      aria-label={`本文：${current.title}`}
      className="relative max-h-[45dvh] overflow-y-auto border-3 border-ink bg-surface p-4 lg:sticky lg:top-20 lg:max-h-[calc(100dvh-6rem)] lg:p-6 [@media(min-width:640px)_and_(max-height:500px)]:max-h-[calc(100dvh-5rem)]"
    >
      <h2 lang="en" className="font-en text-xl font-bold">
        {current.title}
      </h2>
      <PassageText format={current.format} text={current.passage ?? ""} items={items} currentSeq={current.seq} blankRef={blank} />
    </section>
  );
}

/** Paragraphs of the passage; numbered blanks show how each was answered. */
function PassageText({
  format,
  text,
  items,
  currentSeq,
  blankRef,
}: {
  format: Format;
  text: string;
  items: Item[];
  currentSeq?: number;
  blankRef?: RefObject<HTMLSpanElement | null>;
}) {
  return (
    <div lang="en" className="mt-3 max-w-[70ch] space-y-4 font-en text-base leading-read lg:text-lg">
      {text.split(/\n\s*\n/).map((para, i) => (
        <p key={i} className="whitespace-pre-line">
          {format === "passage_blank"
            ? para.split(/\(\s*(\d{1,2})\s*\)/).map((part, j) =>
                j % 2 === 0 ? (
                  part
                ) : (
                  <Blank
                    key={j}
                    n={Number(part)}
                    item={items.find((it) => it.seq === Number(part))}
                    current={Number(part) === currentSeq}
                    blankRef={blankRef}
                  />
                ),
              )
            : para}
        </p>
      ))}
    </div>
  );
}

function Blank({
  n,
  item,
  current,
  blankRef,
}: {
  n: number;
  item?: Item;
  current: boolean;
  blankRef?: RefObject<HTMLSpanElement | null>;
}) {
  const fb = item?.feedback;
  if (item && fb) {
    return (
      <span ref={current ? blankRef : undefined} className="mx-0.5">
        {!fb.correct && (
          <>
            <del className="text-ng decoration-2">{item.choices[fb.selected]}</del>{" "}
          </>
        )}
        <ins className="font-bold no-underline">{item.choices[fb.correct_choice]}</ins>
      </span>
    );
  }
  return (
    <span
      ref={current ? blankRef : undefined}
      aria-current={current ? "true" : undefined}
      className={`mx-0.5 inline-block min-w-14 border-2 border-ink text-center font-bold tabular-nums ${
        current ? "bg-spot text-on-spot" : "bg-surface"
      }`}
    >
      ({n})
    </span>
  );
}

/** What the learner reads and answers. The blank and its answer live in the sentence, like a printed test. */
function Stem({ item }: { item: Item }) {
  const fb = item.feedback;
  const stem = item.stem;
  // Drills show a single prompt: an English headword, or Japanese meanings.
  if (item.format === "vocab_meaning") {
    return (
      <p lang="en" className="font-en text-3xl font-bold">
        {stem}
      </p>
    );
  }
  if (item.format === "vocab_word") return <p className="text-xl font-medium leading-relaxed">{stem}</p>;
  // The blank itself is in the passage; the question only says which one.
  if (item.format === "passage_blank") return <p className="text-lg font-medium leading-relaxed">({item.seq}) に入る語句を選びます。</p>;
  if (item.format === "passage_question") {
    return (
      <p lang="en" className="font-en text-xl font-bold leading-normal">
        {stem}
      </p>
    );
  }
  const [before, after] = stem.split(/\(\s*\)/);
  return (
    <p lang="en" className="font-en text-xl leading-relaxed">
      {before}
      {!fb ? (
        <span className="mx-1 inline-block w-20 border-b-3 border-ink align-baseline">
          <span className="sr-only">（空所）</span>&nbsp;
        </span>
      ) : (
        <span className="mx-1">
          {!fb.correct && (
            <>
              <del className="text-ng decoration-2">{item.choices[fb.selected]}</del>{" "}
            </>
          )}
          <ins className="font-bold no-underline">{item.choices[fb.correct_choice]}</ins>
        </span>
      )}
      {after}
    </p>
  );
}

function Question({
  sessionId,
  item,
  keysRef,
  isLast,
  onAnswered,
  onNext,
}: {
  sessionId: number;
  item: Item;
  keysRef: RefObject<Keys | null>;
  isLast: boolean;
  onAnswered: (fb: Feedback, progress: Session["progress"]) => void;
  onNext: () => void;
}) {
  const shownAt = useRef(0);
  const sheet = useRef<HTMLDivElement>(null);
  const result = useRef<HTMLHeadingElement>(null);
  const hintId = useId();
  const fb = item.feedback;
  const choicesLang = item.format === "vocab_meaning" ? "ja" : "en";

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
  const onKeyDown: Keys = (e) => {
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

  useEffect(() => {
    keysRef.current = onKeyDown;
  });

  return (
    <div onKeyDown={onKeyDown}>
      <div ref={sheet} tabIndex={-1} aria-describedby={hintId} className="outline-offset-4">
        <Sheet className="flex gap-3 pl-3">
          <div className="w-9 shrink-0 text-center">
            {fb ? (
              <Mark correct={fb.correct} className="stamp size-9 text-3xl" />
            ) : (
              <span aria-hidden="true" className="inline-block pt-1 font-bold tabular-nums">
                ({item.seq ?? item.position})
              </span>
            )}
          </div>
          <div className="min-w-0 flex-1">
            <h1 className="sr-only">第{item.position}問</h1>
            <Stem item={item} />
          </div>
        </Sheet>
      </div>
      <p id={hintId} className="mt-2 hidden text-sm text-muted [@media(pointer:fine)]:block">
        キーボードでは 1〜4 で選べます
      </p>

      <ul className="mt-3 space-y-1.5" aria-label="選択肢" aria-busy={answer.isPending}>
        {item.choices.map((c, i) => {
          const isCorrect = fb && i === fb.correct_choice;
          const isWrongPick = fb && !fb.correct && i === fb.selected;
          const isPending = answer.isPending && answer.variables === i;
          let style = "border-ink bg-surface hover:bg-tint";
          if (isPending) style = "border-ink bg-spot text-on-spot";
          if (fb) style = isCorrect ? "border-ok bg-ok-soft" : isWrongPick ? "border-ng bg-ng-soft" : "border-hairline bg-surface text-muted";
          return (
            <li key={i}>
              <button
                onClick={() => choose(i)}
                disabled={!!fb || answer.isPending}
                className={`press flex min-h-13 w-full items-center gap-3 border-3 px-3 py-2 text-left disabled:cursor-default ${style}`}
              >
                <span aria-hidden="true" className="w-4 shrink-0 text-center text-sm font-bold tabular-nums">
                  {i + 1}
                </span>
                <span lang={choicesLang} className="flex-1 font-en text-lg">
                  {c}
                </span>
                {isPending && <span className="text-sm font-bold">採点中</span>}
                {isCorrect && <span className="text-sm font-bold text-ok">正解</span>}
                {isWrongPick && <span className="text-sm font-bold text-ng">あなたの回答</span>}
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
        <div className="mt-5 space-y-3">
          <h2 ref={result} tabIndex={-1} className="scroll-mt-20 outline-offset-2">
            {fb.correct ? (
              <Burst>正解</Burst>
            ) : (
              <span className="stamp block">
                <span className="block font-display text-2xl text-ng">不正解</span>
                <span className="mt-1 block font-medium">
                  正解は <span lang={choicesLang} className="font-en font-bold">{item.choices[fb.correct_choice]}</span>。この問題はまた出題します。
                </span>
              </span>
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
        <span lang="en" className="font-en text-xl font-bold">
          {v.headword}
        </span>
        <span className="text-sm text-muted">{posJa[v.pos] ?? v.pos}</span>
      </h3>
      <p className="mt-1">{v.meanings_ja.join("／")}</p>
      <p lang="en" className="mt-3 font-en text-lg leading-relaxed">
        {v.example_en}
      </p>
      <p className="mt-0.5 text-sm text-muted">{v.example_ja}</p>
    </Sheet>
  );
}

/** Set apart from the answers, and confirmed in place, so it is never hit by accident. */
function QuitButton({ session, onDone }: { session: Session; onDone: (s: Session) => void }) {
  const [asking, setAsking] = useState(false);
  const quit = useMutation({ mutationFn: () => api.finish(session.id), onSuccess: onDone });
  const { answered, total, finished } = session.progress;
  if (finished) return null;
  const left = total - answered;
  if (!asking) {
    return (
      <div className="mt-12 text-center">
        <button onClick={() => setAsking(true)} className="min-h-11 px-3 text-sm text-muted underline underline-offset-4">
          ここで終える
        </button>
      </div>
    );
  }
  return (
    <div className="mx-auto mt-12 max-w-md" role="group" aria-label="演習を終える確認">
      <p className="text-center text-sm leading-relaxed">
        残りの{left}問は解かずに終えます。解いた{answered}問の結果は記録されます。
      </p>
      {quit.isError && <Notice>{errorMessage(quit.error)}</Notice>}
      <div className="mt-2 grid grid-cols-2 gap-1">
        <Button variant="secondary" onClick={() => setAsking(false)}>
          続ける
        </Button>
        <Button variant="secondary" onClick={() => quit.mutate()} disabled={quit.isPending}>
          {left}問残して終える
        </Button>
      </div>
    </div>
  );
}

/** One line per question on the graded sheet: the sentence or prompt with its answer. */
function reviewLine(it: Item): string {
  const fb = it.feedback;
  const stem = it.format === "passage_blank" ? `(${it.seq})` : it.stem;
  if (!fb) return stem;
  const answer = it.choices[fb.correct_choice];
  if (it.format === "mcq") return it.stem.replace(/\(\s*\)/, `(${answer})`);
  return `${stem} — ${answer}`;
}

/** The end of the episode: the score, what comes back, and every question to look over. */
function Summary({ session }: { session: Session }) {
  const router = useRouter();
  const qc = useQueryClient();
  const [open, setOpen] = useState<number | null>(null);
  const title = useRef<HTMLHeadingElement>(null);
  // The button that led here is gone, so focus moves to the result itself.
  useEffect(() => {
    window.scrollTo(0, 0);
    title.current?.focus({ preventScroll: true });
  }, []);
  const { correct, answered, total } = session.progress;
  const daily = session.format === "daily";
  const refresh = () => {
    qc.invalidateQueries({ queryKey: ["daily"] });
    qc.invalidateQueries({ queryKey: ["recent-sessions"] });
    qc.invalidateQueries({ queryKey: ["progress"] });
  };
  const passage = !daily && isPassage(session.format as Format);
  // A passage session is sized in passages, so "again" asks for as many of them.
  const againCount = passage ? new Set(session.items.map((it) => it.group)).size : total;
  const again = useMutation({
    mutationFn: () => api.startSession(session.type_code, againCount),
    onSuccess: (s) => {
      refresh();
      router.push(`/app/practice/${s.id}`);
    },
  });
  const missed = answered - correct;

  return (
    <Page narrow>
      <section className="border-3 border-ink bg-spot p-5 text-on-spot">
        <h1 ref={title} tabIndex={-1} className="font-bold outline-offset-4">
          {daily ? "今日の話、おわり" : `${session.type_name_ja}の採点結果`}
        </h1>
        <p className="mt-2 flex items-baseline gap-2">
          <span className="font-display text-6xl tabular-nums">{correct}</span>
          <span className="font-display text-2xl tabular-nums">／{answered}問</span>
        </p>
        <p className="mt-2 font-medium leading-relaxed">
          {answered === 0
            ? "1問も解かずに終えました。"
            : missed === 0
              ? "全問正解です。"
              : `間違えた${missed}問は、また出題します。`}
          {answered < total && `（${total - answered}問は未回答）`}
        </p>
      </section>

      <div className="mt-3 space-y-1">
        {again.isError && <Notice>{errorMessage(again.error)}</Notice>}
        <ButtonLink href="/app" className="w-full">
          {daily ? "今日の日課に戻る" : "ホームに戻る"}
        </ButtonLink>
        {!daily && (
          <Button variant="secondary" className="w-full" onClick={() => again.mutate()} disabled={again.isPending}>
            {again.isPending ? "用意しています…" : `続けて${againCount}${passage ? "題" : "問"}を解く`}
          </Button>
        )}
      </div>

      <h2 className="mt-8 mb-2 font-display text-2xl">ふりかえり</h2>
      <ol className="space-y-1">
        {session.items.map((it, i) => {
          const fb = it.feedback;
          const panelId = `review-${it.position}`;
          const startsPassage = isPassage(it.format) && it.group !== session.items[i - 1]?.group;
          return (
            <li key={it.position} className="border-3 border-ink bg-surface">
              {startsPassage && (
                <details className="border-b-3 border-ink px-3 py-2">
                  <summary className="cursor-pointer">
                    <span lang="en" className="font-en font-bold">
                      {it.title}
                    </span>
                    <span className="ml-2 text-sm text-muted">本文を見る</span>
                  </summary>
                  <PassageText format={it.format} text={it.passage ?? ""} items={groupItems(session, it.group)} />
                </details>
              )}
              <button
                onClick={() => setOpen(open === it.position ? null : it.position)}
                disabled={!fb}
                aria-expanded={fb ? open === it.position : undefined}
                aria-controls={fb ? panelId : undefined}
                className="flex w-full items-start gap-3 p-3 text-left"
              >
                {fb ? (
                  <Mark correct={fb.correct} className="w-6 shrink-0 text-2xl" />
                ) : (
                  <span aria-hidden="true" className="w-6 shrink-0 text-center text-2xl text-muted">
                    －
                  </span>
                )}
                <span className="sr-only">{!fb ? "未回答" : fb.correct ? "正解" : "不正解"}</span>
                <span lang={it.format === "vocab_word" ? "ja" : "en"} className="font-en leading-relaxed">
                  {reviewLine(it)}
                </span>
              </button>
              {open === it.position && fb && (
                <div id={panelId} className="space-y-2 px-3 pb-3">
                  {!fb.correct && (
                    <p className="text-sm">
                      あなたの回答：
                      <span lang="en" className="font-en text-ng line-through">
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
    </Page>
  );
}
