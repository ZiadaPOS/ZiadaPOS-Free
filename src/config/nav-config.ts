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
        icon: 'cashRegister',
        isActive: false,
        shortcut: ['c', 'o'],
        items: []
      },
      {
        title: 'Transactions',
        url: '/dashboard/transactions',
        icon: 'receipt',
        isActive: false,
        items: []
      },
      { title: 'Sales', url: '/dashboard/sales', icon: 'trendingUp', isActive: false, items: [] }
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
        title: 'Batches & Expiry',
        url: '/dashboard/pharmacy/expiry',
        icon: 'calendarTime',
        isActive: false,
        items: []
      },
      {
        title: 'Prescription records',
        url: '/dashboard/pharmacy/prescriptions',
        icon: 'prescription',
        isActive: false,
        items: []
      },
      {
        title: 'Categories',
        url: '/dashboard/categories',
        icon: 'category',
        isActive: false,
        items: []
      },
      {
        title: 'Units',
        url: '/dashboard/units',
        icon: 'rulerMeasure',
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
        icon: 'wallet',
        isActive: false,
        items: []
      },
      {
        title: 'Expenses',
        url: '/dashboard/expenses',
        icon: 'cashMinus',
        isActive: false,
        items: []
      },
      { title: 'Staff', url: '/dashboard/staff', icon: 'userCog', isActive: false, items: [] },
      {
        title: 'Branches',
        url: '/dashboard/stores',
        icon: 'store',
        isActive: false,
        items: []
      },
      {
        title: 'Analytics',
        url: '/dashboard/analytics',
        icon: 'chartHistogram',
        isActive: false,
        items: []
      },
      {
        title: 'Reports',
        url: '/dashboard/reports',
        icon: 'reportAnalytics',
        isActive: false,
        items: []
      },
      {
        title: 'Notebook',
        url: '/dashboard/notebook',
        icon: 'notebook',
        isActive: false,
        items: []
      },
      {
        title: 'Subscriptions',
        url: '/dashboard/billing',
        icon: 'billing',
        isActive: false,
        items: []
      },
      {
        title: 'SMS messaging',
        url: '/dashboard/sms',
        icon: 'deviceMessage',
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
        title: 'Help & support',
        url: '/dashboard/help',
        icon: 'help',
        isActive: false,
        items: []
      }
    ]
  }
];
