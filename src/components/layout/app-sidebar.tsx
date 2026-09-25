'use client';

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail
} from '@/components/ui/sidebar';
import { navGroups } from '@/config/nav-config';
import { Icons } from '@/components/icons';
import { useRouter, usePathname } from 'next/navigation';
import Link from 'next/link';
import { useEffect, useState } from 'react';
import Image from 'next/image';
import { useLanguage, type TranslationKey } from '@/lib/i18n';

export default function AppSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const [role, setRole] = useState('staff');
  const { t } = useLanguage();
  useEffect(() => {
    try {
      setRole(JSON.parse(localStorage.getItem('pos_user') || '{}').role || 'staff');
    } catch {
      setRole('staff');
    }
  }, []);
  async function signOut() {
    try {
      await fetch('/api/pos-session', { method: 'DELETE' });
    } catch {
      // Clear the local session even when the network is unavailable.
    } finally {
      localStorage.removeItem('pos_user');
      router.replace('/auth/sign-in');
    }
  }
  const managerOnly = new Set(['Staff', 'Branches', 'Reports', 'SMS messaging', 'Settings']);
  const label = (value: string) => {
    const key = (
      {
        'Point of sale': 'pointOfSale',
        Business: 'business',
        Account: 'account',
        Overview: 'overview',
        POS: 'pos',
        Transactions: 'transactions',
        Sales: 'sales',
        Inventory: 'inventory',
        Customers: 'customers',
        Credits: 'credits',
        Suppliers: 'suppliers',
        Expenses: 'expenses',
        Staff: 'staff',
        Branches: 'branches',
        Analytics: 'analytics',
        Reports: 'reports',
        'SMS messaging': 'sms',
        'Ziada AI': 'ziadaAI',
        Settings: 'settings',
        'Business settings': 'businessSettings',
        'Sign out': 'signOut'
      } as Record<string, TranslationKey>
    )[value];
    return key ? t(key) : value;
  };
  return (
    <Sidebar collapsible='icon'>
      <SidebarHeader className='px-4 py-4'>
        <Link href='/dashboard/overview' className='flex items-center gap-2 font-bold'>
          <Image
            src='/ziada-final.jpeg'
            alt='Ziada'
            width={28}
            height={28}
            className='size-7 rounded-md object-cover'
            priority
          />
          <span>Ziada POS</span>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        {navGroups.map((group) => (
          <SidebarGroup key={group.label}>
            <SidebarGroupLabel>{label(group.label)}</SidebarGroupLabel>
            <SidebarMenu>
              {group.items
                .filter(
                  (item) => !managerOnly.has(item.title) || role === 'owner' || role === 'admin'
                )
                .map((item) => {
                  const Icon = item.icon ? Icons[item.icon] : Icons.logo;
                  return item.items?.length ? (
                    <Collapsible
                      key={item.title}
                      defaultOpen={item.isActive}
                      render={<SidebarMenuItem />}
                    >
                      <CollapsibleTrigger
                        render={
                          <SidebarMenuButton
                            tooltip={label(item.title)}
                            isActive={pathname === item.url}
                          />
                        }
                      >
                        <Icon />
                        <span>{label(item.title)}</span>
                        <Icons.chevronRight className='ml-auto size-4' />
                      </CollapsibleTrigger>
                      <CollapsibleContent>
                        <SidebarMenuSub>
                          {item.items.map((child) => (
                            <SidebarMenuSubItem key={child.title}>
                              <SidebarMenuSubButton
                                render={<Link href={child.url} />}
                                isActive={pathname === child.url}
                              >
                                {label(child.title)}
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                          ))}
                        </SidebarMenuSub>
                      </CollapsibleContent>
                    </Collapsible>
                  ) : (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton
                        render={<Link href={item.url} />}
                        tooltip={label(item.title)}
                        isActive={pathname === item.url}
                      >
                        <Icon />
                        <span>{label(item.title)}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
            </SidebarMenu>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton onClick={() => router.push('/dashboard/settings')}>
              <Icons.settings />
              <span>{label('Business settings')}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton onClick={() => void signOut()}>
              <Icons.logout />
              <span>{label('Sign out')}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
