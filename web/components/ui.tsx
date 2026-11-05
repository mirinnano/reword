// Shared UI pieces built on the tokens in app/globals.css.
// Mobile-first: full-width controls and 44px touch targets. Surfaces are
// panels with heavy ink frames, square corners and no shadows.

import Link from "next/link";
import { useId } from "react";

export function Page({
  children,
  narrow = false,
  wide = false,
}: {
  children: React.ReactNode;
  narrow?: boolean;
  wide?: boolean;
}) {
  const width = wide ? "max-w-5xl" : narrow ? "max-w-md" : "max-w-2xl";
  return (
    <main id="main" className={`mx-auto w-full flex-1 px-4 py-6 ${width}`}>
      {children}
    </main>
  );
}

/** A panel: the one container treatment. */
export function Sheet({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <section className={`border-3 border-ink bg-surface p-5 ${className}`}>{children}</section>;
}

const controlClass = "block h-11 w-full border-2 border-ink bg-surface px-3 text-base text-ink";

export function Field({
  label,
  hint,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string }) {
  const hintId = useId();
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      <input {...props} aria-describedby={hint ? hintId : undefined} className={controlClass} />
      {hint && (
        <span id={hintId} className="mt-1 block text-sm text-muted">
          {hint}
        </span>
      )}
    </label>
  );
}

export function Select({
  label,
  children,
  ...props
}: React.SelectHTMLAttributes<HTMLSelectElement> & { label: string }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      <select {...props} className={controlClass}>
        {children}
      </select>
    </label>
  );
}

type Variant = "primary" | "secondary" | "onSpot";

const buttonClass = (variant: Variant) =>
  `press inline-flex min-h-12 items-center justify-center border-2 border-ink px-5 text-base font-bold disabled:cursor-not-allowed disabled:opacity-50 ${
    {
      primary: "bg-ink text-ground",
      secondary: "bg-surface text-ink hover:bg-tint",
      // On spot yellow, which keeps black ink in both themes.
      onSpot: "border-on-spot bg-on-spot text-spot",
    }[variant]
  }`;

export function Button({
  variant = "primary",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return <button {...props} className={`${buttonClass(variant)} ${className}`} />;
}

/** Same look as Button, for navigation. */
export function ButtonLink({
  href,
  variant = "primary",
  className = "",
  children,
}: {
  href: string;
  variant?: Variant;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <Link href={href} className={`${buttonClass(variant)} ${className}`}>
      {children}
    </Link>
  );
}

export function Notice({ kind = "error", children }: { kind?: "error" | "info" | "success"; children: React.ReactNode }) {
  const styles = {
    error: "border-2 border-ng bg-surface text-ink",
    info: "bg-tint text-ink",
    success: "border-2 border-ink bg-spot text-on-spot",
  }[kind];
  return (
    <p role={kind === "error" ? "alert" : "status"} className={`px-3 py-2.5 text-sm leading-relaxed ${styles}`}>
      {children}
    </p>
  );
}

export function TextLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="font-medium underline decoration-2 underline-offset-4 hover:no-underline">
      {children}
    </Link>
  );
}

/** Page-level loading: one centered treatment everywhere. */
export function Loading({ label = "読み込み中…" }: { label?: string }) {
  return (
    <p className="py-16 text-center text-sm text-muted" role="status">
      {label}
    </p>
  );
}

/** The grading mark: a circle for correct, a cross for wrong. The shape, not the color, carries the grade. */
export function Mark({ correct, className = "" }: { correct: boolean; className?: string }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="3"
      className={`shrink-0 ${correct ? "text-ink" : "text-ng"} ${className}`}
    >
      {correct ? <circle cx="12" cy="12" r="9" /> : <path d="M5 5 19 19M19 5 5 19" />}
    </svg>
  );
}

/** Open/closed sign for a disclosure row, drawn to match Mark. */
export function PlusMinus({ open }: { open: boolean }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" className="size-5 shrink-0">
      <path d={open ? "M4 12h16" : "M4 12h16M12 4v16"} />
    </svg>
  );
}

const BURST = (() => {
  const points: string[] = [];
  const spikes = 14;
  for (let i = 0; i < spikes * 2; i++) {
    const r = i % 2 === 0 ? 50 : 38;
    const a = (Math.PI * i) / spikes;
    points.push(`${(60 + r * 1.15 * Math.cos(a)).toFixed(1)},${(50 + r * 0.9 * Math.sin(a)).toFixed(1)}`);
  }
  return points.join(" ");
})();

/** The sound effect of a correct answer: the word in a spiked burst. The one loud moment on a question. */
export function Burst({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <span className={`stamp relative inline-grid h-24 w-32 place-items-center text-on-spot ${className}`}>
      <svg aria-hidden="true" viewBox="0 0 120 100" className="absolute inset-0 size-full">
        <polygon points={BURST} fill="var(--spot)" stroke="var(--ink)" strokeWidth="3" strokeLinejoin="miter" />
      </svg>
      <span className="relative font-display text-2xl">{children}</span>
    </span>
  );
}
