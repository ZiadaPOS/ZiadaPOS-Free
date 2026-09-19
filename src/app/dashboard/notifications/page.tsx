'use client';

import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { posApi, unwrapList } from '@/lib/pos-api';
import { useEffect, useState } from 'react';

type NotificationRecord = { id: string; title: string; body?: string; category?: string; read?: boolean; created_at?: string };

export default function NotificationsPage() {
  const [items, setItems] = useState<NotificationRecord[]>([]);
  const [error, setError] = useState('');
  const [refresh, setRefresh] = useState(0);
  useEffect(() => { posApi<unknown>('/notifications?page=1&page_size=100').then((result) => setItems(unwrapList<NotificationRecord>(result.data))).catch((reason) => setError(reason instanceof Error ? reason.message : 'Could not load notifications.')); }, [refresh]);
  async function markRead(id: string) { try { await posApi<unknown>(`/notifications/${id}/read`, { method: 'PATCH', body: JSON.stringify({}) }); setItems((current) => current.map((item) => item.id === id ? { ...item, read: true } : item)); } catch (reason) { setError(reason instanceof Error ? reason.message : 'Could not update notification.'); } }
  async function markAll() { try { await posApi<unknown>('/notifications/read-all', { method: 'PATCH', body: JSON.stringify({}) }); setItems((current) => current.map((item) => ({ ...item, read: true }))); } catch (reason) { setError(reason instanceof Error ? reason.message : 'Could not update notifications.'); } }
  return <PageContainer pageTitle='Notifications' pageDescription='Stock, sales, credit, supplier, and system alerts.' pageHeaderAction={<div className='flex gap-2'><Button variant='outline' onClick={() => setRefresh((value) => value + 1)}>Refresh</Button><Button onClick={() => void markAll()}>Mark all read</Button></div>}><div className='space-y-3'>{error && <p role='alert' className='text-destructive text-sm'>{error}</p>}{items.map((item) => <Card key={item.id}><CardHeader className='flex flex-row items-start justify-between gap-3 space-y-0'><div><CardTitle className='text-base'>{item.title}</CardTitle><p className='text-muted-foreground mt-1 text-xs'>{item.category || 'Update'}{item.created_at ? ` · ${new Date(item.created_at).toLocaleString()}` : ''}</p></div>{!item.read && <Button size='sm' variant='outline' onClick={() => void markRead(item.id)}>Mark read</Button>}</CardHeader>{item.body && <CardContent className='text-muted-foreground text-sm'>{item.body}</CardContent>}</Card>)}{items.length === 0 && !error && <p className='text-muted-foreground py-12 text-center text-sm'>No notifications yet.</p>}</div></PageContainer>;
}
