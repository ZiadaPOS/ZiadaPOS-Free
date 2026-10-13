'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList } from '@/lib/pos-api';
import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { FormDialog } from '@/components/ui/form-dialog';

export function StoreWorkspace() {
  const text = useText();
  const [stores, setStores] = useState<Record<string, unknown>[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState('');
  const load = useCallback(() => {
    void posApi<unknown>('/stores')
      .then((result) => setStores(unwrapList<Record<string, unknown>>(result.data)))
      .catch((reason) =>
        setMessage(reason instanceof Error ? reason.message : 'Could not load stores.')
      );
  }, []);
  useEffect(() => {
    load();
  }, [load]);
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<unknown>('/stores', { method: 'POST', body: JSON.stringify(values) });
      setShowForm(false);
      load();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not create store.');
    } finally {
      setPending(false);
    }
  }
  return (
    <div className='space-y-4'>
      <div className='flex flex-col items-stretch justify-between gap-3 sm:flex-row sm:items-start'>
        <div>
          <h1 className='text-2xl font-bold tracking-tight'>{text('Stores', 'Matawi')}</h1>
          <p className='text-muted-foreground'>Manage locations and active branch context.</p>
        </div>
        <Button className='w-full sm:w-auto' onClick={() => setShowForm(true)}>
          {text('Add store', 'Ongeza tawi')}
        </Button>
      </div>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          [text('Total stores', 'Jumla ya matawi'), stores.length],
          [
            text('Active stores', 'Matawi hai'),
            stores.filter((store) => store.is_active !== false).length
          ],
          [
            text('Total staff', 'Jumla ya wafanyakazi'),
            stores.reduce((sum, store) => sum + Number(store.staff || 0), 0)
          ],
          [
            'Open stores',
            stores.filter((store) => store.status === 'open' || store.is_active !== false).length
          ]
        ].map(([label, value]) => (
          <Card key={label}>
            <CardContent className='space-y-2 p-4 sm:p-5'>
              <p className='text-muted-foreground text-sm'>{label}</p>
              <p className='text-3xl font-semibold'>{value}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-3'>
        {stores.map((store) => (
          <Card key={String(store.id)}>
            <CardHeader>
              <CardTitle>{String(store.name || store.address || 'Store')}</CardTitle>
              <CardDescription>
                {String(store.area || store.address || 'No address')}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3 text-sm'>
              <div className='flex justify-between'>
                <span className='text-muted-foreground'>Code</span>
                <span>{String(store.code || '—')}</span>
              </div>
              <div className='flex justify-between'>
                <span className='text-muted-foreground'>Email</span>
                <span className='truncate'>{String(store.email || '—')}</span>
              </div>
              <span className='bg-primary/10 text-primary inline-flex rounded-full px-2 py-1 text-xs'>
                {store.is_active === false ? 'Inactive' : 'Active'}
              </span>
            </CardContent>
          </Card>
        ))}
        {!stores.length && (
          <Card>
            <CardContent className='text-muted-foreground p-8 text-center'>
              No stores found.
            </CardContent>
          </Card>
        )}
      </div>
      {message && (
        <p role='alert' className='text-destructive text-sm'>
          {message}
        </p>
      )}
      <FormDialog
        open={showForm}
        onOpenChange={setShowForm}
        title='New store'
        description='Add a location to your business.'
      >
        <form onSubmit={create} className='mt-5 space-y-3'>
          {message && (
            <p role='alert' className='text-destructive text-sm'>
              {message}
            </p>
          )}
          <label className='block text-sm'>
            Store name
            <Input name='name' required />
          </label>
          <label className='block text-sm'>
            Address
            <Input name='address' />
          </label>
          <label className='block text-sm'>
            Area
            <Input name='area' />
          </label>
          <label className='block text-sm'>
            Code
            <Input name='code' />
          </label>
          <label className='block text-sm'>
            Email
            <Input name='email' type='email' />
          </label>
          <Button isLoading={pending}>Create store</Button>
        </form>
      </FormDialog>
    </div>
  );
}
