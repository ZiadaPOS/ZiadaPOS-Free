'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { money, posApi, unwrapList, type PosCreditTab, type PosCustomer } from '@/lib/pos-api';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

export default function CreditDetailPage() {
  const text = useText();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [customer, setCustomer] = useState<PosCustomer | null>(null);
  const [tabs, setTabs] = useState<PosCreditTab[]>([]);
  const [amount, setAmount] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [reminding, setReminding] = useState(false);

  async function load() {
    try {
      const [customerResult, creditResult] = await Promise.all([
        posApi<PosCustomer>(`/customers/${params.id}`),
        posApi<unknown>('/credits?page=1&page_size=100')
      ]);
      setCustomer(customerResult.data);
      setTabs(
        unwrapList<PosCreditTab>(creditResult.data).filter((tab) => tab.customer_id === params.id)
      );
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load credit account.');
    }
  }
  useEffect(() => {
    void load();
  }, [params.id]);
  async function recordPayment() {
    const value = Number(amount);
    if (!value || value <= 0) return setMessage('Enter a payment amount greater than zero.');
    try {
      await posApi<unknown>(`/credits/customers/${params.id}/payments`, {
        method: 'POST',
        body: JSON.stringify({ amount: value, method: 'Cash' })
      });
      setAmount('');
      setMessage('Payment recorded.');
      await load();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Payment could not be recorded.');
    }
  }
  async function sendReminder() {
    setReminding(true);
    setMessage('');
    try {
      await posApi<unknown>(`/credits/customers/${params.id}/remind`, { method: 'POST' });
      setMessage('SMS reminder sent.');
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'SMS reminder could not be sent.');
    } finally {
      setReminding(false);
    }
  }
  if (error)
    return (
      <div className='p-6'>
        <p role='alert' className='text-destructive'>
          {error}
        </p>
        <Button className='mt-4' onClick={() => router.back()}>
          Back to credits
        </Button>
      </div>
    );
  if (!customer) return <div className='text-muted-foreground p-6'>Loading credit account…</div>;
  const balance = tabs.reduce((sum, tab) => sum + tab.balance, 0);
  const original = tabs.reduce((sum, tab) => sum + tab.amount, 0);
  const paid = tabs.reduce((sum, tab) => sum + tab.amount_paid, 0);
  return (
    <div className='w-full space-y-5 p-0 sm:p-6'>
      <div className='flex flex-wrap items-start justify-between gap-3 px-4 pt-4 sm:px-0 sm:pt-0'>
        <div>
          <Button variant='ghost' className='mb-2 -ml-3' onClick={() => router.back()}>
            <Icons.chevronLeft className='mr-1 h-4 w-4' />
            Credits
          </Button>
          <h1 className='text-2xl font-bold tracking-tight'>{customer.name}</h1>
          <p className='text-muted-foreground'>
            {customer.phone || text('No phone', 'Hakuna simu')} ·{' '}
            {text('Credit account', 'Akaunti ya mkopo')}
          </p>
        </div>
        <span className='bg-destructive/10 text-destructive rounded-full px-3 py-1 text-sm'>
          {tabs.some((tab) => tab.is_overdue) ? 'Overdue' : 'Open'}
        </span>
      </div>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          [text('Original debt', 'Deni la mwanzo'), money(original)],
          [text('Paid', 'Imelipwa'), money(paid)],
          [text('Balance', 'Salio'), money(balance)],
          [text('Open tabs', 'Mikopo wazi'), tabs.length]
        ].map(([label, value]) => (
          <Card key={label}>
            <CardContent className='space-y-2 p-4 sm:p-5'>
              <p className='text-muted-foreground text-sm'>{label}</p>
              <p className='truncate text-xl font-semibold sm:text-2xl'>{value}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]'>
        <Card>
          <CardHeader>
            <CardTitle>{text('Credit history', 'Historia ya mkopo')}</CardTitle>
            <CardDescription>
              {text('All credit tabs linked to this customer.', 'Mikopo yote ya mteja huyu.')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className='divide-y'>
              {tabs.length ? (
                tabs.map((tab) => (
                  <div
                    key={tab.id}
                    className='flex flex-wrap items-center justify-between gap-3 py-3 text-sm'
                  >
                    <div>
                      <p className='font-medium capitalize'>
                        {tab.status}
                        {tab.is_overdue ? ' · overdue' : ''}
                      </p>
                      <p className='text-muted-foreground'>
                        {new Date(tab.created_at).toLocaleString()}
                        {tab.due_date
                          ? ` · due ${new Date(tab.due_date).toLocaleDateString()}`
                          : ''}
                      </p>
                    </div>
                    <div className='text-right'>
                      <p className='font-semibold'>{money(tab.balance)}</p>
                      <p className='text-muted-foreground text-xs'>
                        {money(tab.amount_paid)} paid of {money(tab.amount)}
                      </p>
                    </div>
                  </div>
                ))
              ) : (
                <p className='text-muted-foreground py-8 text-center'>No credit history found.</p>
              )}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{text('Record payment', 'Rekodi malipo')}</CardTitle>
            <CardDescription>
              {text(
                'Apply a cash payment to this account.',
                'Weka malipo ya fedha kwenye akaunti hii.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            <Button
              variant='outline'
              className='w-full'
              onClick={() => void sendReminder()}
              disabled={!customer.phone || reminding}
              isLoading={reminding}
            >
              <Icons.send className='size-4' />{' '}
              {text('Send SMS reminder', 'Tuma kikumbusho cha SMS')}
            </Button>
            <Input
              type='number'
              min='1'
              placeholder={text('Amount (TZS)', 'Kiasi (TZS)')}
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
            />
            <Button className='w-full' onClick={() => void recordPayment()} disabled={!balance}>
              {text('Record payment', 'Rekodi malipo')}
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
