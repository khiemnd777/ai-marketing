import { NextResponse, type NextRequest } from "next/server";

export function proxy(request: NextRequest) {
  const returnUrl = `${request.nextUrl.pathname}${request.nextUrl.search}`;
  if (!request.cookies.has("studio_session")) {
    const login = new URL("/login", request.url);
    login.searchParams.set("returnUrl", returnUrl);
    return NextResponse.redirect(login);
  }

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-studio-return-url", returnUrl);
  return NextResponse.next({ request: { headers: requestHeaders } });
}

export const config = {
  matcher: ["/((?!api|login|_next/static|_next/image|favicon.ico|robots.txt).*)"],
};
