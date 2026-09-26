'use client';

import PageContainer from '@/components/layout/page-container';
import { ExpenseWorkspace } from '@/features/expenses/components/expense-workspace';
import { useText } from '@/lib/i18n';
import { useState } from 'react';
import { Button } from '@/components/ui/button';

export default function ExpensesPage() {
  const text = useText();
  const [showForm, setShowForm] = useState(false);
  return (
    <PageContainer
      pageTitle={text('Expenses', 'Gharama')}
      pageDescription={text(
        'Track store expenses and spending trends.',
        'Fuatilia gharama na mwenendo wa matumizi ya tawi.'
      )}
      pageHeaderAction={
        <Button className='w-full sm:w-auto' onClick={() => setShowForm((value) => !value)}>
          {showForm ? text('Close', 'Funga') : text('Add expense', 'Ongeza gharama')}
        </Button>
      }
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <ExpenseWorkspace showForm={showForm} onShowFormChange={setShowForm} />
      </div>
    </PageContainer>
  );
}
