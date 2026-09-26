'use client';

import { BusinessKpiCard } from '@/components/business/business-kpi-card';
import PageContainer from '@/components/layout/page-container';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList, type PosCategory } from '@/lib/pos-api';
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';

export default function CategoriesPage() {
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [name, setName] = useState('');
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      setError('');
      const result = await posApi<unknown>('/inventory/categories');
      setCategories(unwrapList<PosCategory>(result.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load categories.');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    try {
      await posApi('/inventory/categories', {
        method: 'POST',
        body: JSON.stringify({ name: name.trim() })
      });
      setName('');
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create category.');
    } finally {
      setBusy(false);
    }
  }

  async function remove(category: PosCategory) {
    if (!window.confirm(`Delete “${category.name}”?`)) return;
    try {
      setError('');
      await posApi(`/inventory/categories/${category.id}`, { method: 'DELETE' });
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not delete category.');
    }
  }

  const filtered = useMemo(
    () =>
      categories.filter((category) => category.name.toLowerCase().includes(query.toLowerCase())),
    [categories, query]
  );
  const assignedProducts = categories.reduce(
    (sum, category) => sum + (category.product_count ?? 0),
    0
  );
  const emptyCategories = categories.filter((category) => !category.product_count).length;

  return (
    <PageContainer
      pageTitle='Categories'
      pageDescription='Organize products into clear, manageable groups.'
    >
      <div className='space-y-5'>
        <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
          <BusinessKpiCard
            label='Total categories'
            value={String(categories.length)}
            helper='Product groups'
            icon={Icons.folder}
          />
          <BusinessKpiCard
            label='Assigned products'
            value={String(assignedProducts)}
            helper='Products in categories'
            icon={Icons.product}
          />
          <BusinessKpiCard
            label='Empty categories'
            value={String(emptyCategories)}
            helper='Ready for products'
            icon={Icons.info}
          />
          <BusinessKpiCard
            label='Search results'
            value={String(filtered.length)}
            helper='Matching categories'
            icon={Icons.search}
          />
        </div>

        {error && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Manage categories</CardTitle>
            <CardDescription>Create, find, and remove product categories.</CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <form onSubmit={create} className='flex flex-col gap-2 sm:flex-row'>
              <Input
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder='New category name'
                required
              />
              <Button type='submit' disabled={busy} className='sm:w-auto'>
                {busy ? 'Adding…' : 'Add category'}
              </Button>
            </form>
            <Input
              aria-label='Search categories'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='Search categories…'
            />
            <div className='divide-y rounded-lg border'>
              {filtered.map((category) => (
                <div key={category.id} className='flex items-center justify-between gap-3 p-3'>
                  <div className='min-w-0'>
                    <p className='truncate font-medium'>{category.name}</p>
                    <p className='text-muted-foreground text-xs'>
                      {category.product_count ?? 0} products
                    </p>
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    aria-label={`Delete ${category.name}`}
                    onClick={() => void remove(category)}
                  >
                    <Icons.trash className='text-destructive size-4' />
                  </Button>
                </div>
              ))}
              {!filtered.length && (
                <p className='text-muted-foreground p-6 text-center text-sm'>
                  No categories found.
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  );
}
