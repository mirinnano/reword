import Link from "next/link";
import { Page } from "@/components/ui";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <Page narrow>
      <Link href="/" className="mb-8 block text-center text-xl font-bold">
        ReWord<span className="text-sm font-normal text-muted">（仮）</span>
      </Link>
      {children}
    </Page>
  );
}
