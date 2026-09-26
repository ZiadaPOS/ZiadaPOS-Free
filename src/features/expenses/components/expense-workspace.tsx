'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, unwrapList } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

interface ExpenseWorkspaceProps {
  showForm: boolean;
  onShowFormChange: (show: boolean) => void;
}

export function ExpenseWorkspace({ showForm, onShowFormChange }: ExpenseWorkspaceProps) {
  const text = useText();
  const [records, setRecords] = useState<Record<string, unknown>[]>([]);
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('all');
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
  const categories = [
    ...new Set(records.map((record) => String(record.category || 'Other')))
  ].sort();
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
              <p className='truncate text-lg font-semibold sm:text-2xl'>{value}</p>
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
              onChange={(event) => setCategory(event.target.value)}
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
          {message && (
            <p role='alert' className='text-destructive pt-3 text-sm'>
              {message}
            </p>
          )}
        </CardContent>
      </Card>
      {showForm && (
        <div
          className='fixed inset-0 z-50 grid place-items-center overflow-y-auto bg-black/60 p-4'
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) onShowFormChange(false);
          }}
        >
          <Card
            className='my-4 w-full max-w-2xl shadow-2xl'
            onMouseDown={(event) => event.stopPropagation()}
          >
            <CardHeader className='flex flex-row items-center justify-between space-y-0'>
              <div>
                <CardTitle>{text('New expense', 'Gharama mpya')}</CardTitle>
                <CardDescription>Record an operating expense for this store.</CardDescription>
              </div>
              <Button
                variant='ghost'
                size='icon'
                aria-label='Close new expense'
                onClick={() => onShowFormChange(false)}
              >
                <Icons.close className='h-5 w-5' />
              </Button>
            </CardHeader>
            <CardContent>
              <form onSubmit={create} className='grid gap-3 sm:grid-cols-2'>
                {[
                  ['title', 'Expense title'],
                  ['amount', 'Amount (TZS)'],
                  ['category', 'Category'],
                  ['payment_method', 'Payment method'],
                  ['payment_reference', 'Payment reference'],
                  ['notes', 'Notes']
                ].map(([name, label]) => (
                  <label key={name} className='text-sm'>
                    {label}
                    <Input
                      name={name}
                      type={name === 'amount' ? 'number' : 'text'}
                      required={name === 'title' || name === 'amount'}
                    />
                  </label>
                ))}
                <div className='sm:col-span-2'>
                  <Button isLoading={pending}>{text('Save expense', 'Hifadhi gharama')}</Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
