'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, type PosProduct } from '@/lib/pos-api';
import { useRouter } from 'next/navigation';
import { FormEvent, useState } from 'react';
import { useText } from '@/lib/i18n';

export function InventoryTable({
  products,
  isPharmacy,
  onUpdated
}: {
  products: PosProduct[];
  isPharmacy: boolean;
  onUpdated: () => void;
}) {
  const text = useText();
  const router = useRouter();
  const [error, setError] = useState('');
  const [adjustId, setAdjustId] = useState('');
  async function adjust(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<unknown>(`/inventory/products/${adjustId}/adjust`, {
        method: 'POST',
        body: JSON.stringify({
          qty_change: Number(values.qty_change),
          reason: values.reason,
          note: values.note
        })
      });
      setAdjustId('');
      setError('');
      onUpdated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Stock adjustment failed.');
    }
  }
  async function updatePrice(product: PosProduct) {
    const value = window.prompt(
      `New selling price for ${product.name}`,
      String(product.selling_price)
    );
    if (value === null) return;
    const amount = Number(value);
    if (!Number.isFinite(amount) || amount < 0) {
      setError('Enter a valid selling price.');
      return;
    }
    try {
      await posApi<unknown>(`/inventory/products/${product.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ selling_price: amount })
      });
      onUpdated();
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : text('Product update failed.', 'Imeshindikana kusasisha bidhaa.')
      );
    }
  }
  async function deactivate(product: PosProduct) {
    if (!window.confirm(`Deactivate ${product.name}?`)) return;
    try {
      await posApi<unknown>(`/inventory/products/${product.id}`, { method: 'DELETE' });
      onUpdated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not deactivate product.');
    }
  }
  return (
    <>
      <div className='overflow-x-auto rounded-lg border'>
        <table className='w-full text-left text-sm'>
          <thead className='bg-muted/50'>
            <tr>
              {[
                '',
                text('Product', 'Bidhaa'),
                text('SKU / barcode', 'SKU / barcode'),
                text('Price', 'Bei'),
                text('Cost', 'Gharama'),
                isPharmacy ? 'Sellable / batch stock' : text('Stock', 'Stoo'),
                text('Status', 'Hali'),
                text('Actions', 'Vitendo')
              ].map((label) => (
                <th key={label} className='px-4 py-3 font-medium'>
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className='divide-y'>
            {products.length ? (
              products.map((product) => (
                <tr
                  key={product.id}
                  className='cursor-pointer hover:bg-muted/30'
                  onClick={() => router.push(`/inventory/${product.id}`)}
                >
                  <td className='px-4 py-3'>
                    <div className='bg-muted/50 flex h-10 w-10 items-center justify-center overflow-hidden rounded-md font-semibold text-primary'>
                      {product.image ? (
                        <img src={product.image} alt='' className='h-full w-full object-cover' />
                      ) : (
                        product.name.slice(0, 1).toUpperCase()
                      )}
                    </div>
                  </td>
                  <td className='px-4 py-3 font-medium'>{product.name}</td>
                  <td className='px-4 py-3'>{product.sku || product.barcode || '—'}</td>
                  <td className='px-4 py-3'>{money(product.selling_price)}</td>
                  <td className='px-4 py-3'>{money(product.cost_price)}</td>
                  <td className='px-4 py-3'>
                    {isPharmacy ? (
                      <div className='space-y-1'>
                        <div>{product.sellable_stock || 0} sellable</div>
                        {!!product.expired_stock && (
                          <div className='text-destructive text-xs'>
                            {product.expired_stock} expired
                          </div>
                        )}
                        {!!product.untracked_stock && (
                          <div className='text-muted-foreground text-xs'>
                            {product.untracked_stock} unassigned
                          </div>
                        )}
                        {product.next_expiry_date && (
                          <div className='text-muted-foreground text-xs'>
                            Next expiry{' '}
                            {new Date(
                              `${product.next_expiry_date.slice(0, 10)}T00:00:00`
                            ).toLocaleDateString()}
                          </div>
                        )}
                      </div>
                    ) : (
                      product.stock
                    )}
                    {product.low_stock && <span className='text-destructive'>Low</span>}
                  </td>
                  <td className='px-4 py-3'>{product.is_active ? 'Active' : 'Inactive'}</td>
                  <td className='px-4 py-3'>
                    <div className='flex gap-1'>
                      <Button size='sm' variant='outline' onClick={() => setAdjustId(product.id)}>
                        {text('Adjust stock', 'Rekebisha stoo')}
                      </Button>
                      <Button size='sm' variant='ghost' onClick={() => void updatePrice(product)}>
                        Price
                      </Button>
                      <Button size='sm' variant='ghost' onClick={() => void deactivate(product)}>
                        Deactivate
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={8} className='text-muted-foreground px-4 py-8 text-center'>
                  No products found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {error && (
        <p role='alert' className='text-destructive text-sm'>
          {error}
        </p>
      )}
      {adjustId && (
        <div className='fixed inset-0 z-50 grid place-items-center bg-black/40 p-4'>
          <Card className='w-full max-w-md'>
            <CardHeader>
              <CardTitle>{text('Adjust stock', 'Rekebisha stoo')}</CardTitle>
              <CardDescription>
                {products.find((product) => product.id === adjustId)?.name}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={adjust} className='space-y-3'>
                <label className='block text-sm'>
                  {text(
                    'Quantity change (use a negative number to remove stock)',
                    'Mabadiliko ya kiasi (tumia namba hasi kupunguza stoo)'
                  )}
                  <Input name='qty_change' type='number' required />
                </label>
                <label className='block text-sm'>
                  Reason
                  <select
                    className='bg-background mt-1 h-9 w-full rounded-md border px-3'
                    name='reason'
                  >
                    <option value='restock'>{text('Restock', 'Ongeza stoo')}</option>
                    <option value='damage'>Damage</option>
                    <option value='correction'>Correction</option>
                    <option value='return'>Return</option>
                  </select>
                </label>
                <label className='block text-sm'>
                  Note
                  <Input name='note' />
                </label>
                <div className='flex justify-end gap-2'>
                  <Button type='button' variant='outline' onClick={() => setAdjustId('')}>
                    {text('Cancel', 'Ghairi')}
                  </Button>
                  <Button type='submit'>{text('Save adjustment', 'Hifadhi marekebisho')}</Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      )}
    </>
  );
}
