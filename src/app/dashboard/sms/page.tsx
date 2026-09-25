'use client';

import { useCallback, useEffect, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Icons } from '@/components/icons';
import { useText } from '@/lib/i18n';
import { posApi, unwrapList } from '@/lib/pos-api';

type RecordValue = Record<string, any>;

export default function SmsPage() {
  const text = useText();
  const [balance, setBalance] = useState<RecordValue>({});
  const [sender, setSender] = useState<RecordValue>({});
  const [logs, setLogs] = useState<RecordValue[]>([]);
  const [apiKey, setApiKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  const [tab, setTab] = useState('overview');

  const load = useCallback(async () => {
    const results = await Promise.allSettled([
      posApi<RecordValue>('/messaging/balance'),
      posApi<RecordValue>('/messaging/sender-id'),
      posApi<unknown>('/messaging/messages?limit=20')
    ]);
    if (results[0].status === 'fulfilled') setBalance(results[0].value.data ?? {});
    if (results[1].status === 'fulfilled') setSender(results[1].value.data ?? {});
    if (results[2].status === 'fulfilled') setLogs(unwrapList<RecordValue>(results[2].value.data));
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveKey() {
    if (!apiKey.trim()) return;
    setSaving(true);
    setMessage('');
    try {
      await posApi<unknown>('/messaging/settings', {
        method: 'PATCH',
        body: JSON.stringify({ sendafrica_api_key: apiKey.trim() })
      });
      setApiKey('');
      setMessage(
        text('SendAfrica key saved securely.', 'Ufunguo wa SendAfrica umehifadhiwa kwa usalama.')
      );
      await load();
    } catch (reason) {
      setMessage(
        reason instanceof Error
          ? reason.message
          : text('Could not save the key.', 'Imeshindikana kuhifadhi ufunguo.')
      );
    } finally {
      setSaving(false);
    }
  }

  const free = Number(balance.free_sms_remaining ?? 100);
  const paid = Number(balance.paid_sms_balance ?? 0);
  const available = Number(balance.available_sms ?? free + paid);
  const tabs = [
    ['overview', text('Overview', 'Muhtasari')],
    ['setup', text('SendAfrica setup', 'Mipangilio ya SendAfrica')],
    ['sender', text('Sender ID', 'Jina la mtumaji')],
    ['logs', text('Message logs', 'Kumbukumbu za ujumbe')]
  ];

  return (
    <PageContainer
      pageTitle={text('SMS messaging', 'Ujumbe wa SMS')}
      pageDescription={text(
        'Send customer updates and manage your SendAfrica connection.',
        'Tuma taarifa kwa wateja na dhibiti muunganisho wako wa SendAfrica.'
      )}
      pageHeaderAction={
        <Button variant='outline' onClick={() => void load()}>
          <Icons.refresh className='size-4' /> {text('Refresh', 'Onyesha upya')}
        </Button>
      }
    >
      <div className='flex min-w-0 flex-col gap-4 sm:gap-6'>
        <Card className='border-sky-500/20 bg-sky-500/[0.04]'>
          <CardContent className='flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-6'>
            <div>
              <div className='flex items-center gap-2'>
                <Icons.send className='text-sky-500 size-5' />
                <h2 className='font-semibold'>{text('Your SMS wallet', 'Salio lako la SMS')}</h2>
              </div>
              <p className='text-muted-foreground mt-1 text-sm'>
                {text(
                  'Each organisation starts with 100 free SMS credits. Provider balance is separate.',
                  'Kila shirika huanza na SMS 100 za bure. Salio la SendAfrica ni tofauti.'
                )}
              </p>
            </div>
            <div className='text-left sm:text-right'>
              <p className='text-3xl font-bold tabular-nums'>{available}</p>
              <p className='text-muted-foreground text-xs'>
                {text('Available messages', 'Ujumbe unaopatikana')}
              </p>
            </div>
          </CardContent>
        </Card>
        <div className='flex gap-2 overflow-x-auto border-b pb-2'>
          {tabs.map(([key, label]) => (
            <Button
              key={key}
              size='sm'
              variant={tab === key ? 'default' : 'ghost'}
              className='shrink-0'
              onClick={() => setTab(key)}
            >
              {label}
            </Button>
          ))}
        </div>
        {tab === 'overview' && (
          <div className='grid gap-4 sm:grid-cols-3'>
            <Card>
              <CardHeader>
                <CardDescription>
                  {text('Free credits remaining', 'SMS za bure zilizobaki')}
                </CardDescription>
                <CardTitle>{free}</CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader>
                <CardDescription>{text('Paid SMS balance', 'SMS za kulipia')}</CardDescription>
                <CardTitle>{paid}</CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader>
                <CardDescription>{text('Sender status', 'Hali ya mtumaji')}</CardDescription>
                <CardTitle className='text-base'>
                  {sender.status ?? text('ZIADA POS default', 'ZIADA POS ya kawaida')}
                </CardTitle>
              </CardHeader>
            </Card>
          </div>
        )}
        {tab === 'setup' && (
          <Card>
            <CardHeader>
              <CardTitle>{text('Connect SendAfrica', 'Unganisha SendAfrica')}</CardTitle>
              <CardDescription>
                {text(
                  'Create an API key in SendAfrica, then paste it here. Ziada POS never displays the saved key.',
                  'Tengeneza API key kwenye SendAfrica, kisha bandika hapa. Ziada POS haitaonyesha ufunguo uliotunzwa.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              <a
                href='https://sendafrica.online'
                target='_blank'
                rel='noreferrer'
                className='text-primary inline-flex items-center gap-1 text-sm font-medium underline-offset-4 hover:underline'
              >
                {text(
                  'Create your SendAfrica account and API key',
                  'Fungua akaunti ya SendAfrica na API key'
                )}
                <Icons.externalLink className='size-3.5' />
              </a>
              <Input
                type='password'
                value={apiKey}
                onChange={(event) => setApiKey(event.target.value)}
                placeholder={text('Paste SendAfrica API key', 'Bandika API key ya SendAfrica')}
              />
              <Button onClick={() => void saveKey()} isLoading={saving} disabled={!apiKey.trim()}>
                <Icons.check className='size-4' /> {text('Save API key', 'Hifadhi API key')}
              </Button>
              {message && (
                <p role='status' className='text-muted-foreground text-sm'>
                  {message}
                </p>
              )}
            </CardContent>
          </Card>
        )}
        {tab === 'sender' && (
          <Card>
            <CardHeader>
              <CardTitle>{text('Sender ID', 'Jina la mtumaji')}</CardTitle>
              <CardDescription>
                {text(
                  'Use ZIADA POS by default or request an approved custom sender name.',
                  'Tumia ZIADA POS au omba jina maalum liidhinishwe.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='flex flex-wrap items-center gap-3'>
              <Badge variant='secondary'>{sender.active_sender_id ?? 'ZIADA POS'}</Badge>
              <Button variant='outline' disabled>
                {text('Request sender ID', 'Omba jina la mtumaji')} ·{' '}
                {text('Coming soon', 'Inakuja')}
              </Button>
            </CardContent>
          </Card>
        )}
        {tab === 'logs' && (
          <Card>
            <CardHeader>
              <CardTitle>{text('Message logs', 'Kumbukumbu za ujumbe')}</CardTitle>
              <CardDescription>
                {text(
                  'Track sent, delivered, and failed messages.',
                  'Fuatilia ujumbe uliotumwa, uliofikishwa na ulioshindikana.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className='overflow-x-auto'>
                <table className='w-full min-w-[600px] text-sm'>
                  <thead>
                    <tr className='border-b text-left text-muted-foreground'>
                      <th className='pb-2'>Phone</th>
                      <th className='pb-2'>Message</th>
                      <th className='pb-2'>Status</th>
                      <th className='pb-2'>Date</th>
                    </tr>
                  </thead>
                  <tbody>
                    {logs.map((item) => (
                      <tr key={item.id} className='border-b last:border-0'>
                        <td className='py-3'>{item.to ?? item.phone ?? '—'}</td>
                        <td className='max-w-[320px] truncate py-3'>
                          {item.body ?? item.message ?? '—'}
                        </td>
                        <td className='py-3'>
                          <Badge variant='outline'>{item.status ?? '—'}</Badge>
                        </td>
                        <td className='py-3 text-muted-foreground'>
                          {item.created_at ? new Date(item.created_at).toLocaleString() : '—'}
                        </td>
                      </tr>
                    ))}
                    {!logs.length && (
                      <tr>
                        <td colSpan={4} className='text-muted-foreground py-8 text-center'>
                          {text('No messages yet.', 'Hakuna ujumbe bado.')}
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </PageContainer>
  );
}
