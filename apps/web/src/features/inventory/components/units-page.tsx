'use client';

import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { BusinessKpiCard } from '@/components/business/business-kpi-card';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList, type PosUnit } from '@/lib/pos-api';
import { InventoryNameDialog } from '@/features/inventory/components/inventory-name-dialog';

export default function UnitsPage() {
  const [units, setUnits] = useState<PosUnit[]>([]);
  const [editingUnit, setEditingUnit] = useState<PosUnit | null>(null);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isAvailable, setIsAvailable] = useState(true);
  const [query, setQuery] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [showCreate, setShowCreate] = useState(false);

  const load = useCallback(async () => {
    try {
      setError('');
      const result = await posApi<unknown>('/inventory/units');
      setUnits(unwrapList<PosUnit>(result.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load units.');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setError('');
    try {
      await posApi(editingUnit ? `/inventory/units/${editingUnit.id}` : '/inventory/units', {
        method: editingUnit ? 'PATCH' : 'POST',
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim(),
          ...(editingUnit ? { is_active: isAvailable } : {})
        })
      });
      setName('');
      setDescription('');
      setEditingUnit(null);
      setShowCreate(false);
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create unit.');
    } finally {
      setBusy(false);
    }
  }

  async function remove(unit: PosUnit) {
    if (!window.confirm(`Delete “${unit.name}”?`)) return;
    try {
      setError('');
      await posApi(`/inventory/units/${unit.id}`, { method: 'DELETE' });
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not delete unit.');
    }
  }

  function setCreateDialogOpen(open: boolean) {
    setShowCreate(open);
    setError('');
    setName('');
    setDescription('');
    setIsAvailable(true);
    setEditingUnit(null);
  }

  function startEditing(unit: PosUnit) {
    setError('');
    setEditingUnit(unit);
    setName(unit.name);
    setDescription(unit.description ?? '');
    setIsAvailable(unit.is_active !== false);
    setShowCreate(true);
  }

  const filtered = useMemo(
    () => units.filter((unit) => unit.name.toLowerCase().includes(query.toLowerCase())),
    [units, query]
  );

  return (
    <PageContainer
      pageTitle='Units'
      pageDescription='Manage the units used to describe products and medicines.'
      pageHeaderAction={
        <Button className='w-full sm:w-auto' onClick={() => setCreateDialogOpen(true)}>
          Add unit
        </Button>
      }
    >
      <div className='space-y-5'>
        <div className='grid grid-cols-2 gap-3 xl:grid-cols-3'>
          <BusinessKpiCard
            label='Total units'
            value={String(units.length)}
            helper='Available to your business'
            icon={Icons.product}
          />
          <BusinessKpiCard
            label='Active units'
            value={String(units.filter((unit) => unit.is_active !== false).length)}
            helper='Available in product forms'
            icon={Icons.check}
          />
          <BusinessKpiCard
            label='Search results'
            value={String(filtered.length)}
            helper='Matching units'
            icon={Icons.search}
          />
        </div>
        {error && !showCreate && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}
        <Card>
          <CardHeader>
            <CardTitle>Manage units</CardTitle>
            <CardDescription>
              Add units such as tablet, capsule, bottle, or piece. They appear in the medicine form.
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <Input
              aria-label='Search units'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder='Search units…'
            />
            <div className='divide-y rounded-lg border'>
              {filtered.map((unit) => (
                <div key={unit.id} className='flex items-center justify-between gap-3 p-3'>
                  <div className='min-w-0'>
                    <p className='truncate font-medium'>{unit.name}</p>
                    <p className='text-muted-foreground text-xs'>
                      {unit.is_default ? 'Default · ' : ''}
                      {unit.is_active === false ? 'Inactive' : 'Available for products'}
                      {' · '}
                      {unit.usage_count ?? 0} products use this unit
                    </p>
                  </div>
                  <div className='flex shrink-0 items-center gap-1'>
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      aria-label={`Edit ${unit.name}`}
                      onClick={() => startEditing(unit)}
                    >
                      <Icons.edit className='size-4' />
                    </Button>
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon'
                      aria-label={`Delete ${unit.name}`}
                      onClick={() => void remove(unit)}
                    >
                      <Icons.trash className='text-destructive size-4' />
                    </Button>
                  </div>
                </div>
              ))}
              {!filtered.length && (
                <p className='text-muted-foreground p-6 text-center text-sm'>
                  {units.length
                    ? 'No units found.'
                    : 'No units yet. Add one to use it on products.'}
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
      <InventoryNameDialog
        open={showCreate}
        onOpenChange={setCreateDialogOpen}
        title={editingUnit ? 'Edit unit' : 'Add unit'}
        description={
          editingUnit
            ? 'Rename this unit. Existing products and medicine profiles will be updated.'
            : 'Create a reusable unit for your products and medicines.'
        }
        fieldLabel='Unit name'
        placeholder='e.g. Tablet'
        submitLabel={editingUnit ? 'Save changes' : 'Add unit'}
        name={name}
        onNameChange={setName}
        details={description}
        detailsLabel='Description (optional)'
        detailsPlaceholder='e.g. Sold as individual tablets'
        onDetailsChange={setDescription}
        available={isAvailable}
        onAvailableChange={editingUnit ? setIsAvailable : undefined}
        onSubmit={create}
        busy={busy}
        error={showCreate ? error : undefined}
      />
    </PageContainer>
  );
}
