'use client';

import PageContainer from '@/components/layout/page-container';
import { ResourcePage } from '@/features/staff/components/staff-workspace';

export default function StaffPage() {
  return (
    <PageContainer>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <ResourcePage />
      </div>
    </PageContainer>
  );
}
