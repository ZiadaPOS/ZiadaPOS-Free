'use client';

import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { posApi, unwrapList, type PosProduct } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { InventoryWorkspace } from '@/features/inventory/components/inventory-workspace';
import { CategoryManager } from '@/features/inventory/components/category-manager';
import { CreateProduct } from '@/features/inventory/components/create-product';

export default function InventoryPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const load = useCallback(async () => {
    setError('');
    try {
      const result = await posApi<unknown>('/inventory/products?page=1&page_size=100');
      setProducts(unwrapList<PosProduct>(result.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load data.');
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <PageContainer
      pageTitle={text('Inventory', 'Stoo')}
      pageDescription={text(
        'Products and current stock for this store.',
        'Bidhaa na stoo ya sasa ya tawi hili.'
      )}
      pageHeaderAction={
        <div className='grid w-full grid-cols-[2fr_3fr] gap-2 sm:flex sm:w-auto'>
          <CategoryManager />
          <Button className='w-full sm:w-auto' onClick={() => setShowCreate((v) => !v)}>
            {showCreate ? text('Close', 'Funga') : text('Add product', 'Ongeza bidhaa')}
          </Button>
        </div>
      }
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}
        {showCreate && (
          <CreateProduct
            onCreated={() => {
              setShowCreate(false);
              void load();
            }}
          />
        )}
        <InventoryWorkspace products={products} onUpdated={() => void load()} />
      </div>
    </PageContainer>
  );
}
