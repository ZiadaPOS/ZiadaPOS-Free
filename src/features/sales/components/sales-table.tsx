'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, type PosTransaction } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

export function SalesTable({
  transactions,
  onUpdated
}: {
  transactions: PosTransaction[];
  onUpdated: () => void;
}) {
  const router = useRouter();
  const [message, setMessage] = useState('');
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const [payment, setPayment] = useState('all');
  const [range, setRange] = useState('all');
  const now = Date.now();
  const filtered = transactions.filter((transaction) => {
    const haystack =
      `${transaction.txn_number} ${transaction.customer_name || ''} ${transaction.payment_method}`.toLowerCase();
    const matchesQuery = haystack.includes(query.toLowerCase());
    const matchesStatus = status === 'all' || transaction.status.toLowerCase() === status;
    const matchesPayment =
      payment === 'all' || transaction.payment_method.toLowerCase() === payment.toLowerCase();
    const age = now - new Date(transaction.created_at).getTime();
    const matchesRange = range === 'all' || (range === 'today' ? age < 86400000 : age < 604800000);
    return matchesQuery && matchesStatus && matchesPayment && matchesRange;
  });
  const revenue = filtered.reduce((sum, transaction) => sum + Number(transaction.total || 0), 0);
  const paid = filtered.filter((transaction) => transaction.status.toLowerCase() === 'paid').length;
  const average = filtered.length ? revenue / filtered.length : 0;
  async function action(id: string, actionName: 'refund' | 'void') {
    const reason = window.prompt(`Reason for ${actionName}:`);
    if (reason === null) return;
    try {
      await posApi<unknown>(`/pos/transactions/${id}/${actionName}`, {
        method: 'POST',
        body: JSON.stringify({ reason })
      });
      setMessage(`Transaction ${actionName} completed.`);
      onUpdated();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : `Could not ${actionName} transaction.`);
    }
  }
  function clearFilters() {
    setQuery('');
    setStatus('all');
    setPayment('all');
    setRange('all');
  }
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Sales revenue', money(revenue), 'From matching sales'],
          ['Transactions', String(filtered.length), `${paid} paid`],
          ['Average sale', money(average), 'Per transaction'],
          [
            'Refund / void ready',
            String(filtered.filter((item) => item.status === 'paid').length),
            'Paid sales available'
          ]
        ].map(([label, value, detail]) => (
          <Card key={label} className='min-w-0'>
            <CardHeader className='p-3 pb-2 sm:p-6 sm:pb-2'>
              <CardDescription className='truncate text-xs sm:text-sm'>{label}</CardDescription>
              <CardTitle className='truncate text-lg sm:text-2xl'>{value}</CardTitle>
            </CardHeader>
            <CardContent className='truncate p-3 pt-0 text-[11px] text-muted-foreground sm:p-6 sm:pt-0 sm:text-xs'>
              {detail}
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div>
              <CardTitle>Sales history</CardTitle>
              <CardDescription>
                Search receipts, review payment details, and manage completed sales.
              </CardDescription>
            </div>
            <Button variant='outline' className='hidden sm:inline-flex' onClick={clearFilters}>
              Clear filters
            </Button>
          </div>
          <div className='sales-filter-toolbar grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(520px,auto)]'>
            <Input
              className='h-11'
              aria-label='Search sales'
              placeholder='Search receipt, customer, or payment…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <div className='sales-filter-row'>
              <select
                aria-label='Filter by status'
                className='bg-background h-11 rounded-md border px-3 text-sm'
                value={status}
                onChange={(event) => setStatus(event.target.value)}
              >
                <option value='all'>All statuses</option>
                <option value='paid'>Paid</option>
                <option value='refunded'>Refunded</option>
                <option value='voided'>Voided</option>
              </select>
              <select
                aria-label='Filter by payment'
                className='bg-background h-11 rounded-md border px-3 text-sm'
                value={payment}
                onChange={(event) => setPayment(event.target.value)}
              >
                <option value='all'>All payments</option>
                {['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit'].map((method) => (
                  <option key={method}>{method}</option>
                ))}
              </select>
              <select
                aria-label='Filter by date'
                className='bg-background h-11 rounded-md border px-3 text-sm'
                value={range}
                onChange={(event) => setRange(event.target.value)}
              >
                <option value='all'>All dates</option>
                <option value='today'>Today</option>
                <option value='7d'>Last 7 days</option>
              </select>
              <Button
                type='button'
                size='icon'
                variant='outline'
                className='sm:hidden'
                aria-label='Clear filters'
                title='Clear filters'
                onClick={clearFilters}
              >
                <Icons.close className='h-4 w-4' />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[760px] text-left text-sm'>
              <thead className='bg-muted/40'>
                <tr>
                  {['Receipt', 'Customer', 'Date', 'Payment', 'Status', 'Total', 'Actions'].map(
                    (heading) => (
                      <th key={heading} className='px-3 py-3 font-medium'>
                        {heading}
                      </th>
                    )
                  )}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {filtered.length ? (
                  filtered.map((transaction) => (
                    <tr key={transaction.id} className='hover:bg-muted/30'>
                      <td className='px-3 py-3 font-medium'>{transaction.txn_number}</td>
                      <td className='px-3 py-3'>
                        {transaction.customer_name || 'Walk-in customer'}
                      </td>
                      <td className='text-muted-foreground px-3 py-3'>
                        {new Date(transaction.created_at).toLocaleString()}
                      </td>
                      <td className='px-3 py-3'>{transaction.payment_method}</td>
                      <td className='px-3 py-3 capitalize'>{transaction.status}</td>
                      <td className='px-3 py-3 font-semibold'>{money(transaction.total)}</td>
                      <td className='px-3 py-3'>
                        <div className='flex gap-1'>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => router.push(`/dashboard/sales/${transaction.id}`)}
                          >
                            View
                          </Button>
                          {transaction.status === 'paid' && (
                            <>
                              <Button
                                size='sm'
                                variant='ghost'
                                onClick={() => void action(transaction.id, 'refund')}
                              >
                                Refund
                              </Button>
                              <Button
                                size='sm'
                                variant='ghost'
                                onClick={() => void action(transaction.id, 'void')}
                              >
                                Void
                              </Button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={7} className='text-muted-foreground px-3 py-10 text-center'>
                      No sales match these filters.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {message && (
            <p role='status' className='text-muted-foreground pt-3 text-sm'>
              {message}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
