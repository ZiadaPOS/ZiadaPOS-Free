'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { money, posApi, unwrapList, type PosProduct, type PosTransaction } from '@/lib/pos-api';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { useText } from '@/lib/i18n';
import { MedicineProfileEditor } from '@/features/pharmacy/components/medicine-profile-editor';
import { WholesalePricingEditor } from '@/features/inventory/components/wholesale-pricing-editor';

type Adjustment = {
  id: string;
  qty_change: number;
  reason: string;
  note?: string;
  created_at: string;
};

export default function InventoryProductDetailPage() {
  const text = useText();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [product, setProduct] = useState<PosProduct | null>(null);
  const [adjustments, setAdjustments] = useState<Adjustment[]>([]);
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const [error, setError] = useState('');
  const [isPharmacy, setIsPharmacy] = useState(false);
  const [businessType, setBusinessType] = useState('retail');
  useEffect(() => {
    void Promise.all([
      posApi<PosProduct>(`/inventory/products/${params.id}`),
      posApi<unknown>(`/inventory/products/${params.id}/adjustments?limit=20`),
      posApi<unknown>('/pos/transactions?page=1&page_size=100'),
      posApi<{ organisation?: { business_type?: string } }>('/accounts/me')
    ])
      .then(([productResult, adjustmentResult, transactionResult, profile]) => {
        setProduct(productResult.data);
        const type = profile.data.organisation?.business_type || 'retail';
        setBusinessType(type);
        setIsPharmacy(type === 'pharmacy');
        setAdjustments(unwrapList<Adjustment>(adjustmentResult.data));
        setTransactions(unwrapList<PosTransaction>(transactionResult.data));
      })
      .catch((reason) =>
        setError(
          reason instanceof Error
            ? reason.message
            : text('Could not load product details.', 'Imeshindikana kupakia maelezo ya bidhaa.')
        )
      );
  }, [params.id]);
  const sold = useMemo(
    () =>
      transactions
        .flatMap((transaction) => transaction.lines || [])
        .filter((line) => line.product_id === params.id || line.product_name === product?.name)
        .reduce((sum, line) => sum + line.qty, 0),
    [params.id, product, transactions]
  );
  const revenue = useMemo(
    () =>
      transactions
        .flatMap((transaction) => transaction.lines || [])
        .filter((line) => line.product_id === params.id || line.product_name === product?.name)
        .reduce((sum, line) => sum + line.total, 0),
    [params.id, product, transactions]
  );
  if (error)
    return (
      <div className='p-6'>
        <p role='alert' className='text-destructive'>
          {error}
        </p>
        <Button className='mt-4' onClick={() => router.back()}>
          Back to inventory
        </Button>
      </div>
    );
  if (!product) return <div className='text-muted-foreground p-6'>Loading product…</div>;
  return (
    <div className='w-full space-y-5 p-0 sm:p-6'>
      <div className='flex w-full flex-wrap items-start justify-between gap-3 px-4 pt-4 sm:px-0 sm:pt-0'>
        <div>
          <Button variant='ghost' className='mb-2 -ml-3' onClick={() => router.back()}>
            <Icons.chevronLeft className='mr-1 h-4 w-4' />
            Inventory
          </Button>
          <h1 className='text-2xl font-bold tracking-tight'>{product.name}</h1>
          <p className='text-muted-foreground'>
            {product.sku || product.barcode || 'No SKU'} ·{' '}
            {product.category?.name || 'Uncategorized'}
          </p>
        </div>
        <span
          className={
            product.is_active
              ? 'bg-primary/10 text-primary rounded-full px-3 py-1 text-sm'
              : 'bg-muted text-muted-foreground rounded-full px-3 py-1 text-sm'
          }
        >
          {product.is_active ? 'Active' : 'Inactive'}
        </span>
      </div>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Current stock', `${product.stock}`, 'Units available'],
          ['Units sold', `${sold}`, 'From completed sales'],
          ['Sales revenue', money(revenue), 'Product sales'],
          ['Stock value', money(product.stock * product.cost_price), 'At cost']
        ].map(([label, value, detail]) => (
          <Card key={label} className='min-w-0'>
            <CardHeader className='p-3 pb-2 sm:p-6 sm:pb-2'>
              <CardDescription className='truncate text-xs sm:text-sm'>{label}</CardDescription>
              <CardTitle className='truncate text-lg sm:text-2xl'>{value}</CardTitle>
            </CardHeader>
            <CardContent className='truncate p-3 pt-0 text-[11px] text-muted-foreground sm:p-6 sm:pt-0 sm:text-xs'>
              {detail}
            </CardContent>
          </Card>
        ))}
      </div>
      {isPharmacy && <MedicineProfileEditor productId={product.id} />}
      {(businessType === 'wholesale' || businessType === 'retail_wholesale') && (
        <WholesalePricingEditor product={product} onSaved={setProduct} />
      )}
      <div className='grid w-full gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(360px,480px)]'>
        <Card className='w-full'>
          <CardHeader>
            <CardTitle>Product information</CardTitle>
            <CardDescription>
              Pricing, stock thresholds, and identifiers for this product.
            </CardDescription>
          </CardHeader>
          <CardContent className='grid gap-4 sm:grid-cols-2'>
            <div>
              <p className='text-muted-foreground text-xs'>Selling price</p>
              <p className='font-semibold'>{money(product.selling_price)}</p>
            </div>
            <div>
              <p className='text-muted-foreground text-xs'>Cost price</p>
              <p className='font-semibold'>{money(product.cost_price)}</p>
            </div>
            <div>
              <p className='text-muted-foreground text-xs'>Reorder point</p>
              <p className='font-semibold'>{product.reorder_point}</p>
            </div>
            <div>
              <p className='text-muted-foreground text-xs'>Barcode</p>
              <p className='font-semibold'>{product.barcode || '—'}</p>
            </div>
            <div className='sm:col-span-2'>
              <p className='text-muted-foreground text-xs'>Description</p>
              <p>{product.description || 'No description added.'}</p>
            </div>
          </CardContent>
        </Card>
        <Card className='w-full'>
          <CardHeader>
            <CardTitle>Stock activity</CardTitle>
            <CardDescription>Recent adjustments and reasons.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='divide-y'>
              {adjustments.length ? (
                adjustments.map((adjustment) => (
                  <div
                    key={adjustment.id}
                    className='flex items-center justify-between gap-3 py-3 text-sm'
                  >
                    <div>
                      <p className='font-medium capitalize'>{adjustment.reason}</p>
                      <p className='text-muted-foreground text-xs'>
                        {new Date(adjustment.created_at).toLocaleString()} ·{' '}
                        {adjustment.note || 'No note'}
                      </p>
                    </div>
                    <span
                      className={
                        adjustment.qty_change >= 0
                          ? 'text-emerald-500 font-semibold'
                          : 'text-destructive font-semibold'
                      }
                    >
                      {adjustment.qty_change >= 0 ? '+' : ''}
                      {adjustment.qty_change}
                    </span>
                  </div>
                ))
              ) : (
                <p className='text-muted-foreground py-5 text-center text-sm'>
                  No stock adjustments recorded.
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
