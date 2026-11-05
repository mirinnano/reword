import { useEffect } from "react";

/**
 * Names the screen. Most routes are client components, which cannot export
 * metadata, and the title is what announces a route change to screen readers.
 */
export function useTitle(title: string | null | undefined) {
  useEffect(() => {
    if (title) document.title = `${title} | ReWord（仮）`;
  }, [title]);
}
