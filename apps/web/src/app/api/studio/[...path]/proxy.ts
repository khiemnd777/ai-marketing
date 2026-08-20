import { NextResponse } from "next/server";

export type RouteContext = { params: Promise<{ path: string[] }> };
type StreamingRequestInit = RequestInit & { duplex?: "half" };

const requestHopByHopHeaders = new Set([
  "connection",
  "content-length",
  "host",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
]);

const responseHopByHopHeaders = new Set([
  "connection",
  "content-encoding",
  "content-length",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
]);

function problem(status: number, title: string, detail: string) {
  return NextResponse.json(
    { type: "about:blank", title, status, detail },
    { status, headers: { "cache-control": "no-store", "content-type": "application/problem+json" } },
  );
}

function upstreamUrl(request: Request, path: string[]) {
  const configuredBase = process.env.API_URL ?? "http://127.0.0.1:8080";
  const base = new URL(configuredBase);
  if (!["http:", "https:"].includes(base.protocol)) throw new Error("API_URL must use http or https");
  const target = new URL(base);
  target.pathname = `${base.pathname.replace(/\/$/, "")}/v1/${path.map(encodeURIComponent).join("/")}`;
  target.search = new URL(request.url).search;
  return target;
}

function copyRequestHeaders(request: Request) {
  const headers = new Headers();
  request.headers.forEach((value, name) => {
    if (!requestHopByHopHeaders.has(name.toLowerCase())) headers.append(name, value);
  });
  return headers;
}

function copyResponseHeaders(response: Response) {
  const headers = new Headers();
  response.headers.forEach((value, name) => {
    const normalized = name.toLowerCase();
    if (normalized !== "set-cookie" && !responseHopByHopHeaders.has(normalized)) headers.append(name, value);
  });
  for (const cookie of response.headers.getSetCookie()) headers.append("set-cookie", cookie);
  headers.set("cache-control", "no-store");
  return headers;
}

export async function proxyToStudioApi(request: Request, context: RouteContext) {
  let target: URL;
  try {
    const { path } = await context.params;
    target = upstreamUrl(request, path);
  } catch {
    return problem(500, "API proxy configuration error", "The application API endpoint is not configured correctly.");
  }

  const hasBody = !["GET", "HEAD"].includes(request.method.toUpperCase());
  const init: StreamingRequestInit = {
    method: request.method,
    headers: copyRequestHeaders(request),
    body: hasBody ? request.body : undefined,
    cache: "no-store",
    redirect: "manual",
    signal: request.signal,
  };
  if (hasBody && request.body) init.duplex = "half";

  try {
    const response = await fetch(target, init);
    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: copyResponseHeaders(response),
    });
  } catch {
    return problem(502, "API unavailable", "The application API could not be reached.");
  }
}
