"use client";

import { ButtonLink } from "@/components/ui";
import { useMe } from "@/lib/auth";

export default function Home() {
  const me = useMe();
  return (
    <main id="main" className="mx-auto flex w-full max-w-xl flex-1 flex-col justify-center gap-8 px-4 py-16">
      <div>
        <h1 className="text-4xl font-bold tracking-tight">ReWord</h1>
        <p className="mt-4 text-lg leading-relaxed">
          英検の語彙を、解いて、間違えて、覚える。2級の語彙・熟語問題150問を公開中です。
        </p>
      </div>
      <div className="flex flex-col gap-3 sm:flex-row">
        {me.data ? (
          <ButtonLink href="/app">今日の演習へ</ButtonLink>
        ) : (
          <>
            <ButtonLink href="/signup">無料で登録する</ButtonLink>
            <ButtonLink href="/login" variant="secondary">
              ログイン
            </ButtonLink>
          </>
        )}
      </div>
      <p className="text-sm text-muted">ReWord は仮称です。開発中の検証環境のため、データは予告なく削除されることがあります。</p>
    </main>
  );
}
