'use client';

import PageContainer from '@/components/layout/page-container';
import { ExpenseWorkspace } from '@/features/expenses/components/expense-workspace';

export default function ExpensesPage() {
  return (
    <PageContainer pageTitle='Expenses' pageDescription='Track store expenses and spending trends.'>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        <ExpenseWorkspace />
      </div>
    </PageContainer>
  );
}
