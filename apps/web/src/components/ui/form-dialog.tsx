'use client';

import { ReactNode } from 'react';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

interface FormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  children: ReactNode;
  size?: 'compact' | 'wide';
  className?: string;
}

export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  size = 'compact',
  className
}: FormDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        showCloseButton={false}
        overlayClassName='bg-black/30'
        className={cn(
          'block max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] overflow-y-auto rounded-2xl border border-border bg-card p-6 shadow-2xl',
          size === 'wide' ? 'max-w-2xl sm:max-w-2xl' : 'max-w-md sm:max-w-md',
          className
        )}
      >
        <DialogHeader className='flex-row items-start justify-between gap-4 space-y-0'>
          <div className='min-w-0 space-y-1'>
            <DialogTitle className='text-lg font-semibold'>{title}</DialogTitle>
            <DialogDescription>{description}</DialogDescription>
          </div>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            className='-mt-1 -mr-1 shrink-0 text-muted-foreground'
            aria-label='Close dialog'
            onClick={() => onOpenChange(false)}
          >
            <Icons.close className='size-5' />
          </Button>
        </DialogHeader>
        {children}
      </DialogContent>
    </Dialog>
  );
}
