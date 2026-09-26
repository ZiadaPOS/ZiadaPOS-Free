'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useEffect, useState } from 'react';
import { StaffTable } from '@/features/staff/components/staff-table';

export function ResourcePage() {
  const endpoint = '/staff?page=1&page_size=50';
  const title = 'Staff';
  const [records, setRecords] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [pending, setPending] = useState(false);
  const [reload, setReload] = useState(0);
  useEffect(() => {
    let active = true;
    posApi<unknown>(endpoint)
      .then((res) => {
        if (active) setRecords(unwrapList<Record<string, unknown>>(res.data));
      })
      .catch((reason) => {
        if (active) setError(reason instanceof Error ? reason.message : 'Request failed');
      });
    return () => {
      active = false;
    };
  }, [endpoint, reload]);
  const fields = [
    ['first_name', 'First name'],
    ['last_name', 'Last name'],
    ['phone', 'Phone'],
    ['email', 'Email'],
    ['role', 'Role'],
    ['shift', 'Shift'],
    ['pin', '4 digit POS PIN']
  ];
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      const payload = Object.fromEntries(
        Object.entries(values).map(([key, value]) => [
          key,
          ['amount', 'payment_terms'].includes(key) ? Number(value) : value
        ])
      );
      await posApi<unknown>(endpoint.split('?')[0], {
        method: 'POST',
        body: JSON.stringify(payload)
      });
      setShowForm(false);
      setReload((n) => n + 1);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not save the record.');
    } finally {
      setPending(false);
    }
  }
  return (
    <>
      <div className='flex flex-col items-stretch justify-between gap-3 sm:flex-row sm:items-start'>
        <div>
          <h1 className='text-2xl font-bold tracking-tight'>{title}</h1>
          <p className='text-muted-foreground'>
            Manage {title.toLowerCase()} for the active store.
          </p>
        </div>
        {fields.length > 0 && (
          <Button className='w-full sm:w-auto' onClick={() => setShowForm((value) => !value)}>
            {showForm ? 'Close form' : `Add ${'staff member'}`}
          </Button>
        )}
      </div>
      {error && (
        <p className='text-destructive text-sm' role='alert'>
          {error}
        </p>
      )}
      {showForm && (
        <div
          className='fixed inset-0 z-50 flex items-end justify-center bg-black/60 p-0 sm:items-center sm:p-4'
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setShowForm(false);
          }}
        >
          <Card
            className='min-h-[50svh] w-full rounded-b-none shadow-2xl sm:min-h-0 sm:max-w-2xl sm:rounded-xl'
            onMouseDown={(event) => event.stopPropagation()}
          >
            <CardHeader className='flex flex-row items-center justify-between space-y-0'>
              <CardTitle>New staff member</CardTitle>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                aria-label='Close new staff member dialog'
                onClick={() => setShowForm(false)}
              >
                <Icons.close className='size-4' />
              </Button>
            </CardHeader>
            <CardContent>
              <form onSubmit={create} className='grid gap-4 sm:grid-cols-2'>
                {fields.map(([name, label]) => (
                  <div key={name} className='space-y-1'>
                    <label className='text-sm font-medium' htmlFor={name}>
                      {label}
                    </label>
                    <Input
                      id={name}
                      name={name}
                      required={name === 'name' || name === 'title' || name === 'first_name'}
                      type={name === 'amount' ? 'number' : 'text'}
                    />
                  </div>
                ))}
                <div className='sm:col-span-2'>
                  <Button className='w-full' isLoading={pending}>
                    Save
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      )}
      <StaffTable records={records} onUpdated={() => setReload((n) => n + 1)} />
    </>
  );
}
