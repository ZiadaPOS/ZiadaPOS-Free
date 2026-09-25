'use client';

import PageContainer from '@/components/layout/page-container';
import { StoreWorkspace } from '@/features/stores/components/store-workspace';
import { useText } from '@/lib/i18n';

export default function StoresPage() {
  const text = useText();
  return (
    <PageContainer
      pageTitle={text('Stores', 'Matawi')}
      pageDescription={text(
        'Manage branches and active store settings.',
        'Simamia matawi na mipangilio ya tawi linalotumika.'
      )}
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <StoreWorkspace />
      </div>
    </PageContainer>
  );
}
