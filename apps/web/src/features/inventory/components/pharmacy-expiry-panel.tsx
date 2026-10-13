'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FormDialog } from '@/components/ui/form-dialog';
import { Input } from '@/components/ui/input';
import {
  posApi,
  type PosMedicineBatch,
  type PosMedicineStockMovement,
  type PosProduct
} from '@/lib/pos-api';
import { differenceInCalendarDays, parseISO } from 'date-fns';
import { Fragment, FormEvent, useEffect, useState } from 'react';

function expiryLabel(days: number) {
  if (days < 0) return { text: `Expired ${Math.abs(days)}d ago`, tone: 'text-destructive' };
  if (days === 0) return { text: 'Expires today', tone: 'text-destructive' };
  if (days <= 30) return { text: `${days} days`, tone: 'text-destructive' };
  if (days <= 60) return { text: `${days} days`, tone: 'text-amber-600 dark:text-amber-400' };
  return { text: `${days} days`, tone: 'text-muted-foreground' };
}

export function PharmacyExpiryPanel({
  batches,
  products,
  onReceived
}: {
  batches: PosMedicineBatch[];
  products: PosProduct[];
  onReceived: () => void;
}) {
  const [showReceive, setShowReceive] = useState(false);
  const [pending, setPending] = useState(false);
  const [disposalBatch, setDisposalBatch] = useState<string | null>(null);
  const [releaseBatch, setReleaseBatch] = useState<string | null>(null);
  const [historyBatch, setHistoryBatch] = useState<string | null>(null);
  const [history, setHistory] = useState<PosMedicineStockMovement[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [canDispose, setCanDispose] = useState(false);
  const [error, setError] = useState('');
  const today = new Date();
  useEffect(() => {
    try {
      const role = JSON.parse(localStorage.getItem('pos_user') || '{}').role;
      setCanDispose(role === 'owner' || role === 'admin');
    } catch {
      setCanDispose(false);
    }
  }, []);
  const lots = batches
    .map((batch) => ({
      ...batch,
      daysLeft: differenceInCalendarDays(parseISO(batch.expiry_date), today)
    }))
    .sort((a, b) => a.daysLeft - b.daysLeft);
  const stockedLots = lots.filter((batch) => batch.quantity > 0);
  const expired = stockedLots.filter((batch) => batch.daysLeft < 0).length;
  const within30 = stockedLots.filter(
    (batch) => batch.daysLeft >= 0 && batch.daysLeft <= 30
  ).length;
  const within60 = stockedLots.filter(
    (batch) => batch.daysLeft > 30 && batch.daysLeft <= 60
  ).length;

  async function toggleHistory(batchId: string) {
    if (historyBatch === batchId) {
      setHistoryBatch(null);
      return;
    }
    setHistoryBatch(batchId);
    setLoadingHistory(true);
    setError('');
    try {
      const result = await posApi<{ items: PosMedicineStockMovement[] }>(
        `/pharmacy/batches/${batchId}/movements?page=1&page_size=50`
      );
      setHistory(result.data.items || []);
    } catch (reason) {
      setHistory([]);
      setError(reason instanceof Error ? reason.message : 'Could not load batch history.');
    } finally {
      setLoadingHistory(false);
    }
  }

  async function receive(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    setPending(true);
    setError('');
    try {
      await posApi('/pharmacy/products/' + values.product_id + '/batches', {
        method: 'POST',
        body: JSON.stringify({
          batch_number: values.batch_number,
          quantity: Number(values.quantity),
          expiry_date: values.expiry_date,
          cost_price: Number(values.cost_price || 0),
          supplier_name: values.supplier_name,
          reconcile_existing_stock: values.reconcile_existing_stock === 'on'
        })
      });
      form.reset();
      setShowReceive(false);
      onReceived();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not receive this batch.');
    } finally {
      setPending(false);
    }
  }

  async function dispose(event: FormEvent<HTMLFormElement>, batch: PosMedicineBatch) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    setPending(true);
    setError('');
    try {
      await posApi(`/pharmacy/batches/${batch.id}/dispose`, {
        method: 'POST',
        body: JSON.stringify({ quantity: Number(values.quantity), reason: values.reason })
      });
      setDisposalBatch(null);
      onReceived();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not dispose this batch quantity.');
    } finally {
      setPending(false);
    }
  }

  async function release(event: FormEvent<HTMLFormElement>, batch: PosMedicineBatch) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    setPending(true);
    setError('');
    try {
      await posApi(`/pharmacy/batches/${batch.id}/release`, {
        method: 'POST',
        body: JSON.stringify({ inspection_note: values.inspection_note })
      });
      setReleaseBatch(null);
      onReceived();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not release this batch.');
    } finally {
      setPending(false);
    }
  }

  return (
    <Card>
      <CardHeader className='flex flex-row items-start justify-between gap-3'>
        <div>
          <CardTitle>Batch expiry</CardTitle>
          <CardDescription>
            Track medicine by lot and review expiry before it reaches the counter.
          </CardDescription>
        </div>
        <Button
          type='button'
          onClick={() => {
            setError('');
            setShowReceive(true);
          }}
        >
          Receive batch
        </Button>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='flex flex-wrap gap-x-6 gap-y-2 text-sm'>
          <span>
            <strong className='text-destructive'>{expired}</strong> expired lots
          </span>
          <span>
            <strong className='text-destructive'>{within30}</strong> expire within 30 days
          </span>
          <span>
            <strong className='text-amber-600 dark:text-amber-400'>{within60}</strong> expire in
            31–60 days
          </span>
        </div>
        {error && !showReceive && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}
        <FormDialog
          open={showReceive}
          onOpenChange={(open) => {
            setShowReceive(open);
            if (!open) setError('');
          }}
          title='Receive medicine batch'
          description='Record the batch number, quantity, expiry date, and supplier details.'
          size='wide'
        >
          <form onSubmit={receive} className='mt-5 grid gap-4 sm:grid-cols-2'>
            <label className='space-y-1 text-sm'>
              Medicine
              <select
                required
                name='product_id'
                className='bg-background h-11 w-full rounded-lg border border-input px-3'
              >
                <option value=''>Choose medicine</option>
                {products
                  .filter((product) => product.is_active && !product.is_service)
                  .map((product) => (
                    <option key={product.id} value={product.id}>
                      {product.name}
                    </option>
                  ))}
              </select>
            </label>
            <label className='space-y-1 text-sm'>
              Batch number
              <Input name='batch_number' required maxLength={100} />
            </label>
            <label className='space-y-1 text-sm'>
              Quantity received
              <Input name='quantity' type='number' min='1' step='1' required />
            </label>
            <label className='space-y-1 text-sm'>
              Expiry date
              <Input name='expiry_date' type='date' required />
            </label>
            <label className='space-y-1 text-sm'>
              Unit cost (TZS)
              <Input name='cost_price' type='number' min='0' step='1' />
            </label>
            <label className='space-y-1 text-sm'>
              Supplier
              <Input name='supplier_name' maxLength={200} />
            </label>
            <label className='flex items-center gap-2 text-sm sm:col-span-2'>
              <input type='checkbox' name='reconcile_existing_stock' /> Assign existing product
              stock to this batch without adding it again
            </label>
            <div className='sm:col-span-2'>
              <Button disabled={pending} className='w-full sm:w-auto'>
                {pending ? 'Saving…' : 'Save received batch'}
              </Button>
              {error && (
                <p role='alert' className='text-destructive text-sm sm:col-span-2'>
                  {error}
                </p>
              )}
            </div>
          </form>
        </FormDialog>
        {lots.length ? (
          <div className='overflow-x-auto rounded-lg border'>
            <table className='w-full min-w-[800px] text-left text-sm'>
              <thead className='bg-muted/50'>
                <tr>
                  {[
                    'Medicine',
                    'Batch',
                    'Quantity',
                    'Expiry date',
                    'Status',
                    'History',
                    ...(canDispose ? ['Action'] : [])
                  ].map((label) => (
                    <th key={label} className='px-4 py-3 font-medium'>
                      {label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {lots.map((batch) => {
                  const status =
                    batch.stock_status === 'quarantined'
                      ? {
                          text: 'Quarantined · not for sale',
                          tone: 'text-amber-600 dark:text-amber-400'
                        }
                      : batch.quantity > 0
                        ? expiryLabel(batch.daysLeft)
                        : { text: 'Depleted', tone: 'text-muted-foreground' };
                  return (
                    <Fragment key={batch.id}>
                      <tr>
                        <td className='px-4 py-3 font-medium'>
                          {batch.product_name ||
                            products.find((product) => product.id === batch.product_id)?.name ||
                            'Medicine'}
                        </td>
                        <td className='px-4 py-3'>{batch.batch_number}</td>
                        <td className='px-4 py-3 tabular-nums'>{batch.quantity}</td>
                        <td className='px-4 py-3'>
                          {parseISO(batch.expiry_date).toLocaleDateString()}
                        </td>
                        <td className={`px-4 py-3 font-medium ${status.tone}`}>{status.text}</td>
                        <td className='px-4 py-3'>
                          <Button
                            size='sm'
                            variant='ghost'
                            onClick={() => void toggleHistory(batch.id)}
                          >
                            {historyBatch === batch.id ? 'Hide history' : 'History'}
                          </Button>
                        </td>
                        {canDispose && (
                          <td className='px-4 py-3'>
                            {batch.stock_status === 'quarantined' ? (
                              <div className='flex flex-wrap gap-2'>
                                <Button
                                  size='sm'
                                  variant='outline'
                                  disabled={batch.quantity <= 0 || batch.daysLeft < 0}
                                  onClick={() => setReleaseBatch(batch.id)}
                                >
                                  Review and release
                                </Button>
                                <Button
                                  size='sm'
                                  variant='outline'
                                  disabled={batch.quantity <= 0}
                                  onClick={() => setDisposalBatch(batch.id)}
                                >
                                  Dispose
                                </Button>
                              </div>
                            ) : (
                              <Button
                                size='sm'
                                variant='outline'
                                disabled={batch.quantity <= 0}
                                onClick={() => setDisposalBatch(batch.id)}
                              >
                                Dispose
                              </Button>
                            )}
                          </td>
                        )}
                      </tr>
                      {historyBatch === batch.id && (
                        <tr>
                          <td colSpan={6 + (canDispose ? 1 : 0)} className='bg-muted/30 px-4 py-3'>
                            {loadingHistory ? (
                              <p className='text-muted-foreground text-sm'>
                                Loading batch movements…
                              </p>
                            ) : history.length ? (
                              <div className='overflow-x-auto'>
                                <table className='w-full min-w-[650px] text-left text-xs'>
                                  <thead>
                                    <tr className='text-muted-foreground'>
                                      <th className='py-2 pr-3'>Date</th>
                                      <th className='py-2 pr-3'>Movement</th>
                                      <th className='py-2 pr-3'>Quantity</th>
                                      <th className='py-2 pr-3'>Reference</th>
                                      <th className='py-2'>Reason</th>
                                    </tr>
                                  </thead>
                                  <tbody className='divide-y'>
                                    {history.map((movement) => (
                                      <tr key={movement.id}>
                                        <td className='py-2 pr-3 whitespace-nowrap'>
                                          {new Date(movement.created_at).toLocaleString()}
                                        </td>
                                        <td className='py-2 pr-3 capitalize'>
                                          {movement.movement_type.replaceAll('_', ' ')}
                                        </td>
                                        <td className='py-2 pr-3 tabular-nums'>
                                          {movement.quantity > 0
                                            ? `+${movement.quantity}`
                                            : movement.quantity}
                                        </td>
                                        <td className='py-2 pr-3'>
                                          {movement.reference_id || '—'}
                                        </td>
                                        <td className='py-2'>{movement.reason || '—'}</td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                                <p className='text-muted-foreground mt-2 text-xs'>
                                  Showing the latest 50 movements.
                                </p>
                              </div>
                            ) : (
                              <p className='text-muted-foreground text-sm'>
                                No movements recorded for this batch.
                              </p>
                            )}
                          </td>
                        </tr>
                      )}
                      {canDispose && disposalBatch === batch.id && (
                        <tr>
                          <td colSpan={6 + (canDispose ? 1 : 0)} className='bg-muted/30 px-4 py-3'>
                            <form
                              onSubmit={(event) => void dispose(event, batch)}
                              className='flex flex-wrap items-end gap-3'
                            >
                              <label className='space-y-1 text-sm'>
                                Quantity
                                <Input
                                  name='quantity'
                                  type='number'
                                  min='1'
                                  max={batch.quantity}
                                  step='1'
                                  required
                                  className='w-32'
                                />
                              </label>
                              <label className='min-w-56 flex-1 space-y-1 text-sm'>
                                Reason
                                <Input
                                  name='reason'
                                  required
                                  maxLength={500}
                                  placeholder='Expired, damaged, recalled…'
                                />
                              </label>
                              <Button disabled={pending}>
                                {pending ? 'Saving…' : 'Record disposal'}
                              </Button>
                              <Button
                                type='button'
                                variant='ghost'
                                onClick={() => setDisposalBatch(null)}
                              >
                                Cancel
                              </Button>
                            </form>
                          </td>
                        </tr>
                      )}
                      {canDispose && releaseBatch === batch.id && (
                        <tr>
                          <td colSpan={6 + (canDispose ? 1 : 0)} className='bg-muted/30 px-4 py-3'>
                            <form
                              onSubmit={(event) => void release(event, batch)}
                              className='flex flex-wrap items-end gap-3'
                            >
                              <p className='basis-full text-sm'>
                                Confirm physical inspection before releasing returned medicine for
                                sale.
                              </p>
                              <label className='min-w-56 flex-1 space-y-1 text-sm'>
                                Inspection note
                                <Input
                                  name='inspection_note'
                                  required
                                  maxLength={500}
                                  placeholder='Packaging intact, storage confirmed…'
                                />
                              </label>
                              <Button disabled={pending}>
                                {pending ? 'Saving…' : 'Release for sale'}
                              </Button>
                              <Button
                                type='button'
                                variant='ghost'
                                onClick={() => setReleaseBatch(null)}
                              >
                                Cancel
                              </Button>
                            </form>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <p className='text-muted-foreground rounded-lg border border-dashed px-4 py-8 text-center text-sm'>
            No batches recorded. Receive a batch to start tracking expiry dates.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
