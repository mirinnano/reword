"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Button, Loading, Notice, Page, Sheet } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { useTitle } from "@/lib/title";

/** One rule, three examples, then the check. Grammar is read quietly, like a passage. */
export default function GrammarLessonPage() {
  const key = useParams<{ key: string }>().key;
  const router = useRouter();
  const lesson = useQuery({ queryKey: ["grammar-unit", key], queryFn: () => api.grammarUnit(key) });
  const start = useMutation({
    mutationFn: () => api.startGrammar(key),
    onSuccess: (s) => router.push(`/app/practice/${s.id}`),
  });
  useTitle(lesson.data?.title ?? "文法");

  const back = (
    <Link href="/app/grammar" className="font-medium underline decoration-2 underline-offset-4">
      単元の一覧に戻る
    </Link>
  );
  if (lesson.isPending) return <Loading />;
  if (lesson.isError) {
    return (
      <Page narrow>
        <h1 className="mb-3 font-display text-2xl">単元を開けませんでした</h1>
        <Notice>{errorMessage(lesson.error)}</Notice>
        <p className="mt-4 text-center">{back}</p>
      </Page>
    );
  }
  const l = lesson.data;
  return (
    <Page>
      <p className="text-sm font-bold">{l.group}</p>
      <h1 className="font-display text-3xl leading-snug">{l.title}</h1>
      <Sheet className="mt-4">
        <div className="space-y-3 leading-relaxed">
          {l.rule_ja
            .trim()
            .split(/\n\s*\n/)
            .map((para) => (
              <p key={para}>{para}</p>
            ))}
        </div>
      </Sheet>
      <h2 className="mt-6 font-display text-2xl">例文</h2>
      <ul className="mt-2 space-y-3">
        {l.examples.map((e) => (
          <li key={e.en} className="border-l-3 border-ink pl-3">
            <p lang="en" className="font-en text-lg">
              {e.en}
            </p>
            <p className="mt-1 text-sm leading-relaxed">{e.ja}</p>
            {e.note && <p className="mt-1 text-sm leading-relaxed text-muted">{e.note}</p>}
          </li>
        ))}
      </ul>
      {start.isError && (
        <div className="mt-4">
          <Notice>{errorMessage(start.error)}</Notice>
        </div>
      )}
      <Button className="mt-6 min-h-14 w-full" disabled={start.isPending} onClick={() => start.mutate()}>
        {start.isPending ? "用意しています…" : "確認問題を6問解く"}
      </Button>
      <p className="mt-4 text-center">{back}</p>
    </Page>
  );
}
