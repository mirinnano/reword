"use client";

import { Button, Page } from "@/components/ui";

/** A crash inside a page. The learner's answers are saved one by one, so nothing is lost. */
export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <Page narrow>
      <section className="border-3 border-ink p-5">
        <h1 className="font-display text-2xl leading-snug">画面を表示できませんでした</h1>
        <p className="mt-2 leading-relaxed">解いた問題の結果は保存されています。もう一度読み込んでください。</p>
      </section>
      <Button className="mt-1 w-full" onClick={reset}>
        読み込み直す
      </Button>
    </Page>
  );
}
