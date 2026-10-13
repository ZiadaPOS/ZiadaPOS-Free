'use client';

import { BusinessKpiCard } from '@/components/business/business-kpi-card';
import PageContainer from '@/components/layout/page-container';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList, type PosCategory, type PosExpenseCategory } from '@/lib/pos-api';
import { InventoryNameDialog } from '@/features/inventory/components/inventory-name-dialog';
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';

export default function CategoriesPage() {
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [expenseCategories, setExpenseCategories] = useState<PosExpenseCategory[]>([]);
  const [categoryType, setCategoryType] = useState<'inventory' | 'expenses'>('inventory');
  const [editingCategory, setEditingCategory] = useState<PosCategory | PosExpenseCategory | null>(
    null
  );
  const [createType, setCreateType] = useState('');
  const [name, setName] = useState('');
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [showCreate, setShowCreate] = useState(false);

  const load = useCallback(async () => {
    setError('');
    const [inventoryResult, expenseResult] = await Promise.allSettled([
      posApi<unknown>('/inventory/categories'),
      posApi<unknown>('/expenses/categories')
    ]);
    if (inventoryResult.status === 'fulfilled') {
      setCategories(unwrapList<PosCategory>(inventoryResult.value.data));
    } else {
      setError(
        inventoryResult.reason instanceof Error
          ? inventoryResult.reason.message
          : 'Could not load inventory categories.'
      );
    }
    if (expenseResult.status === 'fulfilled') {
      setExpenseCategories(unwrapList<PosExpenseCategory>(expenseResult.value.data));
    } else {
      setError(
        (current) =>
          current ||
          (expenseResult.reason instanceof Error
            ? expenseResult.reason.message
            : 'Could not load expense categories.')
      );
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const targetType = editingCategory ? categoryType : createType;
    if (!name.trim() || !targetType) return;
    setBusy(true);
    try {
      const path = targetType === 'inventory' ? '/inventory/categories' : '/expenses/categories';
      const body =
        editingCategory && targetType === 'inventory'
          ? {
              name: name.trim(),
              description: (editingCategory as PosCategory).description ?? '',
              sort_order: (editingCategory as PosCategory).sort_order ?? 0
            }
          : { name: name.trim() };
      await posApi(editingCategory ? `${path}/${editingCategory.id}` : path, {
        method: editingCategory ? 'PATCH' : 'POST',
        body: JSON.stringify(body)
      });
      setName('');
      setCreateType('');
      setEditingCategory(null);
      await load();
      setShowCreate(false);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create category.');
    } finally {
      setBusy(false);
    }
  }

  async function remove(category: PosCategory | PosExpenseCategory) {
    if (!window.confirm(`Delete “${category.name}”?`)) return;
    try {
      setError('');
      await posApi(
        `${categoryType === 'inventory' ? '/inventory/categories' : '/expenses/categories'}/${category.id}`,
        { method: 'DELETE' }
      );
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not delete category.');
    }
  }

  function setCreateDialogOpen(open: boolean) {
    setShowCreate(open);
    setError('');
    if (open) {
      setEditingCategory(null);
      setName('');
      setCreateType('');
    } else {
      setName('');
      setCreateType('');
      setEditingCategory(null);
    }
  }

  function startEditing(category: PosCategory | PosExpenseCategory) {
    setError('');
    setName(category.name);
    setEditingCategory(category);
    setShowCreate(true);
  }

  const visibleCategories = categoryType === 'inventory' ? categories : expenseCategories;
  const usageCount = (category: PosCategory | PosExpenseCategory) =>
    categoryType === 'inventory'
      ? ((category as PosCategory).product_count ?? 0)
      : ((category as PosExpenseCategory).expense_count ?? 0);
  const filtered = useMemo(
    () =>
      visibleCategories.filter((category) =>
        category.name.toLowerCase().includes(query.toLowerCase())
      ),
    [visibleCategories, query]
  );
  const assignedProducts = visibleCategories.reduce(
    (sum, category) => sum + usageCount(category),
    0
  );
  const emptyCategories = visibleCategories.filter((category) => !usageCount(category)).length;

  return (
    <PageContainer
      pageTitle='Categories'
      pageDescription={
        categoryType === 'inventory'
          ? 'Organize products into clear, manageable groups.'
          : 'Organize business expenses into clear, manageable groups.'
      }
      pageHeaderAction={
        <Button className='w-full sm:w-auto' onClick={() => setCreateDialogOpen(true)}>
          Add category
        </Button>
      }
    >
      <div className='space-y-5'>
        <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
          <BusinessKpiCard
            label={categoryType === 'inventory' ? 'Inventory categories' : 'Expense categories'}
            value={String(visibleCategories.length)}
            helper={categoryType === 'inventory' ? 'Product groups' : 'Expense groups'}
            icon={Icons.folder}
          />
          <BusinessKpiCard
            label={categoryType === 'inventory' ? 'Assigned products' : 'Categorized expenses'}
            value={String(assignedProducts)}
            helper={
              categoryType === 'inventory' ? 'Products in categories' : 'Expenses by category'
            }
            icon={Icons.product}
          />
          <BusinessKpiCard
            label={categoryType === 'inventory' ? 'Empty categories' : 'Unused categories'}
            value={String(emptyCategories)}
            helper={categoryType === 'inventory' ? 'Ready for products' : 'Ready for expenses'}
            icon={Icons.info}
          />
          <BusinessKpiCard
            label='Search results'
            value={String(filtered.length)}
            helper='Matching categories'
            icon={Icons.search}
          />
        </div>

        {error && !showCreate && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}

        <Card>
          <CardHeader className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
            <div className='space-y-1'>
              <CardTitle>Manage categories</CardTitle>
              <CardDescription>
                {categoryType === 'inventory'
                  ? 'Create, find, and remove product categories.'
                  : 'Create, find, and remove expense categories.'}
              </CardDescription>
            </div>
            <div
              className='grid w-full grid-cols-2 rounded-lg border p-1 sm:flex sm:w-auto'
              role='tablist'
              aria-label='Category type'
            >
              {(['inventory', 'expenses'] as const).map((type) => (
                <Button
                  key={type}
                  type='button'
                  size='sm'
                  className='w-full sm:w-auto'
                  variant={categoryType === type ? 'default' : 'ghost'}
                  role='tab'
                  aria-selected={categoryType === type}
                  onClick={() => setCategoryType(type)}
                >
                  {type === 'inventory' ? 'Inventory' : 'Expenses'}
                </Button>
              ))}
            </div>
          </CardHeader>
          <CardContent className='space-y-4'>
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
                      {'is_default' in category && category.is_default ? 'Default · ' : ''}
                      {usageCount(category)}{' '}
                      {categoryType === 'inventory' ? 'products' : 'expenses'}
                    </p>
                  </div>
                  <div className='flex shrink-0 items-center gap-1'>
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      aria-label={`Edit ${category.name}`}
                      onClick={() => startEditing(category)}
                    >
                      <Icons.edit className='size-4' />
                    </Button>
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
      <InventoryNameDialog
        open={showCreate}
        onOpenChange={setCreateDialogOpen}
        title={editingCategory ? 'Edit category' : 'Add category'}
        description={
          editingCategory
            ? 'Update this category name. Existing items will keep their category.'
            : 'Choose where this category will be used, then give it a name.'
        }
        fieldLabel='Category name'
        placeholder='e.g. Personal care'
        submitLabel={editingCategory ? 'Save changes' : 'Add category'}
        name={name}
        onNameChange={setName}
        onSubmit={create}
        busy={busy}
        error={showCreate ? error : undefined}
        typeOptions={
          editingCategory
            ? undefined
            : [
                { value: 'inventory', label: 'Inventory' },
                { value: 'expenses', label: 'Expenses' }
              ]
        }
        selectedType={editingCategory ? categoryType : createType}
        onTypeChange={setCreateType}
      />
    </PageContainer>
  );
}
