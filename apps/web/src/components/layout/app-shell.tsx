'use client';

import KBar from '@/components/kbar';
import AppSidebar from '@/components/layout/app-sidebar';
import Header from '@/components/layout/header';
import { InfobarProvider } from '@/components/ui/infobar';
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { syncCamelIdentity } from '@/lib/camel-analytics';
import { usePathname } from 'next/navigation';
import Script from 'next/script';

export function AppShell({
  children,
  defaultOpen
}: {
  children: React.ReactNode;
  defaultOpen: boolean;
}) {
  const pathname = usePathname();
  const isDesktopLocalMode = process.env.NEXT_PUBLIC_ZIADAPOS_LOCAL_MODE === '1';
  const isPublicOrRedirectRoute =
    pathname === '/' ||
    pathname === '/auth' ||
    pathname.startsWith('/auth/') ||
    pathname.startsWith('/dashboard');

  const tracker = !isDesktopLocalMode && (
    <Script
      id='camel-analytics-tracker'
      src='https://analytics.camelcreatives.com/tracker.js'
      data-site='ziadapos-webapp'
      strategy='afterInteractive'
      defer
      onReady={syncCamelIdentity}
    />
  );

  if (isPublicOrRedirectRoute) {
    return (
      <>
        {tracker}
        {children}
      </>
    );
  }

  return (
    <>
      {tracker}
      <KBar>
        <SidebarProvider defaultOpen={defaultOpen}>
          <a
            href='#main-content'
            className='bg-background ring-ring sr-only rounded-md px-3 py-2 text-sm font-medium shadow focus:not-sr-only focus:absolute focus:top-2 focus:start-2 focus:z-50 focus:ring-2'
          >
            Skip to content
          </a>
          <AppSidebar />
          <SidebarInset id='main-content' tabIndex={-1} className='scroll-mt-16'>
            <Header />
            <InfobarProvider defaultOpen={false}>{children}</InfobarProvider>
          </SidebarInset>
        </SidebarProvider>
      </KBar>
    </>
  );
}
