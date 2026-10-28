// Shared UI pieces built on the tokens in app/globals.css.
// Mobile-first: full-width controls and 44px touch targets. Surfaces are
// separated by a tone step (sheet on paper), not by shadows.

import Link from "next/link";
import { useId } from "react";

export function Page({ children, narrow = false }: { children: React.ReactNode; narrow?: boolean }) {
  return (
    <main id="main" className={`mx-auto w-full flex-1 px-4 py-8 ${narrow ? "max-w-md" : "max-w-2xl"}`}>
      {children}
    </main>
  );
}

/** A sheet of paper on the ground: the one container treatment. */
export function Sheet({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <section className={`rounded-[var(--radius-sheet)] bg-surface p-5 ${className}`}>{children}</section>;
}

export function Field({
  label,
  hint,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string }) {
  const hintId = useId();
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium">{label}</span>
      <input
        {...props}
        aria-describedby={hint ? hintId : undefined}
        className="block h-11 w-full rounded-[var(--radius-control)] border border-line bg-surface px-3 text-base text-ink focus-visible:border-accent"
      />
      {hint && (
        <span id={hintId} className="mt-1 block text-sm text-muted">
          {hint}
        </span>
      )}
    </label>
  );
}

export function Button({
  variant = "primary",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "secondary" }) {
  const styles = {
    primary: "bg-accent text-on-accent hover:opacity-90",
    secondary: "border border-line bg-surface text-ink hover:bg-ground",
  }[variant];
  return (
    <button
      {...props}
      className={`inline-flex min-h-11 items-center justify-center rounded-[var(--radius-control)] px-4 text-base font-medium transition-opacity disabled:cursor-not-allowed disabled:opacity-50 ${styles} ${className}`}
    />
  );
}

/** Same look as Button, for navigation. */
export function ButtonLink({
  href,
  variant = "primary",
  children,
}: {
  href: string;
  variant?: "primary" | "secondary";
  children: React.ReactNode;
}) {
  const styles = {
    primary: "bg-accent text-on-accent hover:opacity-90",
    secondary: "border border-line bg-surface text-ink hover:bg-ground",
  }[variant];
  return (
    <Link
      href={href}
      className={`inline-flex min-h-11 items-center justify-center rounded-[var(--radius-control)] px-5 text-base font-medium ${styles}`}
    >
      {children}
    </Link>
  );
}

export function Notice({ kind = "error", children }: { kind?: "error" | "info" | "success"; children: React.ReactNode }) {
  const styles = {
    error: "bg-ng-soft text-ink",
    info: "bg-ground text-ink",
    success: "bg-ok-soft text-ink",
  }[kind];
  return (
    <p
      role={kind === "error" ? "alert" : "status"}
      className={`rounded-[var(--radius-control)] px-3 py-2.5 text-sm leading-relaxed ${styles}`}
    >
      {children}
    </p>
  );
}

export function TextLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="text-accent underline underline-offset-4 hover:no-underline">
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

/** The red-pen mark used for grading: ○ correct, × wrong. */
export function Mark({ correct, className = "" }: { correct: boolean; className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={`inline-grid place-items-center font-bold leading-none ${correct ? "text-ok" : "text-ng"} ${className}`}
    >
      {correct ? "○" : "×"}
    </span>
  );
}
