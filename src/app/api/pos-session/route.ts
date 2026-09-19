import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';

const API_URL = (process.env.POS_API_URL || 'http://localhost:8080/api/v1').replace(/\/+$/, '');

export async function POST(request: NextRequest) {
  const payload = await request.json();
  const upstream = await fetch(`${API_URL}/auth/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload), cache: 'no-store' });
  const envelope = await upstream.json().catch(() => ({}));
  const pair = envelope.data;
  if (!upstream.ok || !pair?.access_token || !pair?.refresh_token) return NextResponse.json(envelope, { status: upstream.status });
  const response = NextResponse.json({ success: true, message: envelope.message, data: { user: pair.user } });
  response.cookies.set('pos_access_token', pair.access_token, { httpOnly: true, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/', maxAge: pair.expires_in || 900 });
  response.cookies.set('pos_refresh_token', pair.refresh_token, { httpOnly: true, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/', maxAge: 60 * 60 * 24 * 7 });
  return response;
}

export async function GET() {
  const jar = await cookies();
  const accessToken = jar.get('pos_access_token')?.value;
  const refreshToken = jar.get('pos_refresh_token')?.value;
  if (accessToken) {
    const check = await fetch(`${API_URL}/accounts/me`, { headers: { Authorization: `Bearer ${accessToken}` }, cache: 'no-store' });
    if (check.ok) return NextResponse.json({ success: true, data: { authenticated: true } });
  }
  if (!refreshToken) {
    const response = NextResponse.json({ success: false }, { status: 401 });
    response.cookies.delete('pos_access_token');
    response.cookies.delete('pos_refresh_token');
    return response;
  }
  const renewed = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: refreshToken }), cache: 'no-store' });
  const envelope = await renewed.json().catch(() => ({}));
  const pair = envelope.data || envelope;
  if (!renewed.ok || !pair.access_token || !pair.refresh_token) {
    const response = NextResponse.json({ success: false }, { status: 401 });
    response.cookies.delete('pos_access_token');
    response.cookies.delete('pos_refresh_token');
    return response;
  }
  const response = NextResponse.json({ success: true, data: { authenticated: true } });
  response.cookies.set('pos_access_token', pair.access_token, { httpOnly: true, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/', maxAge: pair.expires_in || 900 });
  response.cookies.set('pos_refresh_token', pair.refresh_token, { httpOnly: true, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/', maxAge: 60 * 60 * 24 * 7 });
  return response;
}

export async function PUT(request: NextRequest) {
  const jar = await cookies();
  const accessToken = jar.get('pos_access_token')?.value;
  const refreshToken = jar.get('pos_refresh_token')?.value;
  const { store_id } = await request.json();
  let currentRefreshToken = refreshToken || '';
  let currentAccessToken = accessToken || '';
  if (!currentAccessToken && refreshToken) {
    const renewed = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: refreshToken }), cache: 'no-store' });
    const renewal = await renewed.json().catch(() => ({}));
    const pair = renewal.data || renewal;
    currentAccessToken = pair.access_token || '';
    currentRefreshToken = pair.refresh_token || currentRefreshToken;
  }
  let response = currentAccessToken ? await fetch(`${API_URL}/accounts/switch-store`, { method: 'POST', headers: { Authorization: `Bearer ${currentAccessToken}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ store_id }), cache: 'no-store' }) : new Response(JSON.stringify({ message: 'Session expired' }), { status: 401 });
  if (response.status === 401 && currentRefreshToken) {
    const renewed = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: currentRefreshToken }), cache: 'no-store' });
    const renewal = await renewed.json().catch(() => ({}));
    const pair = renewal.data || renewal;
    if (renewed.ok && pair.access_token) {
      currentAccessToken = pair.access_token;
      currentRefreshToken = pair.refresh_token || currentRefreshToken;
      response = await fetch(`${API_URL}/accounts/switch-store`, { method: 'POST', headers: { Authorization: `Bearer ${currentAccessToken}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ store_id }), cache: 'no-store' });
    }
  }
  const envelope = await response.json().catch(() => ({}));
  if (!response.ok) return NextResponse.json(envelope, { status: response.status });
  if (!currentRefreshToken) return NextResponse.json({ message: 'Refresh token is missing' }, { status: 401 });
  const renewed = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ refresh_token: currentRefreshToken }), cache: 'no-store' });
  const renewedEnvelope = await renewed.json().catch(() => ({}));
  const pair = renewedEnvelope.data || renewedEnvelope;
  if (!renewed.ok || !pair.access_token || !pair.refresh_token) return NextResponse.json(renewedEnvelope, { status: renewed.status });
  const result = NextResponse.json({ success: true, data: { user: pair.user } });
  result.cookies.set('pos_access_token', pair.access_token, { httpOnly: true, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/', maxAge: pair.expires_in || 900 });
  result.cookies.set('pos_refresh_token', pair.refresh_token, { httpOnly: true, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/', maxAge: 60 * 60 * 24 * 7 });
  return result;
}

export async function DELETE() {
  const response = NextResponse.json({ success: true });
  response.cookies.delete('pos_access_token');
  response.cookies.delete('pos_refresh_token');
  return response;
}
