'use client';

import { useCallback, useEffect, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { useText } from '@/lib/i18n';
import { posApi } from '@/lib/pos-api';

type OrganisationSettings = {
  sms_configured?: boolean;
  sms_sender_id?: string;
};
type Balance = { balance?: number; configured?: boolean };

export default function SmsPage() {
  const text = useText();
  const [organisation, setOrganisation] = useState<OrganisationSettings>({});
  const [balance, setBalance] = useState<Balance>({});
  const [apiKey, setApiKey] = useState('');
  const [senderId, setSenderId] = useState('');
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const result = await posApi<OrganisationSettings>('/accounts/organisation');
      setOrganisation(result.data);
      setSenderId(result.data.sms_sender_id || '');
      try {
        const balanceResult = await posApi<Balance>('/accounts/sms-balance');
        setBalance(balanceResult.data);
      } catch {
        setBalance({ configured: false });
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load SendAfrica settings.');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveSettings(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError('');
    setNotice('');
    const payload: Record<string, string> = { sms_sender_id: senderId.trim() };
    if (apiKey.trim()) payload.sendafrica_api_key = apiKey.trim();
    try {
      await posApi<unknown>('/accounts/organisation', {
        method: 'PATCH',
        body: JSON.stringify(payload)
      });
      setApiKey('');
      setNotice(
        text(
          'SendAfrica settings saved. The API key is not displayed again.',
          'Mipangilio ya SendAfrica imehifadhiwa. API key haitaonyeshwa tena.'
        )
      );
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not save SendAfrica settings.');
    } finally {
      setSaving(false);
    }
  }

  async function removeKey() {
    setSaving(true);
    setError('');
    setNotice('');
    try {
      await posApi<unknown>('/accounts/organisation', {
        method: 'PATCH',
        body: JSON.stringify({ sendafrica_api_key: '' })
      });
      setNotice(text('Saved SendAfrica key removed.', 'API key ya SendAfrica imeondolewa.'));
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not remove the key.');
    } finally {
      setSaving(false);
    }
  }

  async function checkBalance() {
    setChecking(true);
    setError('');
    try {
      const result = await posApi<Balance>('/accounts/sms-balance');
      setBalance(result.data);
      if (!result.data.configured)
        setNotice(
          text('Connect your SendAfrica key first.', 'Unganisha API key yako ya SendAfrica kwanza.')
        );
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'SendAfrica connection check failed.');
    } finally {
      setChecking(false);
    }
  }

  return (
    <PageContainer
      pageTitle={text('SMS messaging', 'Ujumbe wa SMS')}
      pageDescription={text(
        'Connect your organization’s SendAfrica account and approved sender ID.',
        'Unganisha akaunti ya SendAfrica ya shirika lako na sender ID iliyoidhinishwa.'
      )}
      pageHeaderAction={
        <Button variant='outline' onClick={() => void load()}>
          <Icons.refresh className='size-4' /> {text('Refresh', 'Onyesha upya')}
        </Button>
      }
    >
      <div className='grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(280px,380px)]'>
        <Card>
          <CardHeader>
            <CardTitle>{text('Connect SendAfrica', 'Unganisha SendAfrica')}</CardTitle>
            <CardDescription>
              {text(
                'Paste the API key from your SendAfrica account. It is sent only to the authenticated server, encrypted before database storage, and never returned to the browser after saving.',
                'Bandika API key ya akaunti yako ya SendAfrica. Inatumwa kwa seva iliyoidhinishwa tu, inasimbwa kabla ya kuhifadhiwa, na hairudishwi kwa kivinjari baada ya kuhifadhi.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <div className='flex items-center gap-2'>
              <Badge variant={organisation.sms_configured ? 'secondary' : 'outline'}>
                {organisation.sms_configured
                  ? text('API key saved', 'API key imehifadhiwa')
                  : text('Not connected', 'Haijaunganishwa')}
              </Badge>
              {organisation.sms_configured && (
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => void removeKey()}
                  disabled={saving}
                >
                  {text('Remove key', 'Ondoa key')}
                </Button>
              )}
            </div>
            <form onSubmit={saveSettings} className='space-y-4'>
              <label className='block space-y-1 text-sm font-medium'>
                {text('SendAfrica API key', 'API key ya SendAfrica')}
                <Input
                  type='password'
                  value={apiKey}
                  onChange={(event) => setApiKey(event.target.value)}
                  placeholder={
                    organisation.sms_configured
                      ? text(
                          'Saved — paste a new key to rotate',
                          'Imehifadhiwa — bandika key mpya kubadilisha'
                        )
                      : text('Paste your API key', 'Bandika API key yako')
                  }
                  autoComplete='new-password'
                  spellCheck={false}
                />
              </label>
              <label className='block space-y-1 text-sm font-medium'>
                {text('Registered sender ID (optional)', 'Sender ID iliyosajiliwa (si lazima)')}
                <Input
                  value={senderId}
                  onChange={(event) => setSenderId(event.target.value)}
                  maxLength={20}
                  autoComplete='off'
                  placeholder={text(
                    'Use the account default, or enter an approved sender ID',
                    'Tumia sender ID ya akaunti au weka iliyoidhinishwa'
                  )}
                />
              </label>
              <p className='text-muted-foreground text-xs'>
                {text(
                  'Sender IDs must be registered and approved in SendAfrica. Approval and allowed formats depend on the destination country; an unapproved ID may be rejected.',
                  'Sender ID lazima isajiliwe na kuidhinishwa SendAfrica. Uidhinishaji na muundo hutegemea nchi ya mpokeaji; jina lisiloidhinishwa linaweza kukataliwa.'
                )}
              </p>
              <Button type='submit' isLoading={saving}>
                <Icons.check className='size-4' /> {text('Save settings', 'Hifadhi mipangilio')}
              </Button>
            </form>
            {error && (
              <p role='alert' className='text-destructive text-sm'>
                {error}
              </p>
            )}
            {notice && (
              <p role='status' className='text-muted-foreground text-sm'>
                {notice}
              </p>
            )}
          </CardContent>
        </Card>
        <Card className='h-fit'>
          <CardHeader>
            <CardTitle>{text('Provider balance', 'Salio la mtoa huduma')}</CardTitle>
            <CardDescription>
              {text(
                'This checks the SendAfrica balance for this organization’s key. It does not send a test SMS.',
                'Hii hukagua salio la SendAfrica kwa key ya shirika. Haitumi SMS ya majaribio.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            <p className='text-3xl font-semibold tabular-nums'>
              {balance.configured ? (balance.balance ?? 0) : '—'}
            </p>
            <Button
              type='button'
              variant='outline'
              onClick={() => void checkBalance()}
              isLoading={checking}
              disabled={!organisation.sms_configured}
            >
              {text('Check connection and balance', 'Angalia muunganisho na salio')}
            </Button>
            <p className='text-muted-foreground text-xs'>
              {text(
                'Provider acceptance means SendAfrica accepted the request; it does not confirm delivery to the handset. Delivery callbacks are not configured in this app.',
                'Kukubaliwa na mtoa huduma kunamaanisha SendAfrica imepokea ombi; hakuthibitishi kufikishwa kwa simu. Callback za delivery hazijaunganishwa kwenye programu hii.'
              )}
            </p>
            <a
              href='https://sendafrica.online'
              target='_blank'
              rel='noreferrer'
              className='text-primary inline-flex items-center gap-1 text-sm underline-offset-4 hover:underline'
            >
              {text('Open SendAfrica', 'Fungua SendAfrica')}{' '}
              <Icons.externalLink className='size-3.5' />
            </a>
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  );
}
