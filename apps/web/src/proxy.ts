import { NextResponse, type NextRequest } from "next/server";
import { contentSecurityPolicy } from "./lib/content-security-policy";

function secured(response: NextResponse) {
  response.headers.set("Content-Security-Policy", contentSecurityPolicy(process.env.BROWSER_STORAGE_ORIGINS));
  return response;
}

export function proxy(request: NextRequest) {
  const returnUrl = `${request.nextUrl.pathname}${request.nextUrl.search}`;
  if (request.nextUrl.pathname !== "/login" && !request.cookies.has("studio_session")) {
    const login = new URL("/login", request.url);
    login.searchParams.set("returnUrl", returnUrl);
    return secured(NextResponse.redirect(login));
  }

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-studio-return-url", returnUrl);
  return secured(NextResponse.next({ request: { headers: requestHeaders } }));
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|robots.txt).*)"],
};
