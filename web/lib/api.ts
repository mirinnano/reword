// Typed client for the Go API (same origin, proxied by app/api/[...path]).
// Types mirror the Go handlers until the OpenAPI contract lands.

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api${path}`, {
    method,
    credentials: "same-origin",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => null);
  if (!res.ok) {
    throw new ApiError(res.status, data?.error?.code ?? "unknown", data?.error?.message ?? res.statusText);
  }
  return data as T;
}

// ---- types

export type Grade = { id: number; code: string; name_ja: string };
export type Profile = { id: number; display_name: string; target_grade_id: number | null };
export type Me = {
  user: { id: number; email: string; role: string };
  profiles: Profile[];
  active_profile_id: number | null;
};

/** mcq: sentence with a blank; vocab_meaning: headword → meaning; vocab_word: meaning → headword. */
export type Format = "mcq" | "vocab_meaning" | "vocab_word";

export type PracticeType = {
  code: string;
  name_ja: string;
  format: Format;
  grade_code: string;
  grade_name_ja: string;
  question_count: number;
};
export type Vocab = {
  key: string;
  headword: string;
  pos: string;
  meanings_ja: string[];
  example_en: string;
  example_ja: string;
};
export type Feedback = {
  selected: number;
  correct: boolean;
  correct_choice: number;
  explanation_ja: string;
  vocab: Vocab[];
};
export type Item = { position: number; stem: string; choices: string[]; feedback: Feedback | null };
export type Progress = { total: number; answered: number; correct: number; finished: boolean };
export type Session = {
  id: number;
  type_code: string;
  type_name_ja: string;
  format: Format;
  started_at: string;
  progress: Progress;
  items: Item[];
};
/** Per question type: seen, mastered (3 spaced correct answers in a row) and due for review now. */
export type TypeProgress = { type_code: string; total: number; seen: number; mastered: number; due: number };

export type SessionSummary = Omit<Session, "items"> & { finished_at: string | null };

// ---- endpoints

export const api = {
  me: () => request<Me>("GET", "/me"),
  grades: () => request<{ grades: Grade[] }>("GET", "/grades"),

  signup: (b: { email: string; password: string; display_name: string; target_grade_id: number | null; accept_terms: boolean }) =>
    request<{ status: string }>("POST", "/auth/signup", b),
  verifySignup: (token: string) => request<Me>("POST", "/auth/signup/verify", { token }),
  login: (email: string, password: string) => request<Me>("POST", "/auth/login", { email, password }),
  logout: () => request<void>("POST", "/auth/logout", {}),
  requestReset: (email: string) => request<{ status: string }>("POST", "/auth/password-reset", { email }),
  confirmReset: (token: string, password: string) =>
    request<void>("POST", "/auth/password-reset/confirm", { token, password }),

  createProfile: (b: { display_name: string; target_grade_id: number | null }) =>
    request<Profile>("POST", "/profiles", b),
  selectProfile: (id: number) => request<Profile>("POST", `/profiles/${id}/select`, {}),

  practiceTypes: () => request<{ types: PracticeType[] }>("GET", "/practice/types"),
  progress: () => request<{ progress: TypeProgress[] }>("GET", "/practice/progress"),
  recentSessions: () => request<{ sessions: SessionSummary[] }>("GET", "/practice/sessions?limit=10"),
  startSession: (type: string, count: number) => request<Session>("POST", "/practice/sessions", { type, count }),
  session: (id: number) => request<Session>("GET", `/practice/sessions/${id}`),
  answer: (id: number, position: number, choice: number, time_ms: number) =>
    request<{ feedback: Feedback; progress: Progress }>("POST", `/practice/sessions/${id}/answers`, {
      position,
      choice,
      time_ms,
    }),
  finish: (id: number) => request<Session>("POST", `/practice/sessions/${id}/finish`, {}),
};

// Japanese messages for API error codes shown to users.
const messages: Record<string, string> = {
  invalid_credentials: "メールアドレスまたはパスワードが正しくありません。",
  rate_limited: "短い時間に何度も試したため、一時的に止めています。15分ほど待ってからお試しください。",
  invalid_email: "メールアドレスの形式が正しくありません。",
  weak_password: "パスワードは10〜128文字で入力してください。",
  invalid_display_name: "表示名は1〜40文字で入力してください。",
  terms_not_accepted: "利用規約とプライバシーポリシーへの同意が必要です。",
  invalid_token: "リンクが無効か、有効期限が切れています。もう一度お手続きください。",
  profile_limit: "プロフィールは5つまで作成できます。",
  profile_required: "学習する人を選んでから始めてください。",
  no_questions: "この形式の問題はまだ公開されていません。",
  already_answered: "この問題はすでに回答済みです。",
  session_finished: "この演習はすでに終了しています。",
};

export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) return messages[e.code] ?? "うまく処理できませんでした。時間をおいて、もう一度お試しください。";
  return "通信できませんでした。接続を確認して、もう一度お試しください。";
}
