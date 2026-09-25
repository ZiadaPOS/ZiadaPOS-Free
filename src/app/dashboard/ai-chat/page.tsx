import PageContainer from '@/components/layout/page-container';
import { ZiadaAiChat } from '@/features/ai-chat/components/ziada-ai-chat';

export const metadata = {
  title: 'Dashboard: AI Chat'
};

export default function Page() {
  return (
    <PageContainer>
      <ZiadaAiChat />
    </PageContainer>
  );
}
