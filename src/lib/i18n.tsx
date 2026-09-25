'use client';

import { createContext, useContext, useEffect, useMemo, useState } from 'react';

export type Language = 'en' | 'sw';

type TranslationKey =
  | 'language'
  | 'english'
  | 'swahili'
  | 'toggleLanguage'
  | 'toggleTheme'
  | 'pointOfSale'
  | 'business'
  | 'account'
  | 'overview'
  | 'pos'
  | 'transactions'
  | 'sales'
  | 'inventory'
  | 'customers'
  | 'credits'
  | 'suppliers'
  | 'expenses'
  | 'staff'
  | 'branches'
  | 'analytics'
  | 'reports'
  | 'ziadaAI'
  | 'settings'
  | 'businessSettings'
  | 'signOut'
  | 'search';

const translations: Record<Language, Record<TranslationKey, string>> = {
  en: {
    language: 'Language',
    english: 'English',
    swahili: 'Swahili',
    toggleLanguage: 'Switch language',
    toggleTheme: 'Toggle theme',
    pointOfSale: 'Point of sale',
    business: 'Business',
    account: 'Account',
    overview: 'Overview',
    pos: 'POS',
    transactions: 'Transactions',
    sales: 'Sales',
    inventory: 'Inventory',
    customers: 'Customers',
    credits: 'Credits',
    suppliers: 'Suppliers',
    expenses: 'Expenses',
    staff: 'Staff',
    branches: 'Branches',
    analytics: 'Analytics',
    reports: 'Reports',
    ziadaAI: 'Ziada AI',
    settings: 'Settings',
    businessSettings: 'Business settings',
    signOut: 'Sign out',
    search: 'Search...'
  },
  sw: {
    language: 'Lugha',
    english: 'Kiingereza',
    swahili: 'Kiswahili',
    toggleLanguage: 'Badili lugha',
    toggleTheme: 'Badili mandhari',
    pointOfSale: 'Mauzo',
    business: 'Biashara',
    account: 'Akaunti',
    overview: 'Muhtasari',
    pos: 'POS',
    transactions: 'Miamala',
    sales: 'Mauzo',
    inventory: 'Stoo',
    customers: 'Wateja',
    credits: 'Mikopo',
    suppliers: 'Wasambazaji',
    expenses: 'Gharama',
    staff: 'Wafanyakazi',
    branches: 'Matawi',
    analytics: 'Uchambuzi',
    reports: 'Ripoti',
    ziadaAI: 'Ziada AI',
    settings: 'Mipangilio',
    businessSettings: 'Mipangilio ya biashara',
    signOut: 'Ondoka',
    search: 'Tafuta...'
  }
};

type LanguageContextValue = {
  language: Language;
  setLanguage: (language: Language) => void;
  toggleLanguage: () => void;
  t: (key: TranslationKey) => string;
};
const LanguageContext = createContext<LanguageContextValue | null>(null);

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguageState] = useState<Language>('en');

  useEffect(() => {
    const saved = window.localStorage.getItem('ziada-language');
    if (saved === 'en' || saved === 'sw') setLanguageState(saved);
  }, []);

  const setLanguage = (next: Language) => {
    setLanguageState(next);
    window.localStorage.setItem('ziada-language', next);
    document.documentElement.lang = next === 'sw' ? 'sw' : 'en';
  };

  const value = useMemo<LanguageContextValue>(
    () => ({
      language,
      setLanguage,
      toggleLanguage: () => setLanguage(language === 'en' ? 'sw' : 'en'),
      t: (key) => translations[language][key]
    }),
    [language]
  );

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>;
}

export function useLanguage() {
  const context = useContext(LanguageContext);
  if (!context) throw new Error('useLanguage must be used within LanguageProvider');
  return context;
}

export type { TranslationKey };
