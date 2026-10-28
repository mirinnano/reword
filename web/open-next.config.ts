import { defineCloudflareConfig } from "@opennextjs/cloudflare";

// No incremental cache yet: every page is static or fully dynamic.
// Add the R2 incremental cache when ISR pages appear.
export default defineCloudflareConfig({});
