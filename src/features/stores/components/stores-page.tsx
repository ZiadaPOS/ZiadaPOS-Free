'use client';

import PageContainer from '@/components/layout/page-container';
import { StoreWorkspace } from '@/features/stores/components/store-workspace';

export default function StoresPage() {
  return (
    <PageContainer pageTitle='Stores' pageDescription='Manage branches and active store settings.'>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <StoreWorkspace />
      </div>
    </PageContainer>
  );
}
