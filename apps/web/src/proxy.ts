import { NextRequest, NextResponse } from 'next/server';

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  if (/\.(?:avif|gif|ico|jpe?g|png|svg|webp|woff2?|ttf|otf|txt)$/i.test(pathname)) {
    return NextResponse.next();
  }
  if (pathname === '/dashboard' || pathname.startsWith('/dashboard/')) {
    const nextPath = pathname === '/dashboard' ? '/overview' : pathname.slice('/dashboard'.length);
    return NextResponse.redirect(new URL(`${nextPath}${request.nextUrl.search}`, request.url));
  }

  const isPublicRoute =
    pathname === '/auth' ||
    pathname === '/auth/sign-in' ||
    pathname === '/auth/sign-up' ||
    pathname === '/auth/forgot-password';
  if (
    !isPublicRoute &&
    !request.cookies.has('pos_access_token') &&
    !request.cookies.has('pos_refresh_token')
  ) {
    return NextResponse.redirect(new URL('/auth/sign-in', request.url));
  }

  if (
    pathname === '/auth/sign-in' &&
    (request.cookies.has('pos_access_token') || request.cookies.has('pos_refresh_token'))
  ) {
    return NextResponse.redirect(new URL('/overview', request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico|icon.jpeg|apple-icon.jpeg).*)']
};
