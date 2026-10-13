'use client';

import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, type PosProduct } from '@/lib/pos-api';

export function WholesalePricingEditor({
  product,
  onSaved
}: {
  product: PosProduct;
  onSaved: (product: PosProduct) => void;
}) {
  const [price, setPrice] = useState(String(product.wholesale_price ?? product.selling_price));
  const [minimum, setMinimum] = useState(String(product.wholesale_minimum || 1));
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  useEffect(() => {
    setPrice(String(product.wholesale_price ?? product.selling_price));
    setMinimum(String(product.wholesale_minimum || 1));
  }, [product]);
  async function save() {
    setBusy(true);
    setMessage('');
    try {
      const result = await posApi<PosProduct>(`/inventory/products/${product.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ wholesale_price: Number(price), wholesale_minimum: Number(minimum) })
      });
      onSaved(result.data);
      setMessage('Wholesale pricing saved.');
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not save wholesale pricing.');
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Wholesale pricing</CardTitle>
        <CardDescription>
          Wholesale checkout uses this price when the basket meets the minimum quantity.
        </CardDescription>
      </CardHeader>
      <CardContent className='grid gap-3 sm:grid-cols-[1fr_1fr_auto]'>
        <label className='text-sm'>
          Unit price (TZS)
          <Input
            type='number'
            min='1'
            value={price}
            onChange={(event) => setPrice(event.target.value)}
          />
        </label>
        <label className='text-sm'>
          Minimum units
          <Input
            type='number'
            min='1'
            value={minimum}
            onChange={(event) => setMinimum(event.target.value)}
          />
        </label>
        <Button
          className='self-end'
          disabled={busy || Number(price) < 1 || Number(minimum) < 1}
          onClick={() => void save()}
        >
          {busy ? 'Saving…' : 'Save pricing'}
        </Button>
        {message && (
          <p role='status' className='text-muted-foreground text-sm sm:col-span-3'>
            {message}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
