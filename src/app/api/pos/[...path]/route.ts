import { cookies } from 'next/headers';
import { NextRequest, NextResponse } from 'next/server';

const API_URL = (process.env.POS_API_URL || 'http://localhost:8080/api/v1').replace(/\/+$/, '');

async function forward(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const jar = await cookies();
  const requestBody =
    request.method === 'GET' || request.method === 'HEAD' ? undefined : await request.arrayBuffer();
  const accessToken = jar.get('pos_access_token')?.value;
  let refreshedPair:
    | { access_token?: string; refresh_token?: string; expires_in?: number }
    | undefined;
  let response = await fetch(
    `${API_URL}/${path.map(encodeURIComponent).join('/')}${request.nextUrl.search}`,
    {
      method: request.method,
      headers: {
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
        ...(request.headers.get('content-type')
          ? { 'Content-Type': request.headers.get('content-type')! }
          : {})
      },
      ...(requestBody === undefined ? {} : { body: requestBody }),
      cache: 'no-store'
    }
  );

  if (response.status === 401 && jar.get('pos_refresh_token')?.value && path[0] !== 'auth') {
    const refreshed = await fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: jar.get('pos_refresh_token')!.value }),
      cache: 'no-store'
    });
    if (refreshed.ok) {
      const envelope = await refreshed.json();
      const pair = envelope.data || envelope;
      if (pair.access_token && pair.refresh_token) {
        refreshedPair = pair;
        jar.set('pos_access_token', pair.access_token, {
          httpOnly: true,
          secure: process.env.NODE_ENV === 'production',
          sameSite: 'lax',
          path: '/',
          maxAge: pair.expires_in || 900
        });
        jar.set('pos_refresh_token', pair.refresh_token, {
          httpOnly: true,
          secure: process.env.NODE_ENV === 'production',
          sameSite: 'lax',
          path: '/',
          maxAge: 60 * 60 * 24 * 7
        });
        response = await fetch(
          `${API_URL}/${path.map(encodeURIComponent).join('/')}${request.nextUrl.search}`,
          {
            method: request.method,
            headers: {
              Authorization: `Bearer ${pair.access_token}`,
              ...(request.headers.get('content-type')
                ? { 'Content-Type': request.headers.get('content-type')! }
                : {})
            },
            ...(requestBody === undefined ? {} : { body: requestBody }),
            cache: 'no-store'
          }
        );
      }
    }
  }

  const payload = await response.text();
  const responseHeaders = new Headers();
  responseHeaders.set('Content-Type', response.headers.get('Content-Type') || 'application/json');
  const disposition = response.headers.get('Content-Disposition');
  if (disposition) responseHeaders.set('Content-Disposition', disposition);
  const result = new NextResponse(payload, { status: response.status, headers: responseHeaders });
  if (response.status === 401) {
    result.cookies.delete('pos_access_token');
    result.cookies.delete('pos_refresh_token');
  }
  if (response.ok && refreshedPair) {
    result.cookies.set('pos_access_token', refreshedPair.access_token!, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: refreshedPair.expires_in || 900
    });
    result.cookies.set('pos_refresh_token', refreshedPair.refresh_token!, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: 60 * 60 * 24 * 7
    });
  }
  return result;
}

export const GET = forward;
export const POST = forward;
export const PATCH = forward;
export const PUT = forward;
export const DELETE = forward;
