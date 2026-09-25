'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { money, posApi, type PosTransaction } from '@/lib/pos-api';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

export default function SaleDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [transaction, setTransaction] = useState<PosTransaction | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [email, setEmail] = useState('');

  useEffect(() => {
    void posApi<PosTransaction>(`/pos/transactions/${params.id}`)
      .then((result) => setTransaction(result.data))
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load this sale.')
      );
  }, [params.id]);
  async function sendReceipt() {
    if (!transaction) return;
    setBusy(true);
    try {
      await posApi<unknown>(`/pos/transactions/${transaction.id}/receipt`, {
        method: 'POST',
        body: JSON.stringify({ email })
      });
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Receipt could not be sent.');
    } finally {
      setBusy(false);
    }
  }
  if (error)
    return (
      <div className='p-6'>
        <p role='alert' className='text-destructive'>
          {error}
        </p>
        <Button className='mt-4' onClick={() => router.back()}>
          Back to sales
        </Button>
      </div>
    );
  if (!transaction) return <div className='text-muted-foreground p-6'>Loading sale…</div>;
  const lines = transaction.lines || [];
  return (
    <div className='space-y-5 p-4 sm:p-6'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <Button variant='ghost' className='mb-2 -ml-3' onClick={() => router.back()}>
            <Icons.chevronLeft className='mr-1 h-4 w-4' />
            Sales
          </Button>
          <h1 className='text-2xl font-bold tracking-tight'>{transaction.txn_number}</h1>
          <p className='text-muted-foreground'>Sale details and receipt history</p>
        </div>
        <span className='bg-primary/10 text-primary rounded-full px-3 py-1 text-sm capitalize'>
          {transaction.status}
        </span>
      </div>
      <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]'>
        <Card>
          <CardHeader>
            <CardTitle>Items</CardTitle>
            <CardDescription>
              {new Date(transaction.created_at).toLocaleString()} ·{' '}
              {transaction.customer_name || 'Walk-in customer'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className='divide-y'>
              {lines.length ? (
                lines.map((line) => (
                  <div
                    key={line.id}
                    className='flex items-center justify-between gap-3 py-3 text-sm'
                  >
                    <div>
                      <p className='font-medium'>{line.product_name}</p>
                      <p className='text-muted-foreground'>
                        {line.qty} × {money(line.unit_price)}
                      </p>
                    </div>
                    <span className='font-semibold'>{money(line.total)}</span>
                  </div>
                ))
              ) : (
                <p className='text-muted-foreground py-6 text-center'>
                  Line items are not available.
                </p>
              )}
            </div>
          </CardContent>
        </Card>
        <div className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>Payment summary</CardTitle>
            </CardHeader>
            <CardContent className='space-y-2 text-sm'>
              <div className='flex justify-between'>
                <span>Subtotal</span>
                <span>{money(transaction.subtotal)}</span>
              </div>
              <div className='flex justify-between'>
                <span>Discount</span>
                <span>{money(transaction.discount_amount)}</span>
              </div>
              <div className='flex justify-between'>
                <span>Tax</span>
                <span>{money(transaction.tax_amount)}</span>
              </div>
              <div className='flex justify-between border-t pt-3 text-base font-semibold'>
                <span>Total</span>
                <span>{money(transaction.total)}</span>
              </div>
              <p className='text-muted-foreground pt-2'>Paid with {transaction.payment_method}</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Send receipt</CardTitle>
              <CardDescription>Email a copy to the customer.</CardDescription>
            </CardHeader>
            <CardContent className='flex gap-2'>
              <Input
                type='email'
                placeholder='Customer email'
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
              <Button isLoading={busy} onClick={() => void sendReceipt()}>
                Send
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
