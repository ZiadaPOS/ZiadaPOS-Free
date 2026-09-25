'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, unwrapList } from '@/lib/pos-api';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

export function SupplierWorkspace() {
  const text = useText();
  const router = useRouter();
  const [suppliers, setSuppliers] = useState<Record<string, unknown>[]>([]);
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const [error, setError] = useState('');
  useEffect(() => {
    posApi<unknown>('/suppliers?page=1&page_size=100')
      .then((result) => setSuppliers(unwrapList<Record<string, unknown>>(result.data)))
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load suppliers.')
      );
  }, []);
  const number = (supplier: Record<string, unknown>, key: string) => Number(supplier[key] || 0);
  const visible = suppliers.filter((supplier) => {
    const text =
      `${supplier.name || ''} ${supplier.category || ''} ${supplier.contact || ''} ${supplier.phone || ''}`.toLowerCase();
    return (
      text.includes(query.toLowerCase()) &&
      (status === 'all' || String(supplier.status || 'active') === status)
    );
  });
  const payable = suppliers.reduce((sum, supplier) => sum + number(supplier, 'payable'), 0);
  const purchased = suppliers.reduce(
    (sum, supplier) => sum + number(supplier, 'total_purchased'),
    0
  );
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          [
            text('Total suppliers', 'Jumla ya wasambazaji'),
            suppliers.length,
            text('All supplier accounts', 'Akaunti zote za wasambazaji')
          ],
          [
            'Active suppliers',
            suppliers.filter((supplier) => supplier.status === 'active' || !supplier.status).length,
            'Ready for purchasing'
          ],
          [
            text('Total payable', 'Jumla ya kulipa'),
            money(payable),
            text('Outstanding supplier balances', 'Madeni ya wasambazaji')
          ],
          ['Purchased this year', money(purchased), 'Recorded deliveries']
        ].map(([label, value, note]) => (
          <Card key={label}>
            <CardContent className='space-y-2 p-4 sm:p-5'>
              <p className='text-muted-foreground text-xs sm:text-sm'>{label}</p>
              <p className='truncate text-lg font-semibold sm:text-2xl'>{value}</p>
              <p className='text-muted-foreground truncate text-[11px] sm:text-xs'>{note}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <CardTitle>{text('Supplier directory', 'Orodha ya wasambazaji')}</CardTitle>
          <CardDescription>
            {text(
              'Review suppliers, purchasing history, and balances.',
              'Kagua wasambazaji, historia ya manunuzi, na salio.'
            )}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='flex flex-col gap-2 sm:flex-row'>
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='Search supplier, category, or contact…'
              className='sm:flex-1'
            />
            <select
              className='bg-background h-9 rounded-md border px-3 text-sm'
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value='all'>All statuses</option>
              <option value='active'>Active</option>
              <option value='pending'>Pending</option>
              <option value='inactive'>Inactive</option>
            </select>
          </div>
          {error && (
            <p role='alert' className='text-destructive text-sm'>
              {error}
            </p>
          )}
          <div className='overflow-x-auto rounded-lg border'>
            <table className='w-full min-w-[760px] text-left text-sm'>
              <thead className='bg-muted/40'>
                <tr>
                  {['Supplier', 'Category', 'Contact', 'Status', 'Payable', 'Purchased', ''].map(
                    (heading) => (
                      <th key={heading} className='px-4 py-3 font-medium'>
                        {heading}
                      </th>
                    )
                  )}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {visible.map((supplier) => {
                  const id = String(supplier.id);
                  return (
                    <tr
                      key={id}
                      className='hover:bg-muted/30 cursor-pointer'
                      onClick={() => router.push(`/dashboard/suppliers/${id}`)}
                    >
                      <td className='px-4 py-3'>
                        <p className='font-medium'>{String(supplier.name || 'Unnamed supplier')}</p>
                        <p className='text-muted-foreground text-xs'>{id.slice(0, 8)}</p>
                      </td>
                      <td className='px-4 py-3'>{String(supplier.category || 'General')}</td>
                      <td className='px-4 py-3'>
                        {String(supplier.phone || supplier.contact || '—')}
                      </td>
                      <td className='px-4 py-3 capitalize'>
                        {String(supplier.status || 'active')}
                      </td>
                      <td className='px-4 py-3 font-medium'>
                        {number(supplier, 'payable')
                          ? money(number(supplier, 'payable'))
                          : 'Settled'}
                      </td>
                      <td className='px-4 py-3'>{money(number(supplier, 'total_purchased'))}</td>
                      <td className='px-4 py-3 text-right'>
                        <Button
                          size='sm'
                          variant='outline'
                          onClick={(event) => {
                            event.stopPropagation();
                            router.push(`/dashboard/suppliers/${id}`);
                          }}
                        >
                          View
                        </Button>
                      </td>
                    </tr>
                  );
                })}
                {!visible.length && (
                  <tr>
                    <td colSpan={7} className='text-muted-foreground px-4 py-10 text-center'>
                      No suppliers match your search.
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
