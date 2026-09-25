'use client';

import PageContainer from '@/components/layout/page-container';
import { HelpWorkspace } from '@/features/help/components/help-workspace';

export default function HelpPage() {
  return (
    <PageContainer pageTitle='Help' pageDescription='Guides and answers for using Ziada POS.'>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <HelpWorkspace />
      </div>
    </PageContainer>
  );
}
