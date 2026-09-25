'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { money, type PosProduct } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { InventoryTable } from '@/features/inventory/components/inventory-table';

export function InventoryWorkspace({
  products,
  onUpdated
}: {
  products: PosProduct[];
  onUpdated: () => void;
}) {
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
          ? product.low_stock && product.stock > 0
          : product.stock === 0);
    return matchesQuery && matchesCategory && matchesStatus;
  });
  const totalValue = products.reduce((sum, product) => sum + product.stock * product.cost_price, 0);
  const lowStock = products.filter((product) => product.low_stock && product.stock > 0).length;
  const outOfStock = products.filter((product) => product.stock === 0).length;
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          [
            'Total SKU',
            String(products.length),
            `${products.filter((product) => product.is_active).length} active`
          ],
          ['Inventory value', money(totalValue), 'At cost value'],
          ['Low stock', String(lowStock), lowStock ? 'Needs restock' : 'All above reorder point'],
          ['Out of stock', String(outOfStock), outOfStock ? 'Needs restock now' : 'Nothing out yet']
        ].map(([label, value, detail]) => (
          <Card key={label} className='min-w-0'>
            <CardHeader className='p-3 pb-2 sm:p-6 sm:pb-2'>
              <CardDescription className='truncate text-xs sm:text-sm'>{label}</CardDescription>
              <CardTitle className='truncate text-lg sm:text-2xl'>{value}</CardTitle>
            </CardHeader>
            <CardContent className='truncate p-3 pt-0 text-[11px] text-muted-foreground sm:p-6 sm:pt-0 sm:text-xs'>
              {detail}
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <div>
              <CardTitle>Products</CardTitle>
              <CardDescription>
                Search, filter, and manage every product in this store.
              </CardDescription>
            </div>
            <div className='inventory-view-controls flex gap-1'>
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
            <div className='inventory-mobile-view-controls inventory-view-controls flex gap-1'>
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
        </CardHeader>
        <CardContent>
          {view === 'table' ? (
            <InventoryTable products={filtered} onUpdated={onUpdated} />
          ) : (
            <div className='grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-4'>
              {filtered.map((product) => (
                <Card
                  key={product.id}
                  className='cursor-pointer overflow-hidden transition-shadow hover:shadow-md'
                  onClick={() => router.push(`/dashboard/inventory/${product.id}`)}
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
                        className={
                          product.stock === 0
                            ? 'text-destructive'
                            : product.low_stock
                              ? 'text-amber-500'
                              : 'text-muted-foreground'
                        }
                      >
                        {product.stock} in stock
                      </span>
                    </div>
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
