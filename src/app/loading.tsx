import Image from 'next/image';
import { Skeleton } from '@/components/ui/skeleton';
import { Spinner } from '@/components/ui/spinner';

export default function Loading() {
  return (
    <main
      role='status'
      aria-label='Loading Ziada POS'
      className='flex min-h-[calc(100svh-4rem)] min-w-0 flex-1 flex-col items-center justify-center gap-8 p-4 sm:p-8'
    >
      <div className='flex flex-col items-center text-center'>
        <Image
          src='/ziada-final.jpeg'
          alt=''
          width={56}
          height={56}
          priority
          className='mb-3 size-14 rounded-xl object-cover shadow-sm'
        />
        <p className='text-lg font-semibold tracking-tight'>Ziada POS</p>
        <p className='text-muted-foreground mt-1 text-sm'>Inapakia ukurasa wako…</p>
        <Spinner aria-hidden className='text-primary mt-5 size-6' />
      </div>

      <div aria-hidden='true' className='grid w-full max-w-5xl gap-4 sm:grid-cols-2'>
        <section className='border-border flex flex-col gap-4 rounded-xl border p-4 sm:p-5'>
          <Skeleton className='h-6 w-40' />
          <Skeleton className='h-4 w-64 max-w-full' />
          <Skeleton className='mt-2 h-10 w-full' />
          <div className='grid grid-cols-2 gap-3 sm:grid-cols-3'>
            <Skeleton className='h-24' />
            <Skeleton className='h-24' />
            <Skeleton className='hidden h-24 sm:block' />
          </div>
        </section>
        <section className='border-border flex flex-col gap-4 rounded-xl border p-4 sm:p-5'>
          <Skeleton className='h-6 w-36' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-4/5' />
          <Skeleton className='mt-2 h-11 w-full rounded-lg' />
        </section>
      </div>
    </main>
  );
}
