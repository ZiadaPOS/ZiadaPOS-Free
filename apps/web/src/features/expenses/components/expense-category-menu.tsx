'use client';

import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Icons } from '@/components/icons';

export function ExpenseCategoryMenu({
  categories,
  selected,
  onSelect
}: {
  categories: string[];
  selected: string;
  onSelect: (category: string) => void;
}) {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button type='button' variant='outline' className='w-full min-w-0 sm:w-auto'>
            <span className='truncate'>Categories ({categories.length})</span>
          </Button>
        }
      />
      <PopoverContent align='end' className='w-[min(22rem,calc(100vw-2rem))] p-0'>
        <Command>
          <CommandInput placeholder='Search expense categories…' />
          <CommandList>
            <CommandEmpty>No categories found.</CommandEmpty>
            <CommandGroup heading='Expense categories'>
              <CommandItem value='all categories' onSelect={() => onSelect('all')}>
                <Icons.check
                  className={`mr-2 size-4 ${selected === 'all' ? 'opacity-100' : 'opacity-0'}`}
                />
                All categories
              </CommandItem>
              {categories.map((category) => (
                <CommandItem key={category} value={category} onSelect={() => onSelect(category)}>
                  <Icons.check
                    className={`mr-2 size-4 ${selected === category ? 'opacity-100' : 'opacity-0'}`}
                  />
                  {category}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
