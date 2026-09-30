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
        return response.json();
      })
      .then((result) => {
        if (!active) return;
        const destination =
          result.data?.business_type === 'pharmacy'
            ? '/dashboard/pharmacy/expiry'
            : '/dashboard/overview';
        router.replace(destination);
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
