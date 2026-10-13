'use client';

import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { posApi, unwrapList, type PosCustomer } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { CreateCustomer } from '@/features/customers/components/create-customer';
import { CustomerWorkspace } from '@/features/customers/components/customer-workspace';

export default function CustomersPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [customers, setCustomers] = useState<PosCustomer[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const load = useCallback(async () => {
    setError('');
    try {
      const result = await posApi<unknown>('/customers?page=1&page_size=100');
      setCustomers(unwrapList<PosCustomer>(result.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load data.');
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <PageContainer
      pageTitle='Customers'
      pageDescription='Customer profiles, segments, spend, and outstanding credit.'
      pageHeaderAction={
        <Button className='w-full sm:w-auto' onClick={() => setShowCreate((v) => !v)}>
          {showCreate ? text('Close', 'Funga') : text('Add customer', 'Ongeza mteja')}
        </Button>
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
          <CreateCustomer
            onCreated={() => {
              setShowCreate(false);
              void load();
            }}
          />
        )}
        <CustomerWorkspace customers={customers} />
      </div>
    </PageContainer>
  );
}
