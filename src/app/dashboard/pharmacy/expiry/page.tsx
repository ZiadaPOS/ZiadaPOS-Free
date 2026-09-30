'use client';

import PageContainer from '@/components/layout/page-container';
import { PharmacyExpiryPanel } from '@/features/inventory/components/pharmacy-expiry-panel';
import { posApi, unwrapList, type PosMedicineBatch, type PosProduct } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';

export default function PharmacyExpiryPage() {
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [batches, setBatches] = useState<PosMedicineBatch[]>([]);
  const [error, setError] = useState('');
  const load = useCallback(async () => {
    setError('');
    try {
      const [profile, productResult, batchResult] = await Promise.all([
        posApi<{ organisation?: { business_type?: string } }>('/accounts/me'),
        posApi<unknown>('/inventory/products?page=1&page_size=100'),
        posApi<unknown>('/pharmacy/batches')
      ]);
      if (profile.data.organisation?.business_type !== 'pharmacy')
        throw new Error('This page is available to pharmacy accounts.');
      setProducts(unwrapList<PosProduct>(productResult.data));
      setBatches(unwrapList<PosMedicineBatch>(batchResult.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load pharmacy batches.');
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <PageContainer
      pageTitle='Batches & Expiry'
      pageDescription='Review batch traceability and remove expiry risk from daily dispensing.'
    >
      {error ? (
        <p role='alert' className='text-destructive rounded-md border p-4'>
          {error}
        </p>
      ) : (
        <PharmacyExpiryPanel batches={batches} products={products} onReceived={() => void load()} />
      )}
    </PageContainer>
  );
}
