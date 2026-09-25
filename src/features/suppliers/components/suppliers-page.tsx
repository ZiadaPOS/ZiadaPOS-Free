'use client';

import PageContainer from '@/components/layout/page-container';
import { SupplierWorkspace } from '@/features/suppliers/components/supplier-workspace';
import { SupplierOperations } from '@/features/suppliers/components/supplier-operations';
import { useText } from '@/lib/i18n';

export default function SuppliersPage() {
  const text = useText();
  return (
    <PageContainer
      pageTitle={text('Suppliers', 'Wasambazaji')}
      pageDescription={text(
        'Manage suppliers, deliveries, and balances.',
        'Simamia wasambazaji, uwasilishaji, na madeni.'
      )}
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <SupplierWorkspace />
        <SupplierOperations />
      </div>
    </PageContainer>
  );
}
