'use client';

import PageContainer from '@/components/layout/page-container';
import { posApi, unwrapList, type PosTransaction } from '@/lib/pos-api';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { SalesTable } from '@/features/sales/components/sales-table';

export default function SalesPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [localMode, setLocalMode] = useState(false);
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const load = useCallback(async () => {
    setError('');
    try {
      const [result, profile] = await Promise.all([
        posApi<unknown>('/pos/transactions?page=1&page_size=50'),
        posApi<{ offline_mode?: boolean }>('/accounts/me')
      ]);
      setLocalMode(Boolean(profile.data.offline_mode));
      setTransactions(unwrapList<PosTransaction>(result.data));
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : text('Could not load data.', 'Imeshindikana kupakia taarifa.')
      );
    }
  }, [text]);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <PageContainer
      pageTitle={text('Sales', 'Mauzo')}
      pageDescription={
        localMode
          ? 'Local cash-sale history stored on this computer; it is not synced online.'
          : text(
              'Transaction ledger, receipts, refunds, and voids.',
              'Orodha ya miamala, risiti, marejesho, na kufuta miamala.'
            )
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
        <SalesTable
          transactions={transactions}
          localMode={localMode}
          onUpdated={() => void load()}
        />
      </div>
    </PageContainer>
  );
}
