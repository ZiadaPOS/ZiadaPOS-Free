import { NextRequest, NextResponse } from 'next/server';

export function proxy(request: NextRequest) {
  if (request.nextUrl.pathname.startsWith('/dashboard') && !request.cookies.has('pos_access_token') && !request.cookies.has('pos_refresh_token')) {
    return NextResponse.redirect(new URL('/auth/sign-in', request.url));
  }
  if (request.nextUrl.pathname === '/auth/sign-in' && (request.cookies.has('pos_access_token') || request.cookies.has('pos_refresh_token'))) {
    return NextResponse.redirect(new URL('/dashboard/overview', request.url));
  }
  return NextResponse.next();
}

export const config = { matcher: ['/dashboard/:path*', '/auth/sign-in'] };
