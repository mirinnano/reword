"use client";

import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { Notice, Button, Sheet, Field, Loading, TextLink } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { useHashToken } from "@/lib/auth";

export default function ResetPasswordPage() {
  const token = useHashToken();
  const [password, setPassword] = useState("");
  const reset = useMutation({ mutationFn: () => api.confirmReset(token!, password) });

  if (token === undefined) return <Loading />;
  if (token === null) {
    return (
      <Sheet>
        <Notice>リンクが正しくありません。メールのリンクをそのまま開いてください。</Notice>
        <p className="mt-4 text-center text-sm">
          <TextLink href="/forgot-password">再設定をやり直す</TextLink>
        </p>
      </Sheet>
    );
  }
  return (
    <Sheet>
      <h1 className="mb-4 text-lg font-bold">新しいパスワード</h1>
      {reset.isSuccess ? (
        <>
          <Notice kind="success">パスワードを変更しました。安全のため、すべての端末でログアウトしています。</Notice>
          <p className="mt-4 text-center text-sm">
            <TextLink href="/login">ログインする</TextLink>
          </p>
        </>
      ) : (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            reset.mutate();
          }}
        >
          <Field
            label="新しいパスワード"
            type="password"
            autoComplete="new-password"
            required
            minLength={10}
            maxLength={128}
            hint="10文字以上"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          {reset.isError && <Notice>{errorMessage(reset.error)}</Notice>}
          <Button type="submit" className="w-full" disabled={reset.isPending}>
            変更する
          </Button>
        </form>
      )}
    </Sheet>
  );
}
