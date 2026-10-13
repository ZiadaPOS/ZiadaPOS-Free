'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { FormDialog } from '@/components/ui/form-dialog';
import { posApi, money, unwrapList } from '@/lib/pos-api';
import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

interface ExpenseWorkspaceProps {
  showForm: boolean;
  onShowFormChange: (show: boolean) => void;
  categories: string[];
  category: string;
  onCategoryChange: (category: string) => void;
}

export function ExpenseWorkspace({
  showForm,
  onShowFormChange,
  categories,
  category,
  onCategoryChange
}: ExpenseWorkspaceProps) {
  const text = useText();
  const [records, setRecords] = useState<Record<string, unknown>[]>([]);
  const [query, setQuery] = useState('');
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState('');
  const load = useCallback(() => {
    void posApi<unknown>('/expenses?page=1&page_size=100')
      .then((result) => setRecords(unwrapList<Record<string, unknown>>(result.data)))
      .catch((reason) =>
        setMessage(reason instanceof Error ? reason.message : 'Could not load expenses.')
      );
  }, []);
  useEffect(() => {
    load();
  }, [load]);
  const amount = (record: Record<string, unknown>) => Number(record.amount || 0);
  const filtered = records.filter(
    (record) =>
      `${record.title || record.name || ''} ${record.category || ''} ${record.notes || record.note || ''}`
        .toLowerCase()
        .includes(query.toLowerCase()) &&
      (category === 'all' || String(record.category || 'Other') === category)
  );
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage('');
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<unknown>('/expenses', {
        method: 'POST',
        body: JSON.stringify({ ...values, amount: Number(values.amount) })
      });
      onShowFormChange(false);
      load();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not save expense.');
    } finally {
      setPending(false);
    }
  }
  const total = records.reduce((sum, record) => sum + amount(record), 0);
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Total expenses', money(total), 'All recorded costs'],
          ['Transactions', records.length, 'Expense records'],
          ['Average expense', money(records.length ? total / records.length : 0), 'Per record'],
          ['This month', money(total), 'Current loaded period']
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
          <div className='flex flex-col gap-2 pt-2 sm:flex-row'>
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={text(
                'Search expense, category, or note…',
                'Tafuta gharama, kundi, au maelezo…'
              )}
              className='sm:flex-1'
            />
            <select
              className='bg-background h-9 rounded-md border px-3 text-sm'
              value={category}
              onChange={(event) => onCategoryChange(event.target.value)}
            >
              <option value='all'>{text('All categories', 'Makundi yote')}</option>
              {categories.map((item) => (
                <option key={item}>{item}</option>
              ))}
            </select>
          </div>
        </CardHeader>
        <CardContent>
          <div className='overflow-x-auto rounded-lg border'>
            <table className='w-full min-w-[700px] text-left text-sm'>
              <thead className='bg-muted/40'>
                <tr>
                  {['Amount', 'Category', 'Notes', 'Payment method', 'Reference', 'Date'].map(
                    (heading) => (
                      <th key={heading} className='px-4 py-3 font-medium'>
                        {heading}
                      </th>
                    )
                  )}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {filtered.map((record) => (
                  <tr key={String(record.id)} className='hover:bg-muted/30'>
                    <td className='px-4 py-3 font-semibold'>{money(amount(record))}</td>
                    <td className='px-4 py-3'>{String(record.category || 'Other')}</td>
                    <td className='px-4 py-3'>{String(record.notes || record.note || '—')}</td>
                    <td className='px-4 py-3'>{String(record.payment_method || 'Cash')}</td>
                    <td className='px-4 py-3'>
                      {String(record.payment_reference || record.reference || '—')}
                    </td>
                    <td className='px-4 py-3'>
                      {String(record.expense_date || record.created_at || '—')}
                    </td>
                  </tr>
                ))}
                {!filtered.length && (
                  <tr>
                    <td colSpan={6} className='text-muted-foreground px-4 py-10 text-center'>
                      No expenses match your filters.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {message && !showForm && (
            <p role='alert' className='text-destructive pt-3 text-sm'>
              {message}
            </p>
          )}
        </CardContent>
      </Card>
      <FormDialog
        open={showForm}
        onOpenChange={(open) => {
          onShowFormChange(open);
          if (!open) setMessage('');
        }}
        title={text('New expense', 'Gharama mpya')}
        description='Record an operating expense for this store.'
        size='wide'
      >
        <form onSubmit={create} className='mt-5 grid gap-4 sm:grid-cols-2'>
          {[
            ['title', 'Expense title'],
            ['amount', 'Amount (TZS)'],
            ['category', 'Category'],
            ['payment_method', 'Payment method'],
            ['payment_reference', 'Payment reference'],
            ['notes', 'Notes']
          ].map(([name, label]) =>
            name === 'category' ? (
              <label key={name} className='grid gap-2 text-sm font-semibold'>
                {label}
                <select
                  name={name}
                  defaultValue=''
                  className='bg-background h-11 w-full rounded-lg border border-input px-3 font-normal'
                  required
                >
                  <option value='' disabled>
                    Select category
                  </option>
                  {categories.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </select>
              </label>
            ) : (
              <label key={name} className='grid gap-2 text-sm font-semibold'>
                {label}
                <Input
                  name={name}
                  type={name === 'amount' ? 'number' : 'text'}
                  required={name === 'title' || name === 'amount'}
                />
              </label>
            )
          )}
          {message && (
            <p
              role='alert'
              className='rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive sm:col-span-2'
            >
              {message}
            </p>
          )}
          <Button isLoading={pending} className='h-12 w-full sm:col-span-2'>
            {text('Save expense', 'Hifadhi gharama')}
          </Button>
        </form>
      </FormDialog>
    </div>
  );
}
