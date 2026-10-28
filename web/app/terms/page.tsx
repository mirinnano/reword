import { Sheet, Page, TextLink } from "@/components/ui";

export const metadata = { title: "利用規約（準備中） | ReWord（仮）" };

export default function TermsPage() {
  return (
    <Page narrow>
      <Sheet>
        <h1 className="mb-3 text-lg font-bold">利用規約</h1>
        <p className="text-sm leading-relaxed text-muted">
          正式公開に向けて準備中です（版: 2026-10-draft）。現在は開発中の検証環境であり、登録内容やデータは予告なく削除されることがあります。
        </p>
        <p className="mt-4 text-sm">
          <TextLink href="/">トップへ</TextLink>
        </p>
      </Sheet>
    </Page>
  );
}
