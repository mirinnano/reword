import type { Metadata, Viewport } from "next";
import { Source_Serif_4, Zen_Kaku_Gothic_New } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";

// UI face: Japanese gothic. Japanese glyphs arrive in unicode-range chunks,
// so nothing is preloaded up front.
const ui = Zen_Kaku_Gothic_New({
  variable: "--font-ui",
  weight: ["400", "500", "700"],
  subsets: ["latin"],
  preload: false,
  display: "swap",
});

// English question text: a book serif, like a printed test sheet.
const serif = Source_Serif_4({
  variable: "--font-serif",
  weight: ["400", "600"],
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "ReWord（仮）",
  description: "英検 準2級〜1級 対策のデジタル学習サービス",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f3f4f1" },
    { media: "(prefers-color-scheme: dark)", color: "#101419" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ja" className={`${ui.variable} ${serif.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
