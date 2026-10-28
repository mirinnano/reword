// Proxies /api/* to the Go API so the browser only ever talks to one origin
// (first-party cookies, no CORS). In staging/production the API sits behind
// Caddy, which rejects requests that lack the shared X-Origin-Auth secret.

const HOP_BY_HOP = [
  "connection",
  "keep-alive",
  "transfer-encoding",
  "upgrade",
  "proxy-authorization",
  "te",
  "trailer",
  "host",
];

async function proxy(req: Request): Promise<Response> {
  const origin = process.env.API_ORIGIN;
  if (!origin) {
    return Response.json({ error: "API_ORIGIN not configured" }, { status: 500 });
  }

  const src = new URL(req.url);
  const target = new URL(src.pathname + src.search, origin);

  const headers = new Headers(req.headers);
  for (const h of HOP_BY_HOP) headers.delete(h);
  headers.delete("x-origin-auth");
  if (process.env.ORIGIN_SECRET) headers.set("x-origin-auth", process.env.ORIGIN_SECRET);
  headers.set("x-forwarded-host", src.host);
  headers.set("x-forwarded-proto", src.protocol.replace(":", ""));
  const clientIp = req.headers.get("cf-connecting-ip");
  if (clientIp) headers.set("x-forwarded-for", clientIp);

  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: req.method,
      headers,
      body: hasBody ? req.body : undefined,
      redirect: "manual",
      // Required by Node's fetch when streaming a request body.
      ...(hasBody ? { duplex: "half" } : {}),
    } as RequestInit);
  } catch {
    return Response.json({ error: "upstream unavailable" }, { status: 502 });
  }

  const resHeaders = new Headers(upstream.headers);
  for (const h of HOP_BY_HOP) resHeaders.delete(h);
  return new Response(upstream.body, { status: upstream.status, headers: resHeaders });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
