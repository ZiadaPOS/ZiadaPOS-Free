import { NavGroup } from '@/types';

export const navGroups: NavGroup[] = [
  {
    label: 'Point of sale',
    items: [
      {
        title: 'Overview',
        url: '/dashboard/overview',
        icon: 'dashboard',
        isActive: false,
        shortcut: ['d', 'd'],
        items: []
      },
      {
        title: 'POS',
        url: '/dashboard/checkout',
        icon: 'product',
        isActive: false,
        shortcut: ['c', 'o'],
        items: []
      },
      {
        title: 'Transactions',
        url: '/dashboard/transactions',
        icon: 'creditCard',
        isActive: false,
        items: []
      },
      { title: 'Sales', url: '/dashboard/sales', icon: 'post', isActive: false, items: [] }
    ]
  },
  {
    label: 'Business',
    items: [
      {
        title: 'Inventory',
        url: '/dashboard/inventory',
        icon: 'product',
        isActive: false,
        items: []
      },
      {
        title: 'Customers',
        url: '/dashboard/customers',
        icon: 'teams',
        isActive: false,
        items: []
      },
      {
        title: 'Credits',
        url: '/dashboard/credits',
        icon: 'creditCard',
        isActive: false,
        items: []
      },
      {
        title: 'Suppliers',
        url: '/dashboard/suppliers',
        icon: 'product',
        isActive: false,
        items: []
      },
      {
        title: 'Expenses',
        url: '/dashboard/expenses',
        icon: 'creditCard',
        isActive: false,
        items: []
      },
      { title: 'Staff', url: '/dashboard/staff', icon: 'user', isActive: false, items: [] },
      {
        title: 'Branches',
        url: '/dashboard/stores',
        icon: 'workspace',
        isActive: false,
        items: []
      },
      {
        title: 'Analytics',
        url: '/dashboard/analytics',
        icon: 'dashboard',
        isActive: false,
        items: []
      },
      {
        title: 'Reports',
        url: '/dashboard/reports',
        icon: 'dashboard',
        isActive: false,
        items: []
      },
      {
        title: 'SMS messaging',
        url: '/dashboard/sms',
        icon: 'send',
        isActive: false,
        items: []
      },
      {
        title: 'Ziada AI',
        url: '/dashboard/ai-chat',
        icon: 'sparkles',
        isActive: false,
        items: []
      }
    ]
  },
  {
    label: 'Account',
    items: [
      {
        title: 'Settings',
        url: '/dashboard/settings',
        icon: 'settings',
        isActive: false,
        items: []
      },
      {
        title: 'Business settings',
        url: '/dashboard/business-settings',
        icon: 'settings',
        isActive: false,
        items: []
      },
      {
        title: 'Help & support',
        url: '/dashboard/help',
        icon: 'help',
        isActive: false,
        items: []
      }
    ]
  }
];
