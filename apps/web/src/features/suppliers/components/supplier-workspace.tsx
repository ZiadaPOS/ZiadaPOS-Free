'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, unwrapList } from '@/lib/pos-api';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { FormDialog } from '@/components/ui/form-dialog';
import type { FormEvent } from 'react';

export function SupplierWorkspace() {
  const text = useText();
  const router = useRouter();
  const [suppliers, setSuppliers] = useState<Record<string, unknown>[]>([]);
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const [error, setError] = useState('');
  const [isPharmacy, setIsPharmacy] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    posApi<unknown>('/suppliers?page=1&page_size=100')
      .then((result) => setSuppliers(unwrapList<Record<string, unknown>>(result.data)))
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load suppliers.')
      );
  }, []);
  useEffect(() => {
    posApi<{ organisation?: { business_type?: string } }>('/accounts/me')
      .then((result) => setIsPharmacy(result.data.organisation?.business_type === 'pharmacy'))
      .catch(() => undefined);
  }, []);
  async function createSupplier(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    setSaving(true);
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      const result = await posApi<Record<string, unknown>>('/suppliers', {
        method: 'POST',
        body: JSON.stringify({
          name: values.name,
          category: values.category,
          rep: values.rep,
          phone: values.phone,
          email: values.email,
          address: values.address,
          payment_terms: Number(values.payment_terms),
          lead_time_days: isPharmacy ? Number(values.lead_time_days) : undefined,
          notes: values.notes
        })
      });
      setSuppliers((current) => [result.data, ...current]);
      setShowCreate(false);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create supplier.');
    } finally {
      setSaving(false);
    }
  }
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
            text('Active suppliers', 'Wasambazaji hai'),
            suppliers.filter((supplier) => supplier.status === 'active' || !supplier.status).length,
            text('Ready for purchasing', 'Tayari kwa manunuzi')
          ],
          [
            text('Total payable', 'Jumla ya kulipa'),
            money(payable),
            text('Outstanding supplier balances', 'Madeni ya wasambazaji')
          ],
          [
            text('Purchased this year', 'Manunuzi ya mwaka huu'),
            money(purchased),
            text('Recorded deliveries', 'Uwasilishaji uliorekodiwa')
          ]
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
          <div className='flex items-start justify-between gap-3'>
            <div>
              <CardTitle>{text('Supplier directory', 'Orodha ya wasambazaji')}</CardTitle>
              <CardDescription>
                {text(
                  'Review suppliers, purchasing history, and balances.',
                  'Kagua wasambazaji, historia ya manunuzi, na salio.'
                )}
              </CardDescription>
            </div>
            <Button onClick={() => setShowCreate(true)}>Add supplier</Button>
          </div>
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
                      onClick={() => router.push(`/suppliers/${id}`)}
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
                            router.push(`/suppliers/${id}`);
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
      <FormDialog
        open={showCreate}
        onOpenChange={setShowCreate}
        title='Add supplier'
        description='Save supplier contact and payment details for purchasing.'
        size='wide'
      >
        <form onSubmit={createSupplier} className='mt-5 grid gap-4 sm:grid-cols-2'>
          {error && (
            <p role='alert' className='text-destructive text-sm sm:col-span-2'>
              {error}
            </p>
          )}
          <label className='grid gap-1.5 text-sm font-medium'>
            Name
            <Input name='name' required maxLength={200} />
          </label>
          <label className='grid gap-1.5 text-sm font-medium'>
            Category
            <Input name='category' maxLength={100} />
          </label>
          <label className='grid gap-1.5 text-sm font-medium'>
            Contact person
            <Input name='rep' maxLength={200} />
          </label>
          <label className='grid gap-1.5 text-sm font-medium'>
            Phone
            <Input name='phone' type='tel' maxLength={50} />
          </label>
          <label className='grid gap-1.5 text-sm font-medium'>
            Email
            <Input name='email' type='email' maxLength={150} />
          </label>
          <label className='grid gap-1.5 text-sm font-medium'>
            Payment terms
            <select
              name='payment_terms'
              defaultValue='30'
              className='bg-background h-11 w-full rounded-lg border border-input px-3 text-sm font-normal'
            >
              <option value='30'>30 days</option>
              <option value='60'>60 days</option>
              <option value='90'>90 days</option>
            </select>
          </label>
          {isPharmacy && (
            <label className='grid gap-1.5 text-sm font-medium'>
              Delivery lead time (days)
              <Input
                name='lead_time_days'
                type='number'
                min='1'
                max='365'
                defaultValue='7'
                required
              />
            </label>
          )}
          <label className='grid gap-1.5 text-sm font-medium sm:col-span-2'>
            Address
            <Input name='address' maxLength={500} />
          </label>
          <label className='grid gap-1.5 text-sm font-medium sm:col-span-2'>
            Notes
            <Input name='notes' maxLength={1000} />
          </label>
          <Button className='sm:col-span-2' disabled={saving}>
            {saving ? 'Saving…' : 'Save supplier'}
          </Button>
        </form>
      </FormDialog>
    </div>
  );
}
