import PageContainer from '@/components/layout/page-container';
import { TransactionsLedger } from '@/features/transactions/components/transactions-ledger';

export const metadata = { title: 'Dashboard: Transactions' };

export default function Page() {
  return (
    <PageContainer
      pageTitle='Transactions'
      pageDescription='A complete record of sales, credits, discounts, refunds, and voids.'
    >
      <TransactionsLedger />
    </PageContainer>
  );
}
