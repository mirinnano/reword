import type { NextConfig } from "next";
import { initOpenNextCloudflareForDev } from "@opennextjs/cloudflare";

// /api/* is proxied to the Go API by app/api/[...path]/route.ts in every
// environment, so the browser always sees a single origin.
//
// cacheComponents is off: on Cloudflare Workers it hangs requests (the
// runtime's setTimeout semantics differ). Revisit when OpenNext supports it.
const nextConfig: NextConfig = {
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;

initOpenNextCloudflareForDev();
