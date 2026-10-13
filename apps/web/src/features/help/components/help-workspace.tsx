'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { useState } from 'react';
import { useText } from '@/lib/i18n';

export function HelpWorkspace() {
  const text = useText();
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('All');
  const articles = [
    [
      'Getting started',
      'Setup',
      'Create your business profile, add a store, invite staff, and configure tax before opening the till.'
    ],
    [
      'Process your first sale',
      'POS',
      'Open POS, search or scan a product, tap the product to add it, choose a payment method, and complete the sale.'
    ],
    [
      'Add products and stock',
      'Inventory',
      'Open Inventory → Add product. Add cost, selling price, opening stock, reorder point, category, and image, then save.'
    ],
    [
      'Record a customer credit sale',
      'Credits',
      'Choose Credit at checkout, select a customer, and complete the sale. The balance appears in Credits with its due status.'
    ],
    [
      'Track expenses correctly',
      'Expenses',
      'Record every operating cost with its category, payment method, reference, and note so analytics and reports stay accurate.'
    ],
    [
      'Manage suppliers',
      'Suppliers',
      'Add supplier deliveries with invoice totals and item counts, then record payments against the supplier balance.'
    ],
    [
      'Use multiple stores',
      'Stores',
      'Create each location from Stores, then switch the active branch from Settings. Inventory, sales, and reports follow the active store.'
    ],
    [
      'Invite a cashier',
      'Settings',
      'Open Settings → Users & roles, invite the team member, assign permissions, and have them sign in with their POS PIN.'
    ],
    [
      'Connect payment integrations',
      'Integrations',
      'Open Settings → Integrations, connect the supported payment or messaging provider, verify credentials, and run a small test transaction.'
    ],
    [
      'Export and understand reports',
      'Reports',
      'Use Reports for focused exports and Analytics for live business trends across sales, stock, customers, expenses, credits, and suppliers.'
    ]
  ];
  const categories = ['All', ...new Set(articles.map((article) => article[1]))];
  const visible = articles.filter(
    (article) =>
      (category === 'All' || article[1] === category) &&
      article.join(' ').toLowerCase().includes(query.toLowerCase())
  );
  return (
    <div className='space-y-4'>
      <div className='rounded-xl border bg-primary/5 p-5 sm:p-7'>
        <p className='text-primary text-xs font-medium uppercase tracking-[0.18em]'>
          {text('Help & support', 'Msaada na usaidizi')}
        </p>
        <h1 className='mt-2 text-2xl font-bold tracking-tight sm:text-3xl'>
          {text(
            'Learn Ziada from setup to daily operations',
            'Jifunze Ziada kuanzia usanidi hadi shughuli za kila siku'
          )}
        </h1>
        <p className='text-muted-foreground mt-2 max-w-2xl'>
          {text(
            'Practical guides for connecting your store, running sales, keeping records accurate, and using analytics to make better decisions.',
            'Mwongozo wa kuunganisha tawi, kufanya mauzo, kuweka kumbukumbu sahihi, na kutumia uchambuzi kufanya maamuzi bora.'
          )}
        </p>
        <div className='mt-5 flex max-w-2xl items-center gap-2'>
          <Icons.search className='text-muted-foreground h-4 w-4' />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={text('Search help articles…', 'Tafuta makala za msaada…')}
          />
        </div>
      </div>
      <div className='flex gap-2 overflow-x-auto pb-1'>
        {categories.map((item) => (
          <Button
            key={item}
            size='sm'
            variant={category === item ? 'default' : 'outline'}
            onClick={() => setCategory(item)}
          >
            {item}
          </Button>
        ))}
      </div>
      <div className='grid gap-3 lg:grid-cols-2'>
        {visible.map(([title, group, body]) => (
          <Card key={title}>
            <CardHeader>
              <CardTitle className='text-base'>{title}</CardTitle>
              <CardDescription>{group}</CardDescription>
            </CardHeader>
            <CardContent>
              <p className='text-muted-foreground text-sm leading-6'>{body}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Need more help?</CardTitle>
          <CardDescription>
            Our support team can help with onboarding, integrations, and account questions.
          </CardDescription>
        </CardHeader>
        <CardContent className='flex flex-wrap gap-2'>
          <a
            className='border-input hover:bg-accent rounded-md border px-3 py-2 text-sm'
            href='mailto:support@ziada.app'
          >
            Email support
          </a>
          <a
            className='border-input hover:bg-accent rounded-md border px-3 py-2 text-sm'
            href='https://wa.me/255800100200'
          >
            WhatsApp support
          </a>
          <a
            className='border-input hover:bg-accent rounded-md border px-3 py-2 text-sm'
            href='tel:+255800100200'
          >
            Call support
          </a>
        </CardContent>
      </Card>
    </div>
  );
}
