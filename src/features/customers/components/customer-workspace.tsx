'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { money, type PosCustomer } from '@/lib/pos-api';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useText } from '@/lib/i18n';

export function CustomerWorkspace({ customers }: { customers: PosCustomer[] }) {
  const text = useText();
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState('All');
  const filtered = customers.filter((customer) => {
    const matchesQuery = `${customer.name} ${customer.phone || ''} ${customer.email || ''}`
      .toLowerCase()
      .includes(query.toLowerCase());
    const matchesFilter = filter === 'All' || customer.segment === filter;
    return matchesQuery && matchesFilter;
  });
  const vip = customers.filter((customer) => customer.segment === 'VIP').length;
  const withCredit = customers.filter((customer) => customer.open_credit > 0).length;
  const totalSpend = customers.reduce((sum, customer) => sum + customer.total_spent, 0);
  const averageSpend = customers.length ? totalSpend / customers.length : 0;
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          [
            text('Total customers', 'Jumla ya wateja'),
            customers.length,
            text('All active profiles', 'Wasifu wote hai')
          ],
          [
            text('VIP customers', 'Wateja VIP'),
            vip,
            text('High-value relationships', 'Wateja wa thamani kubwa')
          ],
          [
            text('With open credit', 'Wenye mkopo unaodaiwa'),
            withCredit,
            text('Needs follow-up', 'Wanahitaji ufuatiliaji')
          ],
          [
            text('Average spend', 'Wastani wa matumizi'),
            money(averageSpend),
            text('Per customer', 'Kwa mteja')
          ]
        ].map(([label, value, note]) => (
          <Card key={label}>
            <CardContent className='space-y-3 p-5'>
              <p className='text-muted-foreground text-sm'>{label}</p>
              <p className='text-2xl font-semibold tracking-tight'>{value}</p>
              <p className='text-muted-foreground text-xs'>{note}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader className='pb-3'>
          <CardTitle>{text('Customer directory', 'Orodha ya wateja')}</CardTitle>
          <CardDescription>
            Search profiles, review spending, and open customer details.
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex flex-col gap-2 lg:flex-row'>
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={text(
                'Search name, phone, or email…',
                'Tafuta jina, simu, au barua pepe…'
              )}
              className='lg:flex-1'
            />
            <div className='flex gap-2 overflow-x-auto pb-1'>
              {['All', 'New', 'Regular', 'VIP', 'Occasional'].map((option) => (
                <Button
                  key={option}
                  size='sm'
                  variant={filter === option ? 'default' : 'outline'}
                  onClick={() => setFilter(option)}
                >
                  {option}
                </Button>
              ))}
            </div>
          </div>
          <div className='overflow-x-auto rounded-lg border'>
            <table className='w-full min-w-[720px] text-sm'>
              <thead className='bg-muted/40 text-muted-foreground text-left'>
                <tr>
                  {[
                    text('Customer', 'Mteja'),
                    text('Phone', 'Simu'),
                    text('Segment', 'Kundi'),
                    text('Lifetime spend', 'Matumizi yote'),
                    text('Open credit', 'Mkopo unaodaiwa'),
                    ''
                  ].map((heading) => (
                    <th key={heading} className='px-4 py-3 font-medium'>
                      {heading}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {filtered.map((customer) => (
                  <tr
                    key={customer.id}
                    className='hover:bg-muted/30 cursor-pointer'
                    onClick={() => router.push(`/dashboard/customers/${customer.id}`)}
                  >
                    <td className='px-4 py-3'>
                      <div className='font-medium'>{customer.name}</div>
                      <div className='text-muted-foreground text-xs'>
                        {customer.email || customer.id}
                      </div>
                    </td>
                    <td className='px-4 py-3'>{customer.phone || '—'}</td>
                    <td className='px-4 py-3'>{customer.segment}</td>
                    <td className='px-4 py-3 font-medium'>{money(customer.total_spent)}</td>
                    <td className='px-4 py-3'>{money(customer.open_credit)}</td>
                    <td className='px-4 py-3 text-right'>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={(event) => {
                          event.stopPropagation();
                          router.push(`/dashboard/customers/${customer.id}`);
                        }}
                      >
                        {text('View', 'Tazama')}
                      </Button>
                    </td>
                  </tr>
                ))}
                {!filtered.length && (
                  <tr>
                    <td colSpan={6} className='text-muted-foreground px-4 py-10 text-center'>
                      {text(
                        'No customers match your search.',
                        'Hakuna wateja wanaolingana na utafutaji wako.'
                      )}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
