import type { Metadata, Viewport } from "next";
import { Atkinson_Hyperlegible, Dela_Gothic_One, Zen_Kaku_Gothic_New } from "next/font/google";
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

// Headings and figures: the heavy gothic of a manga title.
const display = Dela_Gothic_One({
  variable: "--font-title",
  weight: "400",
  subsets: ["latin"],
  preload: false,
  display: "swap",
});

// English text: letterforms built to be told apart, for reading long passages.
const en = Atkinson_Hyperlegible({
  variable: "--font-latin",
  weight: ["400", "700"],
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "ReWord（仮）", template: "%s | ReWord（仮）" },
  description: "英検2級の語彙と長文を、毎日の日課で解いて覚える学習サービス",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#121212" },
  ],
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ja" className={`${ui.variable} ${display.variable} ${en.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
