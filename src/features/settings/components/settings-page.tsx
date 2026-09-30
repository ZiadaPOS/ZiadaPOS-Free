'use client';

import PageContainer from '@/components/layout/page-container';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { posApi } from '@/lib/pos-api';
import { ThemeModeToggle } from '@/components/themes/theme-mode-toggle';
import { useCallback, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';
import { StoreSwitcher } from '@/features/settings/components/store-switcher';
import { BusinessSettings } from '@/features/settings/components/business-settings';
import Link from 'next/link';
import { Icons } from '@/components/icons';

export default function SettingsPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [settingsData, setSettingsData] = useState<Record<string, unknown> | null>(null);
  const load = useCallback(async () => {
    setError('');
    try {
      const me = await posApi<unknown>('/accounts/me');
      setSettingsData(me.data as Record<string, unknown>);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load data.');
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  const [user, setUser] = useState<{ role?: string } | null>(null);
  const businessType = String(
    (settingsData?.organisation as Record<string, unknown> | undefined)?.business_type || ''
  );
  useEffect(() => {
    try {
      const saved = localStorage.getItem('pos_user');
      if (saved) setUser(JSON.parse(saved));
    } catch {
      /* Invalid cached profile. */
    }
  }, []);
  return (
    <PageContainer pageTitle='Settings'>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}
        <Card>
          <CardHeader>
            <CardTitle>{text('Appearance', 'Muonekano')}</CardTitle>
            <CardDescription>
              {text(
                'Choose your workspace style and light or dark mode.',
                'Chagua muonekano wa mfumo na hali ya mwanga au giza.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='flex items-center gap-3'>
            <span className='text-muted-foreground text-sm'>Supabase</span>
            <ThemeModeToggle />
          </CardContent>
        </Card>
        <StoreSwitcher />
        <Card>
          <CardHeader>
            <CardTitle>{text('Business tools', 'Zana za biashara')}</CardTitle>
            <CardDescription>
              {text(
                'Manage your suppliers and pharmacy settings.',
                'Simamia wasambazaji na mipangilio ya famasia.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='grid gap-3 sm:grid-cols-2'>
            <Link
              href='/dashboard/suppliers'
              className='hover:bg-muted/50 focus-visible:ring-ring flex min-h-20 items-center gap-3 rounded-lg border p-4 transition-colors focus-visible:ring-2 focus-visible:outline-none'
            >
              <Icons.truckDelivery className='text-primary size-5 shrink-0' />
              <span className='min-w-0 flex-1'>
                <span className='block font-medium'>{text('Suppliers', 'Wasambazaji')}</span>
                <span className='text-muted-foreground block text-sm'>
                  {text(
                    'Manage suppliers, deliveries, and payments.',
                    'Simamia wasambazaji, mizigo na malipo.'
                  )}
                </span>
              </span>
              <Icons.chevronRight className='text-muted-foreground size-4 shrink-0' />
            </Link>
            {businessType === 'pharmacy' && (
              <Link
                href='/dashboard/pharmacy/schedules'
                className='hover:bg-muted/50 focus-visible:ring-ring flex min-h-20 items-center gap-3 rounded-lg border p-4 transition-colors focus-visible:ring-2 focus-visible:outline-none'
              >
                <Icons.clipboardList className='text-primary size-5 shrink-0' />
                <span className='min-w-0 flex-1'>
                  <span className='block font-medium'>
                    {text('Medicine schedules', 'Ratiba za dawa')}
                  </span>
                  <span className='text-muted-foreground block text-sm'>
                    {text(
                      'Review medicine classifications and checkout rules.',
                      'Kagua uainishaji wa dawa na masharti ya mauzo.'
                    )}
                  </span>
                </span>
                <Icons.chevronRight className='text-muted-foreground size-4 shrink-0' />
              </Link>
            )}
          </CardContent>
        </Card>
        {(user?.role === 'owner' || user?.role === 'admin') && (
          <BusinessSettings
            initialValue={settingsData?.organisation as Record<string, unknown> | undefined}
          />
        )}
      </div>
    </PageContainer>
  );
}
