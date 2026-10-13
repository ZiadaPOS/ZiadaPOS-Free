import Link from 'next/link';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';

interface BusinessKpiCardProps {
  label: string;
  value: string;
  helper: string;
  icon?: React.ComponentType<{ className?: string }>;
  href?: string;
  actionLabel?: string;
}

export function BusinessKpiCard({
  label,
  value,
  helper,
  icon: Icon,
  href,
  actionLabel
}: BusinessKpiCardProps) {
  return (
    <Card className='min-w-0 overflow-hidden'>
      <CardHeader className='p-4 pb-2 sm:p-5 sm:pb-2'>
        <div className='flex items-start justify-between gap-2'>
          <CardDescription className='truncate text-xs sm:text-sm'>{label}</CardDescription>
          {Icon && <Icon className='text-muted-foreground size-4 shrink-0' />}
        </div>
        <CardTitle className='truncate text-2xl tabular-nums sm:text-3xl'>{value}</CardTitle>
      </CardHeader>
      <CardContent className='flex items-center justify-between gap-2 p-4 pt-0 sm:p-5 sm:pt-0'>
        <span className='text-muted-foreground min-w-0 truncate text-xs'>{helper}</span>
        {href && actionLabel && (
          <Link
            href={href}
            aria-label={actionLabel}
            className='hover:bg-muted inline-flex h-9 shrink-0 items-center justify-center gap-2 rounded-md px-2 text-sm font-medium transition-colors'
          >
            <span className='sr-only sm:not-sr-only'>{actionLabel}</span>
            <Icons.chevronRight className='size-4 sm:hidden' />
          </Link>
        )}
      </CardContent>
    </Card>
  );
}
