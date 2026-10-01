'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, type PosCustomer, type PosCreditTab } from '@/lib/pos-api';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useText } from '@/lib/i18n';

export function CreditTable({
  tabs,
  customers,
  onUpdated
}: {
  tabs: PosCreditTab[];
  customers: PosCustomer[];
  onUpdated: () => void;
}) {
  const text = useText();
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [message, setMessage] = useState('');
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const router = useRouter();
  const filteredTabs = tabs.filter((tab) => {
    const customer = customers.find((item) => item.id === tab.customer_id);
    const matchesQuery = `${customer?.name || ''} ${customer?.phone || ''}`
      .toLowerCase()
      .includes(query.toLowerCase());
    const matchesStatus =
      status === 'all' || (status === 'overdue' ? tab.is_overdue : tab.status === status);
    return matchesQuery && matchesStatus;
  });
  const outstanding = tabs.reduce((sum, tab) => sum + tab.balance, 0);
  const overdue = tabs.filter((tab) => tab.is_overdue).reduce((sum, tab) => sum + tab.balance, 0);
  const paid = tabs.reduce((sum, tab) => sum + tab.amount_paid, 0);
  async function pay(tab: PosCreditTab) {
    const amount = Number(amounts[tab.customer_id]);
    if (!amount || amount <= 0) {
      setMessage('Enter a payment amount greater than zero.');
      return;
    }
    try {
      await posApi<unknown>(`/credits/customers/${tab.customer_id}/payments`, {
        method: 'POST',
        body: JSON.stringify({ amount, method: 'Cash' })
      });
      setMessage('Payment recorded.');
      setAmounts({});
      onUpdated();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Payment could not be recorded.');
    }
  }
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Outstanding', money(outstanding), 'Current customer balances'],
          ['Overdue', money(overdue), 'Requires follow-up'],
          ['Collected', money(paid), 'Payments recorded'],
          ['Open tabs', tabs.length, 'Active credit accounts']
        ].map(([label, value, note]) => (
          <Card key={label}>
            <CardContent className='space-y-2 p-4 sm:p-5'>
              <p className='text-muted-foreground text-xs sm:text-sm'>{label}</p>
              <p className='truncate text-2xl font-semibold sm:text-3xl'>{value}</p>
              <p className='text-muted-foreground truncate text-[11px] sm:text-xs'>{note}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <CardTitle>{text('Outstanding tabs', 'Mikopo ambayo haijalipwa')}</CardTitle>
          <CardDescription>
            {text(
              'Track balances, overdue accounts, and record payments.',
              'Fuatilia salio, akaunti zilizochelewa, na rekodi malipo.'
            )}
          </CardDescription>
          <div className='flex flex-col gap-2 pt-2 sm:flex-row'>
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={text('Search customer or phone…', 'Tafuta mteja au simu…')}
              className='sm:flex-1'
            />
            <select
              className='bg-background h-9 rounded-md border px-3 text-sm'
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value='all'>{text('All statuses', 'Hali zote')}</option>
              <option value='open'>{text('Open', 'Wazi')}</option>
              <option value='overdue'>{text('Overdue', 'Imechelewa')}</option>
              <option value='paid'>{text('Paid', 'Imelipwa')}</option>
            </select>
          </div>
        </CardHeader>
        <CardContent>
          <div className='overflow-x-auto'>
            <table className='w-full text-left text-sm'>
              <thead>
                <tr className='border-b'>
                  {['Customer', 'Original', 'Paid', 'Balance', 'Status', 'Record payment'].map(
                    (item) => (
                      <th className='px-2 py-3' key={item}>
                        {item}
                      </th>
                    )
                  )}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {filteredTabs.map((tab) => (
                  <tr
                    key={tab.id}
                    className='hover:bg-muted/30 cursor-pointer'
                    onClick={() => router.push(`/credits/${tab.customer_id}`)}
                  >
                    <td className='px-2 py-3'>
                      {customers.find((customer) => customer.id === tab.customer_id)?.name ||
                        tab.customer_id.slice(0, 8)}
                    </td>
                    <td className='px-2 py-3'>{money(tab.amount)}</td>
                    <td className='px-2 py-3'>{money(tab.amount_paid)}</td>
                    <td className='px-2 py-3 font-semibold'>{money(tab.balance)}</td>
                    <td className='px-2 py-3'>
                      {tab.status}
                      {tab.is_overdue ? ' · overdue' : ''}
                    </td>
                    <td className='px-2 py-3'>
                      <div className='flex gap-2'>
                        <Input
                          onClick={(event) => event.stopPropagation()}
                          className='w-28'
                          type='number'
                          min='1'
                          placeholder='Amount'
                          value={amounts[tab.customer_id] || ''}
                          onChange={(event) =>
                            setAmounts((current) => ({
                              ...current,
                              [tab.customer_id]: event.target.value
                            }))
                          }
                        />
                        <Button
                          size='sm'
                          onClick={(event) => {
                            event.stopPropagation();
                            void pay(tab);
                          }}
                        >
                          Record
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!filteredTabs.length && (
                  <tr>
                    <td colSpan={6} className='text-muted-foreground px-2 py-8 text-center'>
                      No credit tabs found.
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
