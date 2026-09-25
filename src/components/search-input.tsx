'use client';
import { useKBar } from 'kbar';
import { Icons } from '@/components/icons';
import { Button } from './ui/button';
import { useLanguage } from '@/lib/i18n';

export default function SearchInput() {
  const { query } = useKBar();
  const { t } = useLanguage();
  return (
    <div className='w-full min-w-0'>
      <Button
        variant='outline'
        className='bg-background text-muted-foreground relative flex h-10 w-full min-w-0 flex-row flex-nowrap items-center justify-start gap-2 overflow-hidden whitespace-nowrap rounded-[0.5rem] px-3 text-sm font-normal leading-none shadow-none sm:pr-12'
        onClick={query.toggle}
      >
        <Icons.search className='h-4 w-4 shrink-0' />
        <span className='truncate'>{t('search')}</span>
        <kbd className='bg-muted pointer-events-none absolute top-[0.3rem] right-[0.3rem] hidden h-6 items-center gap-1 rounded border px-1.5 font-mono text-[10px] font-medium opacity-100 select-none sm:flex'>
          <span className='text-xs'>⌘</span>K
        </kbd>
      </Button>
    </div>
  );
}
