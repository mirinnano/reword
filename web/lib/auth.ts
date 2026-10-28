"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, api, type Me } from "./api";

export const meKey = ["me"] as const;

/** The signed-in user, or null when signed out. */
export function useMe() {
  return useQuery({
    queryKey: meKey,
    queryFn: async (): Promise<Me | null> => {
      try {
        return await api.me();
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
  });
}

export function useSetMe() {
  const qc = useQueryClient();
  return (me: Me | null) => qc.setQueryData(meKey, me);
}

/**
 * Only same-site paths are accepted as a post-login destination, so a crafted
 * ?next= cannot send users to another site (open redirect).
 */
export function safeNext(next: string | null): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.startsWith("/\\")) return "/app";
  return next;
}

/**
 * Reads #token=... once, then strips it from the address bar. Tokens travel in
 * the fragment so they never reach servers, logs or Referer headers.
 */
export function useHashToken(): string | null | undefined {
  const [token, setToken] = useState<string | null | undefined>(undefined);
  const read = useRef(false);
  useEffect(() => {
    if (read.current) return;
    read.current = true;
    const t = new URLSearchParams(window.location.hash.slice(1)).get("token");
    if (t) window.history.replaceState(null, "", window.location.pathname);
    setToken(t);
  }, []);
  return token;
}
