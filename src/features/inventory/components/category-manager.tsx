'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList, type PosCategory } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useEffect, useRef, useState } from 'react';

export function CategoryManager() {
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [name, setName] = useState('');
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  const panelRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    posApi<unknown>('/inventory/categories')
      .then((result) => setCategories(unwrapList<PosCategory>(result.data)))
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load categories.')
      );
  }, [reload]);
  useEffect(() => {
    if (!open) return;
    function closeOnOutsidePointer(event: PointerEvent) {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener('pointerdown', closeOnOutsidePointer);
    return () => document.removeEventListener('pointerdown', closeOnOutsidePointer);
  }, [open]);
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      await posApi<PosCategory>('/inventory/categories', {
        method: 'POST',
        body: JSON.stringify({ name })
      });
      setName('');
      setReload((value) => value + 1);
      setError('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create category.');
    }
  }
  const visibleCategories = categories.filter((category) =>
    category.name.toLowerCase().includes(query.toLowerCase())
  );
  return (
    <div ref={panelRef} className='relative'>
      <Button variant='outline' onClick={() => setOpen((value) => !value)}>
        Categories ({categories.length})
      </Button>
      {open && (
        <Card className='absolute right-0 top-11 z-30 w-[min(22rem,calc(100vw-2rem))] shadow-xl'>
          <CardHeader className='flex flex-row items-center justify-between space-y-0'>
            <CardTitle>Product categories</CardTitle>
            <Button
              type='button'
              variant='ghost'
              size='icon'
              aria-label='Close categories'
              onClick={() => setOpen(false)}
            >
              <Icons.close className='h-4 w-4' />
            </Button>
          </CardHeader>
          <CardContent className='space-y-3'>
            <Input
              aria-label='Search categories'
              placeholder='Search categories…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <div className='max-h-40 space-y-1 overflow-y-auto text-sm'>
              {visibleCategories.map((category) => (
                <p key={category.id} className='rounded border px-2 py-1'>
                  {category.name}
                </p>
              ))}
              {!visibleCategories.length && (
                <p className='text-muted-foreground py-3 text-center text-xs'>
                  No categories found.
                </p>
              )}
            </div>
            <form onSubmit={create} className='flex gap-2'>
              <Input
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder='New category'
                required
              />
              <Button size='sm'>Add</Button>
            </form>
            {error && (
              <p role='alert' className='text-destructive text-xs'>
                {error}
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
