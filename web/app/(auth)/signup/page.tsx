"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Notice, Button, Select, Sheet, Field, TextLink } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { useTitle } from "@/lib/title";

export default function SignupPage() {
  useTitle("新規登録");
  const grades = useQuery({ queryKey: ["grades"], queryFn: api.grades });
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [grade, setGrade] = useState<string>("");
  const [accept, setAccept] = useState(false);
  const signup = useMutation({
    mutationFn: () =>
      api.signup({
        email,
        password,
        display_name: name,
        target_grade_id: grade ? Number(grade) : null,
        accept_terms: accept,
      }),
  });

  if (signup.isSuccess) {
    return (
      <Sheet>
        <h1 className="mb-3 font-display text-2xl">確認メールを送りました</h1>
        <p className="text-sm leading-relaxed text-muted">
          <strong className="text-ink">{email}</strong> 宛てのメールにあるリンクを開くと、登録が完了します。リンクの有効期限は24時間です。
        </p>
        <p className="mt-4 text-sm text-muted">メールが届かない場合は、迷惑メールフォルダも確認してください。</p>
      </Sheet>
    );
  }

  return (
    <Sheet>
      <h1 className="mb-4 font-display text-2xl">新規登録</h1>
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          signup.mutate();
        }}
      >
        <Field label="メールアドレス" type="email" name="email" autoComplete="email" spellCheck={false} autoCapitalize="off" required value={email} onChange={(e) => setEmail(e.target.value)} />
        <Field
          label="パスワード"
          type="password"
          autoComplete="new-password"
          required
          minLength={10}
          maxLength={128}
          hint="10文字以上"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <Field
          label="表示名"
          required
          maxLength={40}
          hint="学習者の名前です。ログイン後に学習者を追加できます。"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Select label="目標の級（任意）" value={grade} onChange={(e) => setGrade(e.target.value)}>
          <option value="">選択しない</option>
          {grades.data?.grades.map((g) => (
            <option key={g.id} value={g.id}>
              {g.name_ja}
            </option>
          ))}
        </Select>
        <label className="flex min-h-11 items-start gap-3 py-1 text-sm leading-relaxed">
          <input type="checkbox" name="accept_terms" className="mt-0.5 size-5 shrink-0 accent-[var(--ink)]" checked={accept} onChange={(e) => setAccept(e.target.checked)} required />
          <span>
            <TextLink href="/terms">利用規約</TextLink>と<TextLink href="/privacy">プライバシーポリシー</TextLink>に同意します
          </span>
        </label>
        {signup.isError && <Notice>{errorMessage(signup.error)}</Notice>}
        <Button type="submit" className="w-full" disabled={signup.isPending}>
          {signup.isPending ? "送信中…" : "確認メールを送る"}
        </Button>
      </form>
      <p className="mt-5 text-center text-sm">
        登録済みの場合は <TextLink href="/login">ログイン</TextLink>
      </p>
    </Sheet>
  );
}
