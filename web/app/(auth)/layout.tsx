import Link from "next/link";
import { Page } from "@/components/ui";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <Page narrow>
      <Link href="/" className="mb-6 block font-display text-3xl">
        ReWord<span className="font-sans text-sm text-muted">（仮）</span>
      </Link>
      {children}
    </Page>
  );
}
