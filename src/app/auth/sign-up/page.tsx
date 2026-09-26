'use client';

import Link from 'next/link';
import { FormEvent, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Icons } from '@/components/icons';

const steps = [
  [
    ['first_name', 'First name'],
    ['last_name', 'Last name'],
    ['phone', 'Phone number'],
    ['email', 'Email address'],
    ['password', 'Password'],
    ['confirm_password', 'Confirm password']
  ],
  [
    ['shop_name', 'Business name'],
    ['business_type', 'Business type'],
    ['occupation', 'Your occupation']
  ],
  [
    ['region', 'Region'],
    ['district', 'District'],
    ['ward', 'Ward'],
    ['address', 'Address']
  ]
];

export default function SignUpPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  function next(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setStep((value) => value + 1);
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const form = new FormData(event.currentTarget);
    const value = (key: string) => String(form.get(key) || '');
    try {
      const response = await fetch('/api/pos/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          first_name: value('first_name'),
          last_name: value('last_name'),
          phone: value('phone'),
          email: value('email'),
          password: value('password'),
          confirm_password: value('confirm_password'),
          occupation: value('occupation'),
          shop: {
            name: value('shop_name'),
            business_type: value('business_type'),
            region: value('region'),
            district: value('district'),
            ward: value('ward'),
            address: value('address')
          }
        })
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.message || 'Could not create account.');
      router.replace('/auth/sign-in?registered=1');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create account.');
    } finally {
      setPending(false);
    }
  }
  return (
    <main className='bg-muted/30 flex min-h-svh items-center justify-center p-4'>
      <Card className='w-full max-w-2xl'>
        <CardHeader className='space-y-4'>
          <div className='flex items-center justify-center gap-3'>
            <img src='/ziada-final.jpeg' alt='' className='size-10 rounded-xl object-cover' />
            <span className='text-primary font-bold tracking-wide'>ZIADA POS</span>
          </div>
          <div className='text-center'>
            <CardTitle className='text-2xl'>Create your business account</CardTitle>
            <CardDescription>Set up your owner account and store in a few steps.</CardDescription>
          </div>
          <div className='grid grid-cols-3 gap-2'>
            {['Account', 'Business', 'Location'].map((label, index) => (
              <div
                key={label}
                className={`border-t-2 pt-2 text-xs ${index <= step ? 'border-primary text-primary' : 'border-muted text-muted-foreground'}`}
              >
                {index + 1}. {label}
              </div>
            ))}
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={step < 2 ? next : submit} className='space-y-5'>
            <div className='grid gap-4 sm:grid-cols-2'>
              {steps[step].map(([name, label]) => (
                <div key={name} className='space-y-2'>
                  <Label htmlFor={name}>{label}</Label>
                  <div className='relative'>
                    <Input
                      id={name}
                      name={name}
                      className='h-11'
                      type={
                        name.includes('password') && showPassword
                          ? 'text'
                          : name.includes('password')
                            ? 'password'
                            : name === 'email'
                              ? 'email'
                              : name === 'phone'
                                ? 'tel'
                                : 'text'
                      }
                      required={
                        !['last_name', 'email', 'district', 'ward', 'address'].includes(name)
                      }
                    />
                    {name === 'password' && (
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon'
                        className='absolute top-1/2 right-1 -translate-y-1/2'
                        onClick={() => setShowPassword((value) => !value)}
                        aria-label='Toggle password visibility'
                      >
                        {showPassword ? (
                          <Icons.eyeOff className='size-4' />
                        ) : (
                          <Icons.eye className='size-4' />
                        )}
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
            {error && (
              <p role='alert' className='text-destructive text-sm'>
                {error}
              </p>
            )}
            <div className='flex flex-col-reverse gap-2 sm:flex-row sm:justify-between'>
              {step > 0 ? (
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => setStep((value) => value - 1)}
                >
                  Back
                </Button>
              ) : (
                <Link
                  href='/auth/sign-in'
                  className='text-muted-foreground self-center text-sm hover:underline'
                >
                  Already have an account? Sign in
                </Link>
              )}
              <Button className='h-11 sm:min-w-40' type='submit' isLoading={pending}>
                {step < 2 ? 'Continue' : 'Create account'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
