"use client";

import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { Notice, Button, Sheet, Field, TextLink } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const req = useMutation({ mutationFn: () => api.requestReset(email) });

  return (
    <Sheet>
      <h1 className="mb-4 text-lg font-bold">パスワードの再設定</h1>
      {req.isSuccess ? (
        <Notice kind="success">
          登録されているメールアドレスであれば、再設定用のリンクを送りました。リンクの有効期限は30分です。
        </Notice>
      ) : (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            req.mutate();
          }}
        >
          <Field label="メールアドレス" type="email" name="email" autoComplete="email" spellCheck={false} autoCapitalize="off" required value={email} onChange={(e) => setEmail(e.target.value)} />
          {req.isError && <Notice>{errorMessage(req.error)}</Notice>}
          <Button type="submit" className="w-full" disabled={req.isPending}>
            再設定用のリンクを送る
          </Button>
        </form>
      )}
      <p className="mt-5 text-center text-sm">
        <TextLink href="/login">ログインに戻る</TextLink>
      </p>
    </Sheet>
  );
}
