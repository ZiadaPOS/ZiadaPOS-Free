'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList, type PosCategory, type PosProduct } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useEffect, useState } from 'react';

export function CreateProduct({ onCreated }: { onCreated: () => void }) {
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [categories, setCategories] = useState<PosCategory[]>([]);
  useEffect(() => {
    posApi<unknown>('/inventory/categories')
      .then((result) => setCategories(unwrapList<PosCategory>(result.data)))
      .catch(() => setCategories([]));
  }, []);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const raw = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<PosProduct>('/inventory/products', {
        method: 'POST',
        body: JSON.stringify({
          name: raw.name,
          sku: raw.sku,
          barcode: raw.barcode,
          category_id: raw.category_id || undefined,
          selling_price: Number(raw.selling_price),
          cost_price: Number(raw.cost_price || 0),
          stock: Number(raw.stock || 0),
          reorder_point: Number(raw.reorder_point || 0),
          description: raw.description,
          is_service: raw.is_service === 'on',
          is_active: true
        })
      });
      onCreated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create product.');
    } finally {
      setPending(false);
    }
  }
  return (
    <div
      className='fixed inset-0 z-50 grid place-items-center overflow-y-auto bg-black/60 p-4'
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onCreated();
      }}
    >
      <Card
        className='my-4 w-full max-w-4xl shadow-2xl'
        onMouseDown={(event) => event.stopPropagation()}
      >
        <CardHeader className='flex flex-row items-center justify-between space-y-0'>
          <div>
            <CardTitle>New product</CardTitle>
            <CardDescription>Amounts are entered in Tanzanian shillings.</CardDescription>
          </div>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            aria-label='Close new product'
            onClick={onCreated}
          >
            <Icons.close className='h-5 w-5' />
          </Button>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className='grid gap-3 sm:grid-cols-2'>
            {[
              ['name', 'Name', 'text'],
              ['sku', 'SKU', 'text'],
              ['barcode', 'Barcode', 'text'],
              ['selling_price', 'Selling price (TZS)', 'number'],
              ['cost_price', 'Cost price (TZS)', 'number'],
              ['stock', 'Opening stock', 'number'],
              ['reorder_point', 'Low stock alert at', 'number'],
              ['description', 'Description', 'text']
            ].map(([name, label, type]) => (
              <label key={name} className='text-sm'>
                {label}
                <Input
                  name={name}
                  type={type}
                  required={name === 'name' || name === 'selling_price'}
                  min={type === 'number' ? 0 : undefined}
                />
              </label>
            ))}
            <label className='text-sm'>
              Category
              <select
                name='category_id'
                className='bg-background mt-1 h-9 w-full rounded-md border px-3'
              >
                <option value=''>Uncategorized</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
            </label>
            <label className='flex items-center gap-2 text-sm'>
              <input type='checkbox' name='is_service' /> Service (no stock tracking)
            </label>
            {error && (
              <p role='alert' className='text-destructive sm:col-span-2'>
                {error}
              </p>
            )}
            <div className='sm:col-span-2'>
              <Button isLoading={pending}>Create product</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
