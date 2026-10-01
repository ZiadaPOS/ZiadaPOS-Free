'use client';
import PageContainer from '@/components/layout/page-container';
import { TransactionsLedger } from '@/features/transactions/components/transactions-ledger';
import { useText } from '@/lib/i18n';

export default function Page() {
  const text = useText();
  return (
    <PageContainer
      pageTitle={text('Transactions', 'Miamala')}
      pageDescription={text(
        'A complete record of sales, credits, discounts, refunds, and voids.',
        'Rekodi kamili ya mauzo, mikopo, punguzo, marejesho, na miamala iliyofutwa.'
      )}
    >
      <TransactionsLedger />
    </PageContainer>
  );
}
