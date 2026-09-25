'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi } from '@/lib/pos-api';
import { FormEvent, useState } from 'react';

export function BusinessSettings({ initialValue }: { initialValue?: Record<string, unknown> }) {
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setSaved(false);
    const fields = Object.fromEntries(new FormData(event.currentTarget));
    const payload = Object.fromEntries(
      Object.entries(fields).map(([key, value]) => [key, value || undefined])
    );
    try {
      await posApi<unknown>('/accounts/organisation', {
        method: 'PATCH',
        body: JSON.stringify(payload)
      });
      setSaved(true);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not update business profile.');
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Business profile</CardTitle>
        <CardDescription>
          Update the organisation details used in your POS workspace.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className='grid gap-3 sm:grid-cols-2'>
          {[
            ['name', 'Business name'],
            ['tin', 'TIN'],
            ['phone', 'Business phone'],
            ['email', 'Business email'],
            ['region', 'Region'],
            ['district', 'District'],
            ['ward', 'Ward'],
            ['street', 'Street'],
            ['address', 'Address']
          ].map(([name, label]) => (
            <label key={name} className='text-sm'>
              {label}
              <Input name={name} defaultValue={String(initialValue?.[name] || '')} />
            </label>
          ))}
          {error && (
            <p role='alert' className='text-destructive sm:col-span-2'>
              {error}
            </p>
          )}
          {saved && (
            <p role='status' className='text-sm text-green-700 sm:col-span-2'>
              Business profile saved.
            </p>
          )}
          <div className='sm:col-span-2'>
            <Button isLoading={busy}>Save business profile</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
