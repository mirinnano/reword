"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Loading, Notice, Page } from "@/components/ui";
import { api, errorMessage, type GrammarUnit } from "@/lib/api";
import { useTitle } from "@/lib/title";

/** Units in teaching order. Nothing is locked; the next unit is only a suggestion. */
export default function GrammarPage() {
  useTitle("文法");
  // Always fresh on arrival: this is where a learner returns to after a check.
  const units = useQuery({ queryKey: ["grammar-units"], queryFn: api.grammarUnits, refetchOnMount: "always" });
  if (units.isPending) return <Loading />;
  if (units.isError) {
    return (
      <Page>
        <h1 className="mb-3 font-display text-3xl">文法</h1>
        <Notice>{errorMessage(units.error)}</Notice>
      </Page>
    );
  }
  const { units: list, next, pass } = units.data;
  const groups = [...new Set(list.map((u) => u.group))];
  const done = list.filter((u) => u.done).length;
  return (
    <Page>
      <h1 className="font-display text-3xl">文法</h1>
      {list.length === 0 ? (
        <div className="mt-4">
          <Notice kind="info">公開中の単元はまだありません。</Notice>
        </div>
      ) : (
        <>
          <p className="mt-2 leading-relaxed">
            解説を読んで、確認問題を6問解きます。{pass}問以上の正解で修了です。修了 {done}／{list.length}単元
          </p>
          {groups.map((g) => (
            <section key={g} className="mt-6">
              <h2 className="mb-2 font-display text-2xl">{g}</h2>
              <ul className="border-3 border-ink">
                {list
                  .filter((u) => u.group === g)
                  .map((u, i) => (
                    <Row key={u.key} unit={u} first={i === 0} next={u.key === next} pass={pass} />
                  ))}
              </ul>
            </section>
          ))}
        </>
      )}
    </Page>
  );
}

function Row({ unit, first, next, pass }: { unit: GrammarUnit; first: boolean; next: boolean; pass: number }) {
  const status = unit.done ? `修了（${unit.best}／6問）` : unit.best !== null ? `あと${pass - unit.best}問で修了` : next ? "次の単元" : "";
  return (
    <li className={first ? "" : "border-t-2 border-hairline"}>
      <Link
        href={`/app/grammar/${unit.key}`}
        className={`flex min-h-14 items-center justify-between gap-3 px-3 py-2 hover:bg-tint ${next ? "bg-spot text-on-spot hover:bg-spot" : ""}`}
      >
        <span className="min-w-0 font-medium">{unit.title}</span>
        <span className="shrink-0 text-sm font-bold tabular-nums">{status}</span>
      </Link>
    </li>
  );
}
