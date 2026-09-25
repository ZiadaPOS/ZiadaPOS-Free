'use client';

import PageContainer from '@/components/layout/page-container';
import { SupplierWorkspace } from '@/features/suppliers/components/supplier-workspace';
import { SupplierOperations } from '@/features/suppliers/components/supplier-operations';

export default function SuppliersPage() {
  return (
    <PageContainer
      pageTitle='Suppliers'
      pageDescription='Manage suppliers, deliveries, and balances.'
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <SupplierWorkspace />
        <SupplierOperations />
      </div>
    </PageContainer>
  );
}
