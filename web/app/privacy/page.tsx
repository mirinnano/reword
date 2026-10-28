import { Sheet, Page, TextLink } from "@/components/ui";

export const metadata = { title: "プライバシーポリシー（準備中） | ReWord（仮）" };

export default function PrivacyPage() {
  return (
    <Page narrow>
      <Sheet>
        <h1 className="mb-3 text-lg font-bold">プライバシーポリシー</h1>
        <p className="text-sm leading-relaxed text-muted">
          正式公開に向けて準備中です（版: 2026-10-draft）。開発中の検証環境では、ログインと学習記録の保存のためにメールアドレスと学習履歴を保存します。
        </p>
        <p className="mt-4 text-sm">
          <TextLink href="/">トップへ</TextLink>
        </p>
      </Sheet>
    </Page>
  );
}
