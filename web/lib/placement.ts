/** What each stage means: its index is the number of placement blocks passed (api/internal/practice/placement.go). */
export const STAGES = [
  {
    title: "準2級の語彙から",
    body: "日課の新しい問題は、準2級の語彙から出します。2級の長文は、第1段に通るまで日課に入れません。",
  },
  { title: "2級の基本語彙から", body: "日課の新しい問題は、2級の基本的な語彙から先に出します。" },
  { title: "2級の全体から", body: "日課の新しい問題は、2級全体から出します。" },
  { title: "2級の難しい語彙から", body: "日課の新しい問題は、難しい語彙から先に出します。" },
] as const;

export const BLOCKS = ["準2級", "2級 基本", "2級 応用"] as const;
