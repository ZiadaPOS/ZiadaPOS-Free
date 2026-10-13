'use client';

import PageContainer from '@/components/layout/page-container';
import { ExpenseWorkspace } from '@/features/expenses/components/expense-workspace';
import { useText } from '@/lib/i18n';
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { posApi, unwrapList, type PosExpenseCategory } from '@/lib/pos-api';
import { ExpenseCategoryMenu } from '@/features/expenses/components/expense-category-menu';

export default function ExpensesPage() {
  const text = useText();
  const [showForm, setShowForm] = useState(false);
  const [categories, setCategories] = useState<PosExpenseCategory[]>([]);
  const [category, setCategory] = useState('all');
  useEffect(() => {
    void posApi<unknown>('/expenses/categories')
      .then((result) => setCategories(unwrapList<PosExpenseCategory>(result.data)))
      .catch(() => setCategories([]));
  }, []);
  return (
    <PageContainer
      pageTitle={text('Expenses', 'Gharama')}
      pageDescription={text(
        'Track store expenses and spending trends.',
        'Fuatilia gharama na mwenendo wa matumizi ya tawi.'
      )}
      pageHeaderAction={
        <div className='grid w-full grid-cols-[2fr_3fr] gap-2 sm:flex sm:w-auto'>
          <ExpenseCategoryMenu
            categories={categories.map((item) => item.name)}
            selected={category}
            onSelect={setCategory}
          />
          <Button className='w-full sm:w-auto' onClick={() => setShowForm((value) => !value)}>
            {showForm ? text('Close', 'Funga') : text('Add expense', 'Ongeza gharama')}
          </Button>
        </div>
      }
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <ExpenseWorkspace
          showForm={showForm}
          onShowFormChange={setShowForm}
          categories={categories.map((item) => item.name)}
          category={category}
          onCategoryChange={setCategory}
        />
      </div>
    </PageContainer>
  );
}
