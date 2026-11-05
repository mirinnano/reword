"use client";

import { useMutation } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";
import { Notice, Sheet, Loading, TextLink } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { useTitle } from "@/lib/title";
import { useHashToken, useSetMe } from "@/lib/auth";

export default function VerifyEmailPage() {
  useTitle("登録の確認");
  const router = useRouter();
  const token = useHashToken();
  const setMe = useSetMe();
  const verify = useMutation({
    mutationFn: (t: string) => api.verifySignup(t),
    onSuccess: (me) => {
      setMe(me);
      router.replace("/app");
    },
  });
  const { mutate } = verify;
  // The token is single-use: send it once even if the effect runs twice.
  const sent = useRef(false);

  useEffect(() => {
    if (token && !sent.current) {
      sent.current = true;
      mutate(token);
    }
  }, [token, mutate]);

  if (token === null || verify.isError) {
    return (
      <Sheet>
        <h1 className="mb-3 font-display text-2xl">登録を完了できませんでした</h1>
        <Notice>{token === null ? "リンクが正しくありません。メールのリンクをそのまま開いてください。" : errorMessage(verify.error)}</Notice>
        <p className="mt-4 text-center text-sm">
          <TextLink href="/signup">もう一度登録する</TextLink>
        </p>
      </Sheet>
    );
  }
  return <Loading label="登録を完了しています…" />;
}
