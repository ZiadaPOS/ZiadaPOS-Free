'use client';

import PageContainer from '@/components/layout/page-container';
import { ExpenseWorkspace } from '@/features/expenses/components/expense-workspace';
import { useText } from '@/lib/i18n';

export default function ExpensesPage() {
  const text = useText();
  return (
    <PageContainer
      pageTitle={text('Expenses', 'Gharama')}
      pageDescription={text(
        'Track store expenses and spending trends.',
        'Fuatilia gharama na mwenendo wa matumizi ya tawi.'
      )}
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <ExpenseWorkspace />
      </div>
    </PageContainer>
  );
}
