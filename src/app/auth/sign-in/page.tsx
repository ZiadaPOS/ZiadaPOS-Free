'use client';

import Link from 'next/link';
import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Icons } from '@/components/icons';
import { tanzaniaPhoneLocalNumber, tanzaniaPhoneNationalDigits } from '@/lib/tanzania-phone';

export default function SignInPage() {
  const router = useRouter();
  const [phoneDigits, setPhoneDigits] = useState('');
  const [remember, setRemember] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const saved = localStorage.getItem('pos_login_identifier');
    if (saved) {
      const digits = tanzaniaPhoneNationalDigits(saved);
      if (digits.length === 9) {
        setPhoneDigits(digits);
        setRemember(true);
      } else {
        localStorage.removeItem('pos_login_identifier');
      }
    }
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const form = new FormData(event.currentTarget);
    const phone = tanzaniaPhoneLocalNumber(phoneDigits);
    try {
      const response = await fetch('/api/pos-session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ identifier: phone, password: form.get('password') })
      });
      const result = await response.json();
      if (!response.ok || !result.success) throw new Error(result.message || 'Could not sign in.');
      if (remember) localStorage.setItem('pos_login_identifier', phone);
      else localStorage.removeItem('pos_login_identifier');
      localStorage.setItem('pos_user', JSON.stringify(result.data.user));
      if (result.data.business_type)
        localStorage.setItem('pos_business_type', result.data.business_type);
      if (result.data.outlet_type) localStorage.setItem('pos_outlet_type', result.data.outlet_type);
      router.replace('/dashboard/overview');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not sign in.');
    } finally {
      setPending(false);
    }
  }

  return (
    <main className='bg-muted/30 flex min-h-svh items-center justify-center p-4 font-sans'>
      <Card className='w-full max-w-md'>
        <CardHeader className='space-y-4'>
          <div className='flex items-center justify-center gap-3'>
            <img src='/ziada-final.jpeg' alt='' className='size-10 rounded-full object-cover' />
            <span className='text-primary font-bold tracking-wide'>ZIADA POS</span>
          </div>
          <div className='space-y-2 text-center'>
            <CardTitle className='text-2xl'>Sign in to your business</CardTitle>
            <CardDescription>
              Use your Tanzanian phone number and password to continue.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className='space-y-5'>
            <div className='space-y-2'>
              <Label htmlFor='phone'>Phone number</Label>
              <div className='relative'>
                <Input
                  id='phone'
                  name='phone'
                  className='h-11 pl-16'
                  type='tel'
                  inputMode='tel'
                  autoComplete='username'
                  value={phoneDigits}
                  onChange={(event) =>
                    setPhoneDigits(tanzaniaPhoneNationalDigits(event.target.value))
                  }
                  maxLength={9}
                  pattern='[0-9]{9}'
                  required
                  placeholder='712345678'
                />
                <span className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 flex h-5 -translate-y-1/2 items-center border-r border-border pr-2 text-sm'>
                  +255
                </span>
              </div>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='password'>Password</Label>
              <div className='relative'>
                <Input
                  id='password'
                  name='password'
                  className='h-11 pr-11'
                  type={showPassword ? 'text' : 'password'}
                  autoComplete='current-password'
                  required
                />
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  className='absolute top-1/2 right-1 -translate-y-1/2'
                  aria-label='Toggle password visibility'
                  onClick={() => setShowPassword((value) => !value)}
                >
                  {showPassword ? (
                    <Icons.eyeOff className='size-4' />
                  ) : (
                    <Icons.eye className='size-4' />
                  )}
                </Button>
              </div>
            </div>
            <div className='flex items-center justify-between gap-3 text-sm'>
              <label className='flex items-center gap-2'>
                <input
                  type='checkbox'
                  checked={remember}
                  onChange={(event) => setRemember(event.target.checked)}
                />{' '}
                Remember me
              </label>
              <Link href='/auth/forgot-password' className='text-primary hover:underline'>
                Forgot password?
              </Link>
            </div>
            {error && (
              <p className='text-destructive text-sm' role='alert'>
                {error}
              </p>
            )}
            <Button className='h-11 w-full' type='submit' isLoading={pending}>
              Sign in
            </Button>
            <p className='text-muted-foreground text-center text-sm'>
              Don&apos;t have an account?{' '}
              <Link href='/auth/sign-up' className='text-primary font-medium hover:underline'>
                Create one
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
