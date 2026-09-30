'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { money, posApi, unwrapList } from '@/lib/pos-api';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

type SupplierRecord = Record<string, unknown>;

export default function SupplierDetailPage() {
  const text = useText();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [supplier, setSupplier] = useState<SupplierRecord | null>(null);
  const [deliveries, setDeliveries] = useState<SupplierRecord[]>([]);
  const [amount, setAmount] = useState('');
  const [leadTimeDays, setLeadTimeDays] = useState('7');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  async function load() {
    try {
      const [supplierResult, deliveryResult] = await Promise.all([
        posApi<SupplierRecord>(`/suppliers/${params.id}`),
        posApi<unknown>(`/suppliers/${params.id}/deliveries?page=1&page_size=100`)
      ]);
      setSupplier(supplierResult.data);
      setLeadTimeDays(String(supplierResult.data.lead_time_days || 7));
      setDeliveries(unwrapList<SupplierRecord>(deliveryResult.data));
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : text('Could not load supplier.', 'Imeshindikana kupakia msambazaji.')
      );
    }
  }
  useEffect(() => {
    void load();
  }, [params.id]);
  async function pay() {
    const value = Number(amount);
    if (!value || value <= 0)
      return setMessage(
        text('Enter a payment amount greater than zero.', 'Weka kiasi cha malipo zaidi ya sifuri.')
      );
    try {
      await posApi<unknown>(`/suppliers/${params.id}/pay`, {
        method: 'POST',
        body: JSON.stringify({ amount: value, payment_method: 'cash' })
      });
      setAmount('');
      setMessage(text('Payment recorded.', 'Malipo yamehifadhiwa.'));
      await load();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Payment could not be recorded.');
    }
  }
  async function saveLeadTime() {
    const value = Number(leadTimeDays);
    if (!Number.isInteger(value) || value < 1 || value > 365) {
      setMessage('Lead time must be between 1 and 365 days.');
      return;
    }
    try {
      await posApi(`/suppliers/${params.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name: supplier?.name,
          category: supplier?.category,
          rep: supplier?.rep,
          phone: supplier?.phone,
          email: supplier?.email,
          address: supplier?.address,
          payment_terms: supplier?.payment_terms,
          status: supplier?.status,
          notes: supplier?.notes,
          lead_time_days: value
        })
      });
      setMessage('Lead time updated.');
      await load();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not update lead time.');
    }
  }
  if (error)
    return (
      <div className='p-6'>
        <p role='alert' className='text-destructive'>
          {error}
        </p>
        <Button className='mt-4' onClick={() => router.back()}>
          Back to suppliers
        </Button>
      </div>
    );
  if (!supplier) return <div className='text-muted-foreground p-6'>Loading supplier…</div>;
  const value = (key: string) => Number(supplier[key] || 0);
  return (
    <div className='w-full space-y-5 p-0 sm:p-6'>
      <div className='flex flex-wrap items-start justify-between gap-3 px-4 pt-4 sm:px-0 sm:pt-0'>
        <div>
          <Button variant='ghost' className='mb-2 -ml-3' onClick={() => router.back()}>
            <Icons.chevronLeft className='mr-1 h-4 w-4' />
            Suppliers
          </Button>
          <h1 className='text-2xl font-bold tracking-tight'>
            {String(supplier.name || 'Supplier')}
          </h1>
          <p className='text-muted-foreground'>
            {String(supplier.phone || supplier.contact || 'No contact')} ·{' '}
            {String(supplier.category || 'General')}
          </p>
        </div>
        <span className='bg-primary/10 text-primary rounded-full px-3 py-1 text-sm capitalize'>
          {String(supplier.status || 'active')}
        </span>
      </div>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Total purchased', money(value('total_purchased'))],
          ['Payable', money(value('payable'))],
          ['Deliveries', String(deliveries.length)],
          ['Products supplied', String(supplier.products || '—')]
        ].map(([label, amountValue]) => (
          <Card key={label}>
            <CardContent className='space-y-2 p-4 sm:p-5'>
              <p className='text-muted-foreground text-sm'>{label}</p>
              <p className='truncate text-xl font-semibold sm:text-2xl'>{amountValue}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]'>
        <Card>
          <CardHeader>
            <CardTitle>Delivery lead time</CardTitle>
            <CardDescription>
              Used when calculating reorder suggestions for medicines assigned to this supplier.
            </CardDescription>
          </CardHeader>
          <CardContent className='flex gap-2'>
            <Input
              type='number'
              min='1'
              max='365'
              value={leadTimeDays}
              onChange={(event) => setLeadTimeDays(event.target.value)}
              aria-label='Delivery lead time in days'
            />
            <Button onClick={() => void saveLeadTime()}>Save</Button>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Delivery history</CardTitle>
            <CardDescription>
              Invoices and stock deliveries recorded for this supplier.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className='divide-y'>
              {deliveries.length ? (
                deliveries.map((delivery) => (
                  <div
                    key={String(delivery.id)}
                    className='flex flex-wrap items-center justify-between gap-3 py-3 text-sm'
                  >
                    <div>
                      <p className='font-medium'>{String(delivery.invoice_number || 'Delivery')}</p>
                      <p className='text-muted-foreground'>
                        {String(delivery.delivery_date || delivery.created_at || '—')} ·{' '}
                        {String(delivery.items_count || 0)} items
                      </p>
                    </div>
                    <p className='font-semibold'>{money(valueFrom(delivery, 'total_value'))}</p>
                  </div>
                ))
              ) : (
                <p className='text-muted-foreground py-8 text-center'>No deliveries found.</p>
              )}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Pay supplier</CardTitle>
            <CardDescription>Record a payment against the supplier balance.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            <Input
              type='number'
              min='1'
              placeholder='Amount (TZS)'
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
            />
            <Button className='w-full' onClick={() => void pay()} disabled={!value('payable')}>
              Record payment
            </Button>
            {message && (
              <p role='status' className='text-muted-foreground text-sm'>
                {message}
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function valueFrom(record: SupplierRecord, key: string): number {
  return Number(record[key] || 0);
}
