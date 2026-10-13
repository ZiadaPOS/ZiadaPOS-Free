'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { FormDialog } from '@/components/ui/form-dialog';
import { posApi, type PosCustomer } from '@/lib/pos-api';
import { FormEvent, useState } from 'react';
import { useText } from '@/lib/i18n';

export function CreateCustomer({ onCreated }: { onCreated: () => void }) {
  const text = useText();
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
      setError(
        reason instanceof Error
          ? reason.message
          : text('Could not create customer.', 'Imeshindikana kuunda mteja.')
      );
    } finally {
      setPending(false);
    }
  }
  return (
    <FormDialog
      open
      onOpenChange={(open) => !open && onCreated()}
      title={text('New customer', 'Mteja mpya')}
      description='Add a profile for repeat sales and credit tracking.'
      size='wide'
    >
      <form onSubmit={submit} className='mt-5 grid gap-4 sm:grid-cols-2'>
        <label className='grid gap-2 text-sm font-semibold'>
          {text('Full name', 'Jina kamili')}
          <Input name='name' required />
        </label>
        <label className='grid gap-2 text-sm font-semibold'>
          {text('Phone', 'Simu')}
          <Input name='phone' />
        </label>
        <label className='grid gap-2 text-sm font-semibold'>
          {text('Email', 'Barua pepe')}
          <Input type='email' name='email' />
        </label>
        <label className='grid gap-2 text-sm font-semibold'>
          Segment
          <select
            name='segment'
            className='bg-background h-11 w-full rounded-lg border border-foreground/20 px-3 text-sm font-normal'
          >
            <option>New</option>
            <option>Regular</option>
            <option>VIP</option>
            <option>Occasional</option>
          </select>
        </label>
        <label className='grid gap-2 text-sm font-semibold'>
          {text('Credit limit (TZS, -1 for unlimited)', 'Kikomo cha mkopo (TZS, -1 bila kikomo)')}
          <Input type='number' name='credit_limit' defaultValue='0' />
        </label>
        <label className='grid gap-2 text-sm font-semibold'>
          {text('Notes', 'Maelezo')}
          <Input name='notes' />
        </label>
        {error && (
          <p
            role='alert'
            className='rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive sm:col-span-2'
          >
            {error}
          </p>
        )}
        <Button isLoading={pending} className='h-12 w-full sm:col-span-2'>
          Save customer
        </Button>
      </form>
    </FormDialog>
  );
}
