'use client';

import Link from 'next/link';
import { FormEvent, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export default function ForgotPasswordPage() {
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const identifier = String(new FormData(event.currentTarget).get('identifier') || '');
    try {
      const response = await fetch('/api/pos/auth/password/forgot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ identifier })
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.message || 'Could not send reset code.');
      setMessage('If the account exists, a reset code has been sent.');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not send reset code.');
    } finally {
      setPending(false);
    }
  }
  return (
    <main className='bg-muted/30 flex min-h-svh items-center justify-center p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <CardTitle>Reset your password</CardTitle>
          <CardDescription>
            Enter your phone number or email to receive a reset code.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className='space-y-4'>
            <div className='space-y-2'>
              <Label htmlFor='identifier'>Phone or email</Label>
              <Input id='identifier' name='identifier' className='h-11' required />
            </div>
            {message && <p className='text-primary text-sm'>{message}</p>}
            {error && (
              <p role='alert' className='text-destructive text-sm'>
                {error}
              </p>
            )}
            <Button className='h-11 w-full' isLoading={pending}>
              Send reset code
            </Button>
            <Link
              href='/auth/sign-in'
              className='text-primary block text-center text-sm hover:underline'
            >
              Back to sign in
            </Link>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
