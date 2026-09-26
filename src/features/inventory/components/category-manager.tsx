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
    <div ref={panelRef} className='relative w-full sm:w-auto'>
      <Button
        className='w-full sm:w-auto'
        variant='outline'
        onClick={() => setOpen((value) => !value)}
      >
        Categories ({categories.length})
      </Button>
      {open && (
        <Card className='fixed top-20 right-2 left-2 z-30 max-h-[calc(100svh-6rem)] w-auto overflow-y-auto shadow-xl sm:absolute sm:top-11 sm:right-0 sm:left-auto sm:max-h-none sm:w-[min(22rem,calc(100vw-2rem))]'>
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
          <CardContent className='flex flex-col gap-3'>
            <form onSubmit={create} className='flex gap-2'>
              <Input
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder='New category'
                required
              />
              <Button size='sm'>Add</Button>
            </form>
            <Input
              aria-label='Search categories'
              placeholder='Search categories…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <div className='max-h-48 space-y-1 overflow-y-auto pr-1 text-sm'>
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
