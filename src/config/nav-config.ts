import { NavGroup } from '@/types';

export const navGroups: NavGroup[] = [
  {
    label: 'Point of sale',
    items: [
      {
        title: 'Overview',
        url: '/overview',
        icon: 'dashboard',
        isActive: false,
        shortcut: ['d', 'd'],
        items: []
      },
      {
        title: 'POS',
        url: '/checkout',
        icon: 'cashRegister',
        isActive: false,
        shortcut: ['c', 'o'],
        items: []
      },
      {
        title: 'Transactions',
        url: '/transactions',
        icon: 'receipt',
        isActive: false,
        items: []
      },
      { title: 'Sales', url: '/sales', icon: 'trendingUp', isActive: false, items: [] }
    ]
  },
  {
    label: 'Business',
    items: [
      {
        title: 'Inventory',
        url: '/inventory',
        icon: 'product',
        isActive: false,
        items: []
      },
      {
        title: 'Batches & Expiry',
        url: '/pharmacy/expiry',
        icon: 'calendarTime',
        isActive: false,
        items: []
      },
      {
        title: 'Prescription records',
        url: '/pharmacy/prescriptions',
        icon: 'prescription',
        isActive: false,
        items: []
      },
      {
        title: 'Categories',
        url: '/categories',
        icon: 'category',
        isActive: false,
        items: []
      },
      {
        title: 'Units',
        url: '/units',
        icon: 'rulerMeasure',
        isActive: false,
        items: []
      },
      {
        title: 'Customers',
        url: '/customers',
        icon: 'teams',
        isActive: false,
        items: []
      },
      {
        title: 'Credits',
        url: '/credits',
        icon: 'wallet',
        isActive: false,
        items: []
      },
      {
        title: 'Expenses',
        url: '/expenses',
        icon: 'cashMinus',
        isActive: false,
        items: []
      },
      { title: 'Staff', url: '/staff', icon: 'userCog', isActive: false, items: [] },
      {
        title: 'Branches',
        url: '/stores',
        icon: 'store',
        isActive: false,
        items: []
      },
      {
        title: 'Analytics',
        url: '/analytics',
        icon: 'chartHistogram',
        isActive: false,
        items: []
      },
      {
        title: 'Reports',
        url: '/reports',
        icon: 'reportAnalytics',
        isActive: false,
        items: []
      },
      {
        title: 'Notebook',
        url: '/notebook',
        icon: 'notebook',
        isActive: false,
        items: []
      },
      {
        title: 'Subscriptions',
        url: '/billing',
        icon: 'billing',
        isActive: false,
        items: []
      },
      {
        title: 'SMS messaging',
        url: '/sms',
        icon: 'deviceMessage',
        isActive: false,
        items: []
      },
      {
        title: 'Ziada AI',
        url: '/ai-chat',
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
        url: '/settings',
        icon: 'settings',
        isActive: false,
        items: []
      },
      {
        title: 'Help & support',
        url: '/help',
        icon: 'help',
        isActive: false,
        items: []
      }
    ]
  }
];
