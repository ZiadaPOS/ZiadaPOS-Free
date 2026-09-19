'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { PosUser } from '@/lib/pos-api';
import { useRouter } from 'next/navigation';
import { FormEvent, useState } from 'react';

export default function SignInPage() {
  const router = useRouter();
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    setPending(true);
    const form = new FormData(event.currentTarget);
    try {
      const response = await fetch('/api/pos-session', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ identifier: form.get('identifier'), password: form.get('password') }) });
      const result = await response.json();
      if (!response.ok || !result.success) throw new Error(result.message || 'Could not sign in.');
      localStorage.setItem('pos_user', JSON.stringify(result.data.user));
      router.replace('/dashboard/overview');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not sign in. Check your connection and try again.');
    } finally {
      setPending(false);
    }
  }

  return (
    <main className='bg-muted/30 flex min-h-svh items-center justify-center p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader className='space-y-2'>
          <div className='text-primary text-sm font-semibold tracking-wide'>ZIADA POS</div>
          <CardTitle className='text-2xl'>Sign in to your business</CardTitle>
          <CardDescription>Use the phone number or email and password for your ZiadaPOS account.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className='space-y-5'>
            <div className='space-y-2'><Label htmlFor='identifier'>Phone or email</Label><Input id='identifier' name='identifier' autoComplete='username' required placeholder='0712 345 678' /></div>
            <div className='space-y-2'><Label htmlFor='password'>Password</Label><Input id='password' name='password' type='password' autoComplete='current-password' required /></div>
            {error && <p className='text-destructive text-sm' role='alert'>{error}</p>}
            <Button className='w-full' type='submit' isLoading={pending}>Sign in</Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
