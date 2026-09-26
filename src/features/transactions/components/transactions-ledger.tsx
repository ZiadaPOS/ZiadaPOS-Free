'use client';

import { useEffect, useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { money, posApi, unwrapList, type PosTransaction } from '@/lib/pos-api';
import { useText } from '@/lib/i18n';
import { useRouter } from 'next/navigation';

type TransactionKind = 'sale' | 'credit' | 'discount' | 'refund' | 'void';

function kindOf(transaction: PosTransaction): TransactionKind {
  const status = transaction.status.toLowerCase();
  if (status.includes('refund')) return 'refund';
  if (status.includes('void')) return 'void';
  if (transaction.payment_method.toLowerCase() === 'credit') return 'credit';
  if (Number(transaction.discount_amount) > 0) return 'discount';
  return 'sale';
}

function kindLabel(kind: TransactionKind): string {
  return {
    sale: 'Sale',
    credit: 'Credit sale',
    discount: 'Discounted sale',
    refund: 'Refund',
    void: 'Voided'
  }[kind];
}

export function TransactionsLedger() {
  const text = useText();
  const router = useRouter();
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const [query, setQuery] = useState('');
  const [kind, setKind] = useState<'all' | TransactionKind>('all');
  const [payment, setPayment] = useState('all');
  const [range, setRange] = useState('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function load() {
    setLoading(true);
    try {
      const result = await posApi<unknown>('/pos/transactions?page=1&page_size=100');
      setTransactions(unwrapList<PosTransaction>(result.data));
      setError('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load transactions.');
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void load();
  }, []);

  const filtered = useMemo(() => {
    const now = Date.now();
    return transactions.filter((transaction) => {
      const haystack =
        `${transaction.txn_number} ${transaction.customer_name} ${transaction.payment_method}`.toLowerCase();
      const age = now - new Date(transaction.created_at).getTime();
      return (
        haystack.includes(query.toLowerCase()) &&
        (kind === 'all' || kindOf(transaction) === kind) &&
        (payment === 'all' || transaction.payment_method === payment) &&
        (range === 'all' || (range === 'today' ? age < 86400000 : age < 604800000))
      );
    });
  }, [transactions, query, kind, payment, range]);

  const totals = useMemo(
    () => ({
      value: filtered.reduce(
        (sum, item) => sum + (kindOf(item) === 'refund' ? -item.total : item.total),
        0
      ),
      credits: filtered
        .filter((item) => kindOf(item) === 'credit')
        .reduce((sum, item) => sum + item.total, 0),
      discounts: filtered.reduce((sum, item) => sum + Number(item.discount_amount || 0), 0),
      refunds: filtered
        .filter((item) => kindOf(item) === 'refund')
        .reduce((sum, item) => sum + item.total, 0)
    }),
    [filtered]
  );
  function clear() {
    setQuery('');
    setKind('all');
    setPayment('all');
    setRange('all');
  }

  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Recorded value', money(totals.value), `${filtered.length} records`],
          [
            text('Credit sales', 'Mauzo ya mkopo'),
            money(totals.credits),
            text('Outstanding at sale', 'Deni wakati wa mauzo')
          ],
          ['Discounts', money(totals.discounts), 'Value given away'],
          ['Refunds', money(totals.refunds), 'Returned value']
        ].map(([label, value, detail]) => (
          <Card key={label} className='min-w-0'>
            <CardHeader className='p-3 pb-2 sm:p-5 sm:pb-2'>
              <CardDescription className='truncate'>{label}</CardDescription>
              <CardTitle className='truncate text-lg sm:text-2xl'>{value}</CardTitle>
            </CardHeader>
            <CardContent className='truncate p-3 pt-0 text-xs text-muted-foreground sm:p-5 sm:pt-0'>
              {detail}
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <div className='flex items-start justify-between gap-2'>
            <div className='min-w-0 flex-1'>
              <CardTitle>{text('Transaction ledger', 'Orodha ya miamala')}</CardTitle>
              <CardDescription>
                Every recorded money movement and sale adjustment in one place.
              </CardDescription>
            </div>
            <Button className='shrink-0' variant='outline' size='sm' onClick={() => void load()}>
              Refresh
            </Button>
          </div>
          <div className='grid gap-2 lg:grid-cols-[minmax(0,1fr)_auto]'>
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='Search receipt, customer, payment, or reference…'
            />
            <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
              <select
                className='bg-background h-10 rounded-md border px-3 text-sm'
                value={kind}
                onChange={(event) => setKind(event.target.value as typeof kind)}
              >
                <option value='all'>All activity</option>
                <option value='sale'>Sales</option>
                <option value='credit'>Credits</option>
                <option value='discount'>Discounts</option>
                <option value='refund'>Refunds</option>
                <option value='void'>Voids</option>
              </select>
              <select
                className='bg-background h-10 rounded-md border px-3 text-sm'
                value={payment}
                onChange={(event) => setPayment(event.target.value)}
              >
                <option value='all'>All payments</option>
                {['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit'].map((item) => (
                  <option key={item}>{item}</option>
                ))}
              </select>
              <select
                className='bg-background h-10 rounded-md border px-3 text-sm'
                value={range}
                onChange={(event) => setRange(event.target.value)}
              >
                <option value='all'>All dates</option>
                <option value='today'>Today</option>
                <option value='7d'>Last 7 days</option>
              </select>
              <Button variant='outline' onClick={clear}>
                <span className='inline-flex items-center gap-1.5'>
                  <Icons.close className='size-4 shrink-0' />
                  <span>Clear</span>
                </span>
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {error && (
            <p role='alert' className='text-destructive py-4 text-sm'>
              {error}
            </p>
          )}
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[850px] text-left text-sm'>
              <thead className='bg-muted/40'>
                <tr>
                  {[
                    'Reference',
                    'Customer',
                    'Date',
                    'Activity',
                    'Payment',
                    'Status',
                    'Amount',
                    ''
                  ].map((heading) => (
                    <th key={heading} className='px-3 py-3 font-medium'>
                      {heading}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {!loading &&
                  filtered.map((transaction) => {
                    const activity = kindOf(transaction);
                    return (
                      <tr
                        key={transaction.id}
                        className='hover:bg-muted/30 cursor-pointer'
                        onClick={() => router.push(`/dashboard/sales/${transaction.id}`)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault();
                            router.push(`/dashboard/sales/${transaction.id}`);
                          }
                        }}
                        tabIndex={0}
                        role='link'
                        aria-label={`Open transaction ${transaction.txn_number}`}
                      >
                        <td className='px-3 py-3 font-medium'>{transaction.txn_number}</td>
                        <td className='px-3 py-3'>
                          {transaction.customer_name || 'Walk-in customer'}
                        </td>
                        <td className='text-muted-foreground px-3 py-3'>
                          {new Date(transaction.created_at).toLocaleString()}
                        </td>
                        <td className='px-3 py-3'>{kindLabel(activity)}</td>
                        <td className='px-3 py-3'>{transaction.payment_method}</td>
                        <td className='px-3 py-3 capitalize'>{transaction.status}</td>
                        <td className='px-3 py-3 font-semibold'>{money(transaction.total)}</td>
                        <td className='px-3 py-3'>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={(event) => {
                              event.stopPropagation();
                              router.push(`/dashboard/sales/${transaction.id}`);
                            }}
                          >
                            View
                          </Button>
                        </td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
          {!loading && !filtered.length && (
            <p className='text-muted-foreground py-10 text-center text-sm'>
              No transactions match these filters.
            </p>
          )}
          {loading && (
            <p className='text-muted-foreground py-10 text-center text-sm'>
              {text('Loading ledger…', 'Inapakia miamala…')}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
