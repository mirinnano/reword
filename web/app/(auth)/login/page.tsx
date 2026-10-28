"use client";

import { useMutation } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { Notice, Button, Sheet, Field, TextLink } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { safeNext, useSetMe } from "@/lib/auth";

function LoginForm() {
  const router = useRouter();
  const next = safeNext(useSearchParams().get("next"));
  const setMe = useSetMe();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useMutation({
    mutationFn: () => api.login(email, password),
    onSuccess: (me) => {
      setMe(me);
      router.replace(next);
    },
  });

  return (
    <Sheet>
      <h1 className="mb-4 text-lg font-bold">ログイン</h1>
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          login.mutate();
        }}
      >
        <Field label="メールアドレス" type="email" name="email" autoComplete="email" spellCheck={false} autoCapitalize="off" required value={email} onChange={(e) => setEmail(e.target.value)} />
        <Field label="パスワード" type="password" name="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
        {login.isError && <Notice>{errorMessage(login.error)}</Notice>}
        <Button type="submit" className="w-full" disabled={login.isPending}>
          {login.isPending ? "ログイン中…" : "ログイン"}
        </Button>
      </form>
      <div className="mt-5 space-y-2 text-center text-sm">
        <p>
          <TextLink href="/forgot-password">パスワードを忘れた場合</TextLink>
        </p>
        <p>
          アカウントがない場合は <TextLink href="/signup">新規登録</TextLink>
        </p>
      </div>
    </Sheet>
  );
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginForm />
    </Suspense>
  );
}
