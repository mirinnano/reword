import { ButtonLink, Page } from "@/components/ui";

export const metadata = { title: "ページが見つかりません" };

export default function NotFound() {
  return (
    <Page narrow>
      <section className="border-3 border-ink bg-spot p-5 text-on-spot">
        <h1 className="font-display text-3xl leading-snug">このページはありません</h1>
        <p className="mt-2 font-medium leading-relaxed">URL が間違っているか、ページが移動した可能性があります。</p>
      </section>
      <ButtonLink href="/app" className="mt-1 w-full">
        今日の日課へ
      </ButtonLink>
    </Page>
  );
}
