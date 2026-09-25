'use client';

import PageContainer from '@/components/layout/page-container';
import { ResourcePage } from '@/features/staff/components/staff-workspace';
import { useText } from '@/lib/i18n';

export default function StaffPage() {
  const text = useText();
  return (
    <PageContainer
      pageTitle={text('Staff', 'Wafanyakazi')}
      pageDescription={text(
        'Manage staff for the active store.',
        'Simamia wafanyakazi wa tawi linalotumika.'
      )}
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <ResourcePage />
      </div>
    </PageContainer>
  );
}
