'use client';
import React from 'react';
import { ActiveThemeProvider } from '../themes/active-theme';
import QueryProvider from './query-provider';
import { LanguageProvider } from '@/lib/i18n';

export default function Providers({
  activeThemeValue,
  children
}: {
  activeThemeValue: string;
  children: React.ReactNode;
}) {
  return (
    <>
      <LanguageProvider>
        <ActiveThemeProvider initialTheme={activeThemeValue}>
          <QueryProvider>{children}</QueryProvider>
        </ActiveThemeProvider>
      </LanguageProvider>
    </>
  );
}
