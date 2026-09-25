import React from 'react';
import { SidebarTrigger } from '../ui/sidebar';
import { Separator } from '../ui/separator';
import { Breadcrumbs } from '../breadcrumbs';
import SearchInput from '../search-input';
import { ThemeModeToggle } from '../themes/theme-mode-toggle';
import { NotificationCenter } from '@/features/notifications/components/notification-center';

export default function Header() {
  return (
    <header className='bg-background/60 sticky top-0 z-20 flex h-16 shrink-0 items-center justify-between gap-2 backdrop-blur-md md:h-14'>
      <div className='min-w-0 shrink-0 px-2 sm:px-4'>
        <SidebarTrigger className='-ml-1' />
        <Separator orientation='vertical' className='mr-2 h-4 data-vertical:self-center' />
        <span className='hidden sm:inline-flex'><Breadcrumbs /></span>
      </div>

      <div className='flex min-w-0 flex-1 justify-center px-1 sm:px-4'>
        <div className='hidden w-full max-w-2xl md:flex'>
          <SearchInput />
        </div>
      </div>

      <div className='flex shrink-0 items-center gap-1 px-2 sm:gap-2 sm:px-4'>
        <ThemeModeToggle />
        <NotificationCenter />
      </div>
    </header>
  );
}
