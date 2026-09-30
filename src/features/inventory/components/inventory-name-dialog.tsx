'use client';

import { FormEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { FormDialog } from '@/components/ui/form-dialog';

interface InventoryNameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  fieldLabel: string;
  placeholder: string;
  submitLabel: string;
  name: string;
  onNameChange: (name: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  busy: boolean;
  error?: string;
  typeOptions?: { value: string; label: string }[];
  selectedType?: string;
  onTypeChange?: (value: string) => void;
  details?: string;
  detailsLabel?: string;
  detailsPlaceholder?: string;
  onDetailsChange?: (value: string) => void;
  available?: boolean;
  onAvailableChange?: (value: boolean) => void;
}

export function InventoryNameDialog({
  open,
  onOpenChange,
  title,
  description,
  fieldLabel,
  placeholder,
  submitLabel,
  name,
  onNameChange,
  onSubmit,
  busy,
  error,
  typeOptions,
  selectedType = '',
  onTypeChange,
  details,
  detailsLabel,
  detailsPlaceholder,
  onDetailsChange,
  available,
  onAvailableChange
}: InventoryNameDialogProps) {
  return (
    <FormDialog open={open} onOpenChange={onOpenChange} title={title} description={description}>
      <form onSubmit={onSubmit} className='mt-5 space-y-4'>
        {typeOptions?.length ? (
          <label className='block text-sm font-semibold'>
            Category type
            <select
              value={selectedType}
              onChange={(event) => onTypeChange?.(event.target.value)}
              className='bg-background mt-2 h-11 w-full rounded-lg border border-foreground/20 px-3 font-normal'
              required
            >
              <option value='' disabled>
                Select category type
              </option>
              {typeOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
        ) : null}
        <label className='block text-sm font-semibold'>
          {fieldLabel}
          <Input
            autoFocus
            value={name}
            onChange={(event) => onNameChange(event.target.value)}
            placeholder={placeholder}
            className='mt-2 h-11 rounded-lg px-3 font-normal'
            required
          />
        </label>
        {detailsLabel && onDetailsChange ? (
          <label className='block text-sm font-semibold'>
            {detailsLabel}
            <Input
              value={details ?? ''}
              onChange={(event) => onDetailsChange(event.target.value)}
              placeholder={detailsPlaceholder}
              className='mt-2 h-11 rounded-lg px-3 font-normal'
            />
          </label>
        ) : null}
        {onAvailableChange ? (
          <label className='flex items-center gap-2 text-sm font-medium'>
            <input
              type='checkbox'
              checked={available ?? true}
              onChange={(event) => onAvailableChange(event.target.checked)}
            />
            Available for new products
          </label>
        ) : null}
        {error && (
          <p
            role='alert'
            className='rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive'
          >
            {error}
          </p>
        )}
        <Button
          type='submit'
          disabled={busy}
          className='mt-2 h-12 w-full rounded-lg px-4 text-sm font-semibold'
        >
          {busy ? 'Saving…' : submitLabel}
        </Button>
      </form>
    </FormDialog>
  );
}
