'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { posApi, unwrapList, type PosCategory, type PosProduct } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useEffect, useState } from 'react';

export function CreateProduct({ onCreated }: { onCreated: () => void }) {
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [imageFile, setImageFile] = useState<File | null>(null);
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
      let image: string | undefined;
      if (imageFile) {
        const upload = new FormData();
        upload.append('file', imageFile);
        upload.append('kind', 'image');
        upload.append('folder', 'products');
        const uploaded = await posApi<{ url: string }>('/uploads', {
          method: 'POST',
          body: upload
        });
        image = uploaded.data.url;
      }
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
          image,
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
        className='my-4 max-h-[min(90svh,48rem)] w-full max-w-3xl overflow-y-auto shadow-2xl'
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
          <form onSubmit={submit} className='flex flex-col gap-4'>
            <div className='grid gap-3 sm:grid-cols-2'>
              <label className='text-sm'>
                Name
                <Input name='name' required />
              </label>
              <label className='text-sm'>
                Selling price (TZS)
                <Input name='selling_price' type='number' min='0' required />
              </label>
            </div>
            <details className='group rounded-lg border bg-muted/20 p-3'>
              <summary className='cursor-pointer list-none font-medium [&::-webkit-details-marker]:hidden'>
                <span className='flex items-center justify-between'>
                  More product details
                  <Icons.chevronDown className='size-4 transition-transform group-open:rotate-180' />
                </span>
              </summary>
              <div className='mt-4 grid gap-3 sm:grid-cols-2'>
                <label className='text-sm'>
                  SKU
                  <Input name='sku' />
                </label>
                <label className='text-sm'>
                  Barcode
                  <Input name='barcode' />
                </label>
                <label className='text-sm'>
                  Cost price (TZS)
                  <Input name='cost_price' type='number' min='0' />
                </label>
                <label className='text-sm'>
                  Opening stock
                  <Input name='stock' type='number' min='0' />
                </label>
                <label className='text-sm'>
                  Low stock alert at
                  <Input name='reorder_point' type='number' min='0' />
                </label>
                <label className='text-sm'>
                  Product image
                  <Input
                    type='file'
                    accept='image/jpeg,image/png,image/webp,image/gif'
                    onChange={(event) => setImageFile(event.target.files?.[0] || null)}
                  />
                </label>
                <label className='text-sm sm:col-span-2'>
                  Description
                  <Textarea name='description' rows={3} />
                </label>
              </div>
            </details>
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
            <div className='flex justify-end'>
              <Button isLoading={pending}>Create product</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
