'use client';

import { useCallback, useEffect, useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi } from '@/lib/pos-api';

type OrganisationSettings = {
  ngamia_configured?: boolean;
  ngamia_model?: string;
};
type NgamiaModel = {
  id?: string;
  model?: string;
  display_name?: string;
};

export function ProviderIntegrations() {
  const [organisation, setOrganisation] = useState<OrganisationSettings>({});
  const [apiKey, setApiKey] = useState('');
  const [models, setModels] = useState<NgamiaModel[]>([]);
  const [model, setModel] = useState('');
  const [busy, setBusy] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const load = useCallback(async () => {
    try {
      const result = await posApi<OrganisationSettings>('/accounts/organisation');
      setOrganisation(result.data);
      setModel(result.data.ngamia_model || '');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load integration settings.');
    }
  }, []);

  useEffect(() => {
    let active = true;
    void posApi<OrganisationSettings>('/accounts/organisation')
      .then((result) => {
        if (!active) return;
        setOrganisation(result.data);
        setModel(result.data.ngamia_model || '');
      })
      .catch((reason) => {
        if (!active) return;
        setError(reason instanceof Error ? reason.message : 'Could not load integration settings.');
      });
    return () => {
      active = false;
    };
  }, []);

  async function save(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setNotice('');
    const payload: Record<string, string> = { ngamia_model: model.trim() };
    if (apiKey.trim()) payload.ngamia_api_key = apiKey.trim();
    try {
      await posApi<unknown>('/accounts/organisation', {
        method: 'PATCH',
        body: JSON.stringify(payload)
      });
      setApiKey('');
      setNotice('Ngamia settings saved. The API key is not returned to the browser.');
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not save Ngamia settings.');
    } finally {
      setBusy(false);
    }
  }

  async function removeKey() {
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await posApi<unknown>('/accounts/organisation', {
        method: 'PATCH',
        body: JSON.stringify({ ngamia_api_key: '' })
      });
      setNotice('Saved Ngamia key removed.');
      setModels([]);
      await load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not remove the Ngamia key.');
    } finally {
      setBusy(false);
    }
  }

  async function loadModels() {
    setChecking(true);
    setError('');
    setNotice('');
    try {
      const result = await posApi<NgamiaModel[]>('/ai/models');
      const available = result.data || [];
      setModels(available);
      if (available.length) {
        const saved = organisation.ngamia_model || '';
        const selectedExists = available.some((entry) => (entry.id || entry.model) === saved);
        if (!selectedExists) setModel(available[0].id || available[0].model || '');
        setNotice(`Connected to Ngamia. Loaded ${available.length} model(s).`);
      } else {
        setNotice('Ngamia returned an empty model catalog.');
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not connect to Ngamia.');
    } finally {
      setChecking(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Ngamia AI</CardTitle>
        <CardDescription>
          Connect your Ngamia API key for Ziada AI. The key is sent to the authenticated API server,
          encrypted before database storage, and never returned after saving.
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-4'>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge variant={organisation.ngamia_configured ? 'secondary' : 'outline'}>
            {organisation.ngamia_configured ? 'API key saved' : 'Not connected'}
          </Badge>
          {organisation.ngamia_configured && (
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => void removeKey()}
              disabled={busy}
            >
              Remove key
            </Button>
          )}
        </div>
        <form
          onSubmit={save}
          className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-end'
        >
          <label htmlFor='ngamia-api-key' className='space-y-1 text-sm font-medium'>
            Ngamia API key
            <Input
              id='ngamia-api-key'
              type='password'
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              placeholder={
                organisation.ngamia_configured
                  ? 'Saved — paste a new key to rotate'
                  : 'Paste your ngm_ API key'
              }
              autoComplete='new-password'
              spellCheck={false}
            />
          </label>
          <label className='space-y-1 text-sm font-medium'>
            Model
            {models.length ? (
              <select
                className='border-input bg-background h-10 w-full rounded-md border px-3 text-sm'
                value={model}
                onChange={(event) => setModel(event.target.value)}
              >
                {models.map((entry) => {
                  const id = entry.id || entry.model || '';
                  return (
                    <option key={id} value={id}>
                      {entry.display_name ? `${entry.display_name} (${id})` : id}
                    </option>
                  );
                })}
              </select>
            ) : (
              <Input
                value={model}
                onChange={(event) => setModel(event.target.value)}
                placeholder='Fetch the model catalog after saving a key'
              />
            )}
          </label>
          <Button type='submit' isLoading={busy}>
            Save
          </Button>
        </form>
        <div className='flex flex-wrap gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={() => void loadModels()}
            isLoading={checking}
            disabled={!organisation.ngamia_configured}
          >
            Fetch models / test connection
          </Button>
          <a
            href='https://docs.ngamia.cc/'
            target='_blank'
            rel='noreferrer'
            className='text-primary inline-flex items-center text-sm underline-offset-4 hover:underline'
          >
            Ngamia API docs
          </a>
        </div>
        <div className='bg-muted/40 text-muted-foreground rounded-md p-3 text-sm'>
          When you confirm and send a chat message, Ziada AI sends that text, recent chat history,
          and a minimized shop summary (sales totals, low-stock product names/SKUs, and aggregate
          credit totals) to Ngamia. Customer names, phone numbers, addresses, and transaction IDs
          are excluded from the generated shop context. Avoid entering personal, payment, or other
          sensitive details. Ngamia may charge the balance associated with the selected API key.
        </div>
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
  );
}
