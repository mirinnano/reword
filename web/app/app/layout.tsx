"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useId, useRef, useState } from "react";
import { Button, Field, Loading, Notice, Page } from "@/components/ui";
import { api, errorMessage, type Me } from "@/lib/api";
import { meKey, useMe } from "@/lib/auth";

/** Everything under /app requires a session; signed-out visitors go to /login. */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const me = useMe();

  useEffect(() => {
    if (me.data === null) router.replace(`/login?next=${encodeURIComponent(pathname)}`);
  }, [me.data, pathname, router]);

  if (me.isError) {
    return (
      <Page narrow>
        <Notice>{errorMessage(me.error)}</Notice>
      </Page>
    );
  }
  if (!me.data) return <Loading />;
  return (
    <>
      <Header me={me.data} />
      {me.data.active_profile_id ? children : <PickProfile me={me.data} />}
    </>
  );
}

function Header({ me }: { me: Me }) {
  const qc = useQueryClient();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const panelId = useId();
  const trigger = useRef<HTMLButtonElement>(null);
  const panel = useRef<HTMLDivElement>(null);
  const active = me.profiles.find((p) => p.id === me.active_profile_id);

  const close = () => {
    setOpen(false);
    trigger.current?.focus();
  };

  // Opening moves focus into the panel; Escape closes and returns it.
  useEffect(() => {
    if (!open) return;
    panel.current?.querySelector<HTMLElement>("button")?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        trigger.current?.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open]);

  const select = useMutation({
    mutationFn: (id: number) => api.selectProfile(id),
    onSuccess: async () => {
      close();
      // Practice data belongs to the profile: drop everything cached for the old one.
      await qc.invalidateQueries();
    },
  });
  const logout = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      qc.clear();
      router.replace("/login");
    },
  });

  return (
    <header className="sticky top-0 z-10 border-b border-hairline bg-ground">
      <div className="mx-auto flex h-14 max-w-2xl items-center justify-between px-4">
        <Link href="/app" className="text-lg font-bold tracking-tight">
          ReWord
        </Link>
        <div className="relative">
          <button
            ref={trigger}
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-controls={panelId}
            className="flex min-h-11 items-center gap-2 rounded-full border border-line bg-surface pr-3 pl-1.5 text-sm"
          >
            <span aria-hidden="true" className="grid size-7 place-items-center rounded-full bg-accent text-sm font-bold text-on-accent">
              {active?.display_name.slice(0, 1) ?? "?"}
            </span>
            <span className="max-w-32 truncate">{active?.display_name ?? "未選択"}</span>
            <span className="sr-only">（学習者の切り替えとログアウト）</span>
          </button>
          {open && (
            // Backdrop: tapping outside the panel closes it.
            <div aria-hidden="true" className="fixed inset-0" onClick={close} />
          )}
          {open && (
            <div
              ref={panel}
              id={panelId}
              className="absolute right-0 mt-2 w-72 rounded-[var(--radius-sheet)] border border-hairline bg-surface p-2 shadow-lg"
            >
              <p className="truncate px-2 py-1.5 text-sm text-muted">{me.user.email}</p>
              <p className="px-2 pt-1 text-sm font-medium">学習者</p>
              <ul>
                {me.profiles.map((p) => (
                  <li key={p.id}>
                    <button
                      onClick={() => select.mutate(p.id)}
                      aria-current={p.id === me.active_profile_id ? "true" : undefined}
                      className="flex min-h-11 w-full items-center rounded-[var(--radius-control)] px-2 text-left hover:bg-ground"
                    >
                      <span className="min-w-0 flex-1 truncate">{p.display_name}</span>
                      {p.id === me.active_profile_id && <span className="text-sm text-accent">選択中</span>}
                    </button>
                  </li>
                ))}
              </ul>
              <AddProfile onDone={close} full={me.profiles.length >= 5} />
              <hr className="my-2 border-hairline" />
              <button
                onClick={() => logout.mutate()}
                className="flex min-h-11 w-full items-center rounded-[var(--radius-control)] px-2 text-left text-ng hover:bg-ground"
              >
                ログアウト
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}

function AddProfile({ onDone, full }: { onDone: () => void; full: boolean }) {
  const qc = useQueryClient();
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const create = useMutation({
    mutationFn: () => api.createProfile({ display_name: name, target_grade_id: null }),
    onSuccess: async (p) => {
      await api.selectProfile(p.id);
      await qc.invalidateQueries();
      setName("");
      setAdding(false);
      onDone();
    },
  });
  if (full) return <p className="px-2 py-2 text-sm text-muted">学習者は5人まで登録できます</p>;
  if (!adding) {
    return (
      <button
        onClick={() => setAdding(true)}
        className="flex min-h-11 w-full items-center rounded-[var(--radius-control)] px-2 text-left text-accent hover:bg-ground"
      >
        学習者を追加する
      </button>
    );
  }
  return (
    <form
      className="space-y-2 p-2"
      onSubmit={(e) => {
        e.preventDefault();
        create.mutate();
      }}
    >
      <Field label="表示名" name="display_name" autoComplete="off" required maxLength={40} value={name} onChange={(e) => setName(e.target.value)} />
      {create.isError && <Notice>{errorMessage(create.error)}</Notice>}
      <Button type="submit" className="w-full" disabled={create.isPending}>
        追加して切り替える
      </Button>
    </form>
  );
}

function PickProfile({ me }: { me: Me }) {
  const qc = useQueryClient();
  const select = useMutation({
    mutationFn: (id: number) => api.selectProfile(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: meKey }),
  });
  return (
    <Page narrow>
      <h1 className="mb-4 text-xl font-bold">だれが学習しますか？</h1>
      <div className="space-y-2">
        {me.profiles.map((p) => (
          <Button key={p.id} variant="secondary" className="w-full" onClick={() => select.mutate(p.id)}>
            {p.display_name}
          </Button>
        ))}
      </div>
    </Page>
  );
}
