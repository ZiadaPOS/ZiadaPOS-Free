'use client';

import PageContainer from '@/components/layout/page-container';
import { posApi, unwrapList, type PosTransaction } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { SalesTable } from '@/features/sales/components/sales-table';

export default function SalesPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const load = useCallback(async () => {
    setError('');
    try {
      const result = await posApi<unknown>('/pos/transactions?page=1&page_size=50');
      setTransactions(unwrapList<PosTransaction>(result.data));
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
    <PageContainer pageTitle='Sales'>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}
        <SalesTable transactions={transactions} onUpdated={() => void load()} />
      </div>
    </PageContainer>
  );
}
