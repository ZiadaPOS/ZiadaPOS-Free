'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { posApi } from '@/lib/pos-api';
import { useEffect, useState } from 'react';

export function StoreSwitcher() {
  const [stores, setStores] = useState<{ id: string; name: string }[]>([]);
  const [error, setError] = useState('');
  useEffect(() => {
    posApi<unknown>('/accounts/me')
      .then((result) => {
        const payload = result.data as {
          stores?: { id: string; name: string }[];
          store?: { id: string };
        };
        setStores(
          (payload.stores || []).map((store) => ({
            ...store,
            name: `${store.name}${store.id === payload.store?.id ? ' (current)' : ''}`
          }))
        );
      })
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load branches.')
      );
  }, []);
  async function change(store_id: string) {
    try {
      const session = await fetch('/api/pos-session', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ store_id })
      });
      if (!session.ok) {
        const data = await session.json();
        throw new Error(data.message || 'Could not switch branch.');
      }
      window.location.reload();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not switch branch.');
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Active branch</CardTitle>
        <CardDescription>Branch context is embedded in your API access token.</CardDescription>
      </CardHeader>
      <CardContent className='space-y-2'>
        <select
          className='bg-background h-9 w-full rounded-md border px-3'
          onChange={(event) => void change(event.target.value)}
          defaultValue=''
        >
          <option value='' disabled>
            Select branch
          </option>
          {stores.map((store) => (
            <option value={store.id} key={store.id}>
              {store.name}
            </option>
          ))}
        </select>
        {error && <p className='text-destructive text-sm'>{error}</p>}
      </CardContent>
    </Card>
  );
}
