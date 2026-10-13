'use client';

import Link from 'next/link';
import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

type LocalStatus = { success?: boolean; data?: { local_mode?: boolean; configured?: boolean } };

export default function LocalSetupPage() {
  const router = useRouter();
  const [ready, setReady] = useState(false);
  const [localMode, setLocalMode] = useState(false);
  const [configured, setConfigured] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    void fetch('/api/local/status', { cache: 'no-store' })
      .then((response) => response.json() as Promise<LocalStatus>)
      .then((result) => {
        setLocalMode(Boolean(result.success && result.data?.local_mode));
        setConfigured(Boolean(result.success && result.data?.configured));
      })
      .catch(() => setLocalMode(false))
      .finally(() => setReady(true));
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const form = new FormData(event.currentTarget);
    const payload = Object.fromEntries(form.entries());
    try {
      const response = await fetch('/api/local/setup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const result = await response.json();
      if (!response.ok || !result.success)
        throw new Error(result.message || 'Could not set up this shop.');
      localStorage.setItem('pos_user', JSON.stringify(result.data.user));
      localStorage.setItem('pos_business_type', 'retail');
      router.replace('/checkout');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not set up this shop.');
    } finally {
      setPending(false);
    }
  }

  return (
    <main className='bg-muted/30 flex min-h-svh items-center justify-center p-4 font-sans'>
      <Card className='w-full max-w-lg'>
        <CardHeader className='space-y-3'>
          <div className='flex items-center justify-center gap-3'>
            <Image
              src='/ziada-final.jpeg'
              alt=''
              width={40}
              height={40}
              className='size-10 rounded-full object-cover'
            />
            <span className='text-primary font-bold tracking-wide'>ZIADA POS · LOCAL</span>
          </div>
          <div className='space-y-2 text-center'>
            <CardTitle className='text-2xl'>Set up this computer</CardTitle>
            <CardDescription>
              Create a local shop and owner sign-in. Your shop data will be stored on this computer.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          {!ready ? (
            <p className='text-muted-foreground py-4 text-center text-sm'>Checking local setup…</p>
          ) : !localMode ? (
            <div className='space-y-4 text-center text-sm'>
              <p>This setup is available only in the ZiadaPOS desktop app.</p>
              <Link href='/auth/sign-in' className='text-primary hover:underline'>
                Back to sign in
              </Link>
            </div>
          ) : configured ? (
            <div className='space-y-4 text-center text-sm'>
              <p>This computer already has a local shop. Sign in with its local owner account.</p>
              <Link href='/auth/sign-in' className='text-primary hover:underline'>
                Back to sign in
              </Link>
            </div>
          ) : (
            <form onSubmit={submit} className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='business_name'>Shop name</Label>
                <Input
                  id='business_name'
                  name='business_name'
                  required
                  maxLength={120}
                  autoComplete='organization'
                />
              </div>
              <div className='grid gap-4 sm:grid-cols-2'>
                <div className='space-y-2'>
                  <Label htmlFor='first_name'>Owner first name</Label>
                  <Input
                    id='first_name'
                    name='first_name'
                    required
                    maxLength={80}
                    autoComplete='given-name'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='last_name'>Owner last name</Label>
                  <Input
                    id='last_name'
                    name='last_name'
                    required
                    maxLength={80}
                    autoComplete='family-name'
                  />
                </div>
              </div>
              <div className='space-y-2'>
                <Label htmlFor='phone'>Owner phone number</Label>
                <Input
                  id='phone'
                  name='phone'
                  type='tel'
                  inputMode='tel'
                  autoComplete='tel'
                  required
                  placeholder='0712345678'
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='password'>Local sign-in password</Label>
                <Input
                  id='password'
                  name='password'
                  type='password'
                  autoComplete='new-password'
                  minLength={8}
                  required
                />
                <p className='text-muted-foreground text-xs'>
                  Use at least 8 characters. This password is for this computer’s local account.
                </p>
              </div>
              {error && (
                <p className='text-destructive text-sm' role='alert'>
                  {error}
                </p>
              )}
              <Button className='h-11 w-full' type='submit' isLoading={pending}>
                Create local shop
              </Button>
              <p className='text-muted-foreground text-center text-sm'>
                Already set up?{' '}
                <Link href='/auth/sign-in' className='text-primary font-medium hover:underline'>
                  Sign in
                </Link>
              </p>
            </form>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
