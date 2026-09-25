'use client';

import PageContainer from '@/components/layout/page-container';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { posApi } from '@/lib/pos-api';
import { ThemeModeToggle } from '@/components/themes/theme-mode-toggle';
import { ThemeSelector } from '@/components/themes/theme-selector';
import { useCallback, useEffect, useState } from 'react';
import { StoreSwitcher } from '@/features/settings/components/store-switcher';
import { BusinessSettings } from '@/features/settings/components/business-settings';

export default function SettingsPage() {
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
            <CardTitle>Appearance</CardTitle>
            <CardDescription>Choose your workspace style and light or dark mode.</CardDescription>
          </CardHeader>
          <CardContent className='flex flex-wrap items-center gap-3'>
            <ThemeSelector />
            <ThemeModeToggle />
          </CardContent>
        </Card>
        <StoreSwitcher />
        {(user?.role === 'owner' || user?.role === 'admin') && (
          <BusinessSettings
            initialValue={settingsData?.organisation as Record<string, unknown> | undefined}
          />
        )}
      </div>
    </PageContainer>
  );
}
