'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { money, type PosProduct } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { InventoryTable } from '@/features/inventory/components/inventory-table';
import { useText } from '@/lib/i18n';
import { BusinessKpiCard } from '@/components/business/business-kpi-card';

export function InventoryWorkspace({
  products,
  isPharmacy,
  onUpdated
}: {
  products: PosProduct[];
  isPharmacy: boolean;
  onUpdated: () => void;
}) {
  const text = useText();
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('all');
  const [status, setStatus] = useState('all');
  const [view, setView] = useState<'table' | 'grid'>('table');
  const categories = Array.from(
    new Map(
      products.map((product) => [
        product.category_id || product.category?.id || 'uncategorized',
        product.category?.name || 'Uncategorized'
      ])
    ).entries()
  );
  const filtered = products.filter((product) => {
    const text = `${product.name} ${product.sku} ${product.barcode}`.toLowerCase();
    const matchesQuery = text.includes(query.toLowerCase());
    const matchesCategory =
      category === 'all' ||
      (product.category_id || product.category?.id || 'uncategorized') === category;
    const matchesStatus =
      status === 'all' ||
      (status === 'active'
        ? product.is_active && !product.low_stock
        : status === 'low'
          ? product.low_stock &&
            (isPharmacy ? (product.sellable_stock || 0) > 0 : product.stock > 0)
          : (isPharmacy ? product.sellable_stock || 0 : product.stock) === 0);
    return matchesQuery && matchesCategory && matchesStatus;
  });
  const totalValue = products.reduce((sum, product) => sum + product.stock * product.cost_price, 0);
  const lowStock = products.filter(
    (product) =>
      product.low_stock && (isPharmacy ? (product.sellable_stock || 0) > 0 : product.stock > 0)
  ).length;
  const outOfStock = products.filter(
    (product) => (isPharmacy ? product.sellable_stock || 0 : product.stock) === 0
  ).length;
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        <BusinessKpiCard
          label={text('Total SKU', 'Jumla ya SKU')}
          value={String(products.length)}
          helper={`${products.filter((product) => product.is_active).length} ${text('active', 'hai')}`}
          icon={Icons.product}
        />
        <BusinessKpiCard
          label={text('Inventory value', 'Thamani ya stoo')}
          value={money(totalValue)}
          helper={text('At cost value', 'Kwa thamani ya gharama')}
          icon={Icons.product}
        />
        <BusinessKpiCard
          label={text('Low stock', 'Stoo kidogo')}
          value={String(lowStock)}
          helper={
            lowStock
              ? text('Needs restock', 'Inahitaji kuongezwa')
              : text('All above reorder point', 'Zote ziko juu ya kiwango')
          }
          icon={Icons.toastWarning}
        />
        <BusinessKpiCard
          label={text('Out of stock', 'Imeisha stoo')}
          value={String(outOfStock)}
          helper={
            outOfStock
              ? text('Needs restock now', 'Inahitaji kuongezwa sasa')
              : text('Nothing out yet', 'Hakuna iliyoisha')
          }
          icon={Icons.toastInfo}
        />
      </div>
      <Card>
        <CardHeader>
          <div className='flex items-start justify-between gap-3'>
            <div className='min-w-0 flex-1'>
              <CardTitle>{text('Products', 'Bidhaa')}</CardTitle>
              <CardDescription>
                Search, filter, and manage every product in this store.
              </CardDescription>
            </div>
            <div className='flex shrink-0 gap-1'>
              <Button
                size='icon'
                variant={view === 'table' ? 'default' : 'outline'}
                aria-label='Table view'
                onClick={() => setView('table')}
              >
                <Icons.page className='h-4 w-4' />
              </Button>
              <Button
                size='icon'
                variant={view === 'grid' ? 'default' : 'outline'}
                aria-label='Grid view'
                onClick={() => setView('grid')}
              >
                <Icons.dashboard className='h-4 w-4' />
              </Button>
            </div>
          </div>
          <div className='inventory-filter-row grid gap-2 md:grid-cols-[minmax(0,1fr)_160px_160px]'>
            <Input
              className='h-11'
              aria-label='Search inventory'
              placeholder='Search name, SKU, or barcode…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <select
              aria-label='Filter by category'
              className='bg-background h-11 min-w-0 rounded-md border px-3 text-sm'
              value={category}
              onChange={(event) => setCategory(event.target.value)}
            >
              <option value='all'>All categories</option>
              {categories.map(([id, name]) => (
                <option key={id} value={id}>
                  {name}
                </option>
              ))}
            </select>
            <select
              aria-label='Filter by stock status'
              className='bg-background h-11 min-w-0 rounded-md border px-3 text-sm'
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value='all'>All stock</option>
              <option value='active'>Active</option>
              <option value='low'>Low stock</option>
              <option value='out'>Out of stock</option>
            </select>
          </div>
        </CardHeader>
        <CardContent>
          {view === 'table' ? (
            <InventoryTable products={filtered} isPharmacy={isPharmacy} onUpdated={onUpdated} />
          ) : (
            <div className='grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-4'>
              {filtered.map((product) => (
                <Card
                  key={product.id}
                  className='cursor-pointer overflow-hidden transition-shadow hover:shadow-md'
                  onClick={() => router.push(`/inventory/${product.id}`)}
                >
                  <div className='bg-muted/40 flex aspect-[4/3] items-center justify-center text-3xl font-semibold text-primary'>
                    {product.image ? (
                      <img src={product.image} alt='' className='h-full w-full object-cover' />
                    ) : (
                      product.name.slice(0, 1).toUpperCase()
                    )}
                  </div>
                  <CardContent className='space-y-2 p-3'>
                    <p className='truncate font-medium'>{product.name}</p>
                    <p className='text-muted-foreground truncate text-xs'>
                      {product.sku || product.barcode || 'No SKU'}
                    </p>
                    <div className='flex items-center justify-between text-sm'>
                      <span className='font-semibold'>{money(product.selling_price)}</span>
                      <span
                        className={product.low_stock ? 'text-amber-500' : 'text-muted-foreground'}
                      >
                        {isPharmacy
                          ? `${product.sellable_stock || 0} sellable`
                          : `${product.stock} in stock`}
                      </span>
                    </div>
                    {isPharmacy && !!product.expired_stock && (
                      <p className='text-destructive text-xs'>{product.expired_stock} expired</p>
                    )}
                    {isPharmacy && product.next_expiry_date && (
                      <p className='text-muted-foreground text-xs'>
                        Next expiry{' '}
                        {new Date(
                          `${product.next_expiry_date.slice(0, 10)}T00:00:00`
                        ).toLocaleDateString()}
                      </p>
                    )}
                  </CardContent>
                </Card>
              ))}
              {!filtered.length && (
                <p className='text-muted-foreground col-span-full py-8 text-center text-sm'>
                  No products match these filters.
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
