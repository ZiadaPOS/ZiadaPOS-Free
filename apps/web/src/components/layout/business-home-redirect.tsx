'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export function BusinessHomeRedirect() {
  const router = useRouter();

  useEffect(() => {
    let active = true;
    fetch('/api/pos-session', { cache: 'no-store' })
      .then(async (response) => {
        if (!response.ok) throw new Error('Unauthenticated');
        if (!active) return;
        const session = await response.json();
        router.replace(session.data?.offline_mode ? '/checkout' : '/overview');
      })
      .catch(() => {
        if (active) router.replace('/auth/sign-in');
      });
    return () => {
      active = false;
    };
  }, [router]);

  return (
    <main className='text-muted-foreground flex min-h-svh items-center justify-center text-sm'>
      Loading your workspace…
    </main>
  );
}
