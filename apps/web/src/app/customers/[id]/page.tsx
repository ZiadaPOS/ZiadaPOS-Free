'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Icons } from '@/components/icons';
import { money, posApi, unwrapList, type PosCustomer, type PosTransaction } from '@/lib/pos-api';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

export default function CustomerDetailPage() {
  const text = useText();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [customer, setCustomer] = useState<PosCustomer | null>(null);
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const [email, setEmail] = useState('');
  const [sms, setSms] = useState('');
  const [smsStatus, setSmsStatus] = useState('');
  const [sendingSms, setSendingSms] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    void Promise.all([
      posApi<PosCustomer>(`/customers/${params.id}`),
      posApi<unknown>('/pos/transactions?page=1&page_size=100')
    ])
      .then(([customerResult, transactionResult]) => {
        setCustomer(customerResult.data);
        setTransactions(
          unwrapList<PosTransaction>(transactionResult.data).filter(
            (transaction) =>
              transaction.customer_id === params.id ||
              transaction.customer_name === customerResult.data.name
          )
        );
      })
      .catch((reason) =>
        setError(
          reason instanceof Error
            ? reason.message
            : text('Could not load customer.', 'Imeshindikana kupakia mteja.')
        )
      );
  }, [params.id]);

  if (error)
    return (
      <div className='p-6'>
        <p role='alert' className='text-destructive'>
          {error}
        </p>
        <Button className='mt-4' onClick={() => router.back()}>
          Back to customers
        </Button>
      </div>
    );
  if (!customer) return <div className='text-muted-foreground p-6'>Loading customer…</div>;

  const visits = customer.transaction_count ?? transactions.length;
  const average = visits ? customer.total_spent / visits : 0;
  const currentCustomer = customer;
  async function sendSms() {
    if (!sms.trim() || !currentCustomer.phone) return;
    setSendingSms(true);
    setSmsStatus('');
    try {
      await posApi<unknown>(`/customers/${currentCustomer.id}/sms`, {
        method: 'POST',
        body: JSON.stringify({ message: sms.trim() })
      });
      setSms('');
      setSmsStatus('SMS sent successfully.');
    } catch (reason) {
      setSmsStatus(reason instanceof Error ? reason.message : 'SMS could not be sent.');
    } finally {
      setSendingSms(false);
    }
  }
  return (
    <div className='w-full space-y-5 p-0 sm:p-6'>
      <div className='flex w-full flex-wrap items-start justify-between gap-3 px-4 pt-4 sm:px-0 sm:pt-0'>
        <div>
          <Button variant='ghost' className='mb-2 -ml-3' onClick={() => router.back()}>
            <Icons.chevronLeft className='mr-1 h-4 w-4' />
            Customers
          </Button>
          <h1 className='text-2xl font-bold tracking-tight'>{customer.name}</h1>
          <p className='text-muted-foreground'>
            {customer.phone || 'No phone'} · {customer.email || 'No email'}
          </p>
        </div>
        <span className='bg-primary/10 text-primary rounded-full px-3 py-1 text-sm'>
          {customer.segment}
        </span>
      </div>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Lifetime spend', money(customer.total_spent)],
          ['Visits', visits],
          ['Average visit', money(average)],
          ['Open credit', money(customer.open_credit)]
        ].map(([label, value]) => (
          <Card key={label}>
            <CardContent className='space-y-2 p-5'>
              <p className='text-muted-foreground text-sm'>{label}</p>
              <p className='text-2xl font-semibold'>{value}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(280px,360px)]'>
        <Card>
          <CardHeader>
            <CardTitle>{text('Purchase history', 'Historia ya manunuzi')}</CardTitle>
            <CardDescription>
              {text(
                'Recent sales connected to this customer.',
                'Mauzo ya hivi karibuni yanayohusiana na mteja huyu.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className='divide-y'>
              {transactions.length ? (
                transactions.map((transaction) => (
                  <div
                    key={transaction.id}
                    className='flex flex-wrap items-center justify-between gap-3 py-3 text-sm'
                  >
                    <div>
                      <p className='font-medium'>{transaction.txn_number}</p>
                      <p className='text-muted-foreground'>
                        {new Date(transaction.created_at).toLocaleString()} ·{' '}
                        {transaction.payment_method}
                      </p>
                    </div>
                    <div className='text-right'>
                      <p className='font-semibold'>{money(transaction.total)}</p>
                      <p className='text-muted-foreground text-xs capitalize'>
                        {transaction.status}
                      </p>
                    </div>
                  </div>
                ))
              ) : (
                <p className='text-muted-foreground py-8 text-center'>
                  {text('No purchase history found.', 'Hakuna historia ya manunuzi iliyopatikana.')}
                </p>
              )}
            </div>
          </CardContent>
        </Card>
        <div className='space-y-4'>
          <Card>
            <CardHeader>
              <CardTitle>{text('Customer profile', 'Wasifu wa mteja')}</CardTitle>
            </CardHeader>
            <CardContent className='space-y-3 text-sm'>
              <div className='flex justify-between gap-3'>
                <span className='text-muted-foreground'>{text('Phone', 'Simu')}</span>
                <span>{customer.phone || '—'}</span>
              </div>
              <div className='flex justify-between gap-3'>
                <span className='text-muted-foreground'>{text('Email', 'Barua pepe')}</span>
                <span className='truncate'>{customer.email || '—'}</span>
              </div>
              <div className='flex justify-between gap-3'>
                <span className='text-muted-foreground'>
                  {text('Credit limit', 'Kikomo cha mkopo')}
                </span>
                <span>{customer.credit_limit == null ? '—' : money(customer.credit_limit)}</span>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>{text('Send SMS', 'Tuma SMS')}</CardTitle>
              <CardDescription>
                {text(
                  'Send a message using your SendAfrica setup.',
                  'Tuma ujumbe kwa kutumia usanidi wako wa SendAfrica.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              <Textarea
                value={sms}
                onChange={(event) => setSms(event.target.value)}
                placeholder={
                  customer.phone
                    ? 'Write a short customer message…'
                    : 'Customer has no phone number'
                }
                disabled={!customer.phone}
                maxLength={480}
              />
              <div className='flex items-center justify-between gap-2'>
                <span className='text-muted-foreground text-xs'>{sms.length}/480</span>
                <Button
                  onClick={() => void sendSms()}
                  disabled={!customer.phone || !sms.trim()}
                  isLoading={sendingSms}
                >
                  <Icons.send className='size-4' /> Send SMS
                </Button>
              </div>
              {smsStatus && (
                <p role='status' className='text-muted-foreground text-sm'>
                  {smsStatus}
                </p>
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>{text('Send receipt', 'Tuma risiti')}</CardTitle>
              <CardDescription>
                {text(
                  'Email a copy of a recent receipt.',
                  'Tuma nakala ya risiti ya hivi karibuni kwa barua pepe.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='flex gap-2'>
              <Input
                type='email'
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder='Customer email'
              />
              <Button disabled={!email}>Send</Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
