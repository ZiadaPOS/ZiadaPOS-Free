'use client';

import PageContainer from '@/components/layout/page-container';
import { HelpWorkspace } from '@/features/help/components/help-workspace';
import { useText } from '@/lib/i18n';

export default function HelpPage() {
  const text = useText();
  return (
    <PageContainer
      pageTitle={text('Help', 'Msaada')}
      pageDescription={text(
        'Guides and answers for using Ziada POS.',
        'Mwongozo na majibu ya kutumia Ziada POS.'
      )}
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <HelpWorkspace />
      </div>
    </PageContainer>
  );
}
