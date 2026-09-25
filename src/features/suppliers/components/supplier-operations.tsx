'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList } from '@/lib/pos-api';
import { FormEvent, useEffect, useState } from 'react';

export function SupplierOperations() {
  const [suppliers, setSuppliers] = useState<Record<string, unknown>[]>([]);
  const [message, setMessage] = useState('');
  const [reload, setReload] = useState(0);
  useEffect(() => {
    posApi<unknown>('/suppliers?page=1&page_size=100')
      .then((result) => setSuppliers(unwrapList<Record<string, unknown>>(result.data)))
      .catch((reason) =>
        setMessage(reason instanceof Error ? reason.message : 'Could not load suppliers.')
      );
  }, [reload]);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    const supplierId = String(values.supplier_id || '');
    try {
      if (values.operation === 'delivery')
        await posApi<unknown>(`/suppliers/${supplierId}/deliveries`, {
          method: 'POST',
          body: JSON.stringify({
            invoice_number: values.invoice_number,
            delivery_date: new Date(`${values.delivery_date}T00:00:00`).toISOString(),
            total_value: Number(values.total_value),
            items_count: Number(values.items_count || 1),
            is_paid: false,
            notes: values.notes
          })
        });
      else
        await posApi<unknown>(`/suppliers/${supplierId}/pay`, {
          method: 'POST',
          body: JSON.stringify({
            amount: Number(values.amount),
            payment_method: values.payment_method,
            reference: values.reference,
            notes: values.notes
          })
        });
      setMessage('Supplier record saved.');
      setReload((value) => value + 1);
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not save supplier record.');
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Deliveries and payments</CardTitle>
        <CardDescription>Record an invoice delivery or pay a supplier balance.</CardDescription>
      </CardHeader>
      <CardContent className='grid gap-6 lg:grid-cols-2'>
        <form onSubmit={submit} className='space-y-3'>
          <input type='hidden' name='operation' value='delivery' />
          <label className='block text-sm'>
            Supplier
            <select
              className='bg-background mt-1 h-9 w-full rounded-md border px-3'
              name='supplier_id'
              required
            >
              {suppliers.map((s) => (
                <option value={String(s.id)} key={String(s.id)}>
                  {String(s.name)}
                </option>
              ))}
            </select>
          </label>
          <label className='block text-sm'>
            Invoice number
            <Input name='invoice_number' />
          </label>
          <label className='block text-sm'>
            Delivery date
            <Input name='delivery_date' type='date' required />
          </label>
          <label className='block text-sm'>
            Invoice total (TZS)
            <Input name='total_value' type='number' min='1' required />
          </label>
          <label className='block text-sm'>
            Items count
            <Input name='items_count' type='number' min='1' defaultValue='1' />
          </label>
          <Button>Record delivery</Button>
        </form>
        <form onSubmit={submit} className='space-y-3'>
          <input type='hidden' name='operation' value='payment' />
          <label className='block text-sm'>
            Supplier
            <select
              className='bg-background mt-1 h-9 w-full rounded-md border px-3'
              name='supplier_id'
              required
            >
              {suppliers.map((s) => (
                <option value={String(s.id)} key={String(s.id)}>
                  {String(s.name)}
                </option>
              ))}
            </select>
          </label>
          <label className='block text-sm'>
            Payment amount (TZS)
            <Input name='amount' type='number' min='1' required />
          </label>
          <label className='block text-sm'>
            Method
            <select
              name='payment_method'
              className='bg-background mt-1 h-9 w-full rounded-md border px-3'
            >
              <option value='cash'>Cash</option>
              <option value='bank'>Bank</option>
              <option value='mobile'>Mobile</option>
            </select>
          </label>
          <label className='block text-sm'>
            Reference
            <Input name='reference' />
          </label>
          <Button>Record payment</Button>
        </form>
        {message && (
          <p role='status' className='text-muted-foreground text-sm lg:col-span-2'>
            {message}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
