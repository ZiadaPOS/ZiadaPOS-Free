'use client';

import PageContainer from '@/components/layout/page-container';
import { posApi, unwrapList, type PosCustomer, type PosCreditTab } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { CreditTable } from '@/features/credits/components/credit-table';

export default function CreditsPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [customers, setCustomers] = useState<PosCustomer[]>([]);
  const [creditTabs, setCreditTabs] = useState<PosCreditTab[]>([]);
  const load = useCallback(async () => {
    setError('');
    try {
      const [credit, customerResult] = await Promise.all([
        posApi<unknown>('/credits?page=1&page_size=100'),
        posApi<unknown>('/customers?page=1&page_size=100')
      ]);
      setCreditTabs(unwrapList<PosCreditTab>(credit.data));
      setCustomers(unwrapList<PosCustomer>(customerResult.data));
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : text('Could not load data.', 'Imeshindikana kupakia taarifa.')
      );
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <PageContainer pageTitle={text('Credit accounts', 'Akaunti za mikopo')}>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}
        <CreditTable tabs={creditTabs} customers={customers} onUpdated={() => void load()} />
      </div>
    </PageContainer>
  );
}
