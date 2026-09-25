'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, type PosCustomer } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useState } from 'react';

export function CreateCustomer({ onCreated }: { onCreated: () => void }) {
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<PosCustomer>('/customers', {
        method: 'POST',
        body: JSON.stringify({
          name: values.name,
          phone: values.phone,
          email: values.email,
          segment: values.segment,
          credit_limit: values.credit_limit === '' ? undefined : Number(values.credit_limit),
          notes: values.notes
        })
      });
      onCreated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create customer.');
    } finally {
      setPending(false);
    }
  }
  return (
    <div
      className='fixed inset-0 z-50 grid place-items-center overflow-y-auto bg-black/60 p-4'
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onCreated();
      }}
    >
      <Card
        className='my-4 w-full max-w-2xl shadow-2xl'
        onMouseDown={(event) => event.stopPropagation()}
      >
        <CardHeader className='flex flex-row items-center justify-between space-y-0'>
          <div>
            <CardTitle>New customer</CardTitle>
            <CardDescription>Add a profile for repeat sales and credit tracking.</CardDescription>
          </div>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            aria-label='Close new customer'
            onClick={onCreated}
          >
            <Icons.close className='h-5 w-5' />
          </Button>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className='grid gap-3 sm:grid-cols-2'>
            <label className='text-sm'>
              Full name
              <Input name='name' required />
            </label>
            <label className='text-sm'>
              Phone
              <Input name='phone' />
            </label>
            <label className='text-sm'>
              Email
              <Input type='email' name='email' />
            </label>
            <label className='text-sm'>
              Segment
              <select
                name='segment'
                className='bg-background mt-1 h-9 w-full rounded-md border px-3'
              >
                <option>New</option>
                <option>Regular</option>
                <option>VIP</option>
                <option>Occasional</option>
              </select>
            </label>
            <label className='text-sm'>
              Credit limit (TZS, -1 for unlimited)
              <Input type='number' name='credit_limit' defaultValue='0' />
            </label>
            <label className='text-sm'>
              Notes
              <Input name='notes' />
            </label>
            {error && <p className='text-destructive sm:col-span-2'>{error}</p>}
            <div className='sm:col-span-2'>
              <Button isLoading={pending}>Save customer</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
