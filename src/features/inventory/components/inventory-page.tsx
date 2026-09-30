'use client';

import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { posApi, unwrapList, type PosMedicineBatch, type PosProduct } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { InventoryWorkspace } from '@/features/inventory/components/inventory-workspace';
import { CategoryManager } from '@/features/inventory/components/category-manager';
import { CreateProduct } from '@/features/inventory/components/create-product';
import { differenceInCalendarDays, parseISO } from 'date-fns';

const CREATE_PRODUCT_DIALOG_KEY = 'ziada-pos:inventory:create-product:open:v1';

export default function InventoryPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [batches, setBatches] = useState<PosMedicineBatch[]>([]);
  const [isPharmacy, setIsPharmacy] = useState(false);
  const [businessType, setBusinessType] = useState('retail');
  const [showCreate, setShowCreate] = useState(false);
  useEffect(() => {
    try {
      setShowCreate(localStorage.getItem(CREATE_PRODUCT_DIALOG_KEY) === 'open');
    } catch {
      setShowCreate(false);
    }
  }, []);
  function setCreateDialogOpen(open: boolean) {
    setShowCreate(open);
    try {
      if (open) localStorage.setItem(CREATE_PRODUCT_DIALOG_KEY, 'open');
      else localStorage.removeItem(CREATE_PRODUCT_DIALOG_KEY);
    } catch {
      // The dialog remains usable when browser storage is unavailable.
    }
  }
  const load = useCallback(async () => {
    setError('');
    try {
      const result = await posApi<unknown>('/inventory/products?page=1&page_size=100');
      const productList = unwrapList<PosProduct>(result.data);
      const profile = await posApi<{ organisation?: { business_type?: string } }>('/accounts/me');
      const type = profile.data.organisation?.business_type || 'retail';
      const pharmacy = type === 'pharmacy';
      setBusinessType(type);
      setIsPharmacy(pharmacy);
      if (pharmacy) {
        const lotResult = await posApi<unknown>('/pharmacy/batches');
        const lotList = unwrapList<PosMedicineBatch>(lotResult.data);
        setBatches(lotList);
        const today = new Date();
        const stockByProduct = new Map<
          string,
          { tracked: number; sellable: number; expired: number; nextExpiry?: string }
        >();
        for (const batch of lotList) {
          if (batch.quantity <= 0) continue;
          const current = stockByProduct.get(batch.product_id) || {
            tracked: 0,
            sellable: 0,
            expired: 0
          };
          const daysLeft = differenceInCalendarDays(parseISO(batch.expiry_date), today);
          current.tracked += batch.quantity;
          if (daysLeft < 0) current.expired += batch.quantity;
          else {
            current.sellable += batch.quantity;
            if (!current.nextExpiry || batch.expiry_date < current.nextExpiry)
              current.nextExpiry = batch.expiry_date;
          }
          stockByProduct.set(batch.product_id, current);
        }
        setProducts(
          productList.map((product) => {
            const batchesForProduct = stockByProduct.get(product.id) || {
              tracked: 0,
              sellable: 0,
              expired: 0
            };
            const sellable = batchesForProduct.sellable;
            return {
              ...product,
              sellable_stock: sellable,
              expired_stock: batchesForProduct.expired,
              untracked_stock: Math.max(0, product.stock - batchesForProduct.tracked),
              next_expiry_date: batchesForProduct.nextExpiry,
              low_stock: !product.is_service && sellable <= product.reorder_point
            };
          })
        );
      } else {
        setBatches([]);
        setProducts(productList);
      }
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
          <Button className='w-full sm:w-auto' onClick={() => setCreateDialogOpen(!showCreate)}>
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
            isPharmacy={isPharmacy}
            businessType={businessType}
            onCreated={() => {
              setCreateDialogOpen(false);
              void load();
            }}
          />
        )}
        <InventoryWorkspace
          products={products}
          isPharmacy={isPharmacy}
          onUpdated={() => void load()}
        />
      </div>
    </PageContainer>
  );
}
