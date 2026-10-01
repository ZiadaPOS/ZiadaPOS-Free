'use client';

import { Icons } from '@/components/icons';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { useEffect, useState } from 'react';
import { posApi, unwrapList } from '@/lib/pos-api';

const MAX_VISIBLE = 5;

export function NotificationCenter() {
  const [remoteNotifications, setRemoteNotifications] = useState<
    { id: string; title: string; body?: string; read?: boolean }[]
  >([]);
  useEffect(() => {
    posApi<unknown>('/notifications?page=1&page_size=20')
      .then((result) =>
        setRemoteNotifications(
          unwrapList<{ id: string; title: string; body?: string; read?: boolean }>(result.data)
        )
      )
      .catch(() => setRemoteNotifications([]));
  }, []);
  const count = remoteNotifications.filter((item) => !item.read).length;

  return (
    <Popover>
      <PopoverTrigger render={<Button variant='ghost' size='icon' className='relative h-8 w-8' />}>
        <Icons.notification className='h-4 w-4' />
        {count > 0 && (
          <span className='bg-destructive text-destructive-foreground absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-medium'>
            {count > 9 ? '9+' : count}
          </span>
        )}
        <span className='sr-only'>Notifications</span>
      </PopoverTrigger>
      <PopoverContent align='end' className='w-[calc(100vw-2rem)] p-0 sm:w-[380px]' sideOffset={8}>
        <div className='flex items-center justify-between px-4 pt-3'>
          <Link href='/notifications' className='group flex items-center gap-1'>
            <h4 className='text-sm font-semibold group-hover:underline'>Notifications</h4>
            <Icons.chevronRight className='text-muted-foreground h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5' />
          </Link>
          <div className='flex items-center gap-2'>
            {count > 0 && (
              <span className='bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-xs'>
                {count} new
              </span>
            )}
            {count > 0 && (
              <Button
                variant='ghost'
                size='sm'
                className='text-muted-foreground h-auto px-2 py-1 text-xs'
                onClick={async () => {
                  await posApi<unknown>('/notifications/read-all', {
                    method: 'PATCH',
                    body: JSON.stringify({})
                  }).catch(() => undefined);
                  setRemoteNotifications((items) => items.map((item) => ({ ...item, read: true })));
                }}
              >
                Mark all as read
              </Button>
            )}
          </div>
        </div>
        <Separator />
        <ScrollArea className='h-[400px]'>
          {remoteNotifications.length === 0 ? (
            <div className='flex flex-col items-center justify-center py-12'>
              <Icons.notification className='text-muted-foreground/40 mb-2 h-8 w-8' />
              <p className='text-muted-foreground text-sm'>No notifications yet</p>
            </div>
          ) : (
            <div className='flex flex-col gap-1 p-2'>
              {remoteNotifications.length > 0 &&
                remoteNotifications.slice(0, MAX_VISIBLE).map((notification) => (
                  <div key={notification.id} className='rounded-md border p-3'>
                    <p className='text-sm font-medium'>{notification.title}</p>
                    {notification.body && (
                      <p className='text-muted-foreground mt-1 text-xs'>{notification.body}</p>
                    )}
                    {!notification.read && (
                      <Button
                        size='sm'
                        variant='ghost'
                        className='mt-1 h-7 px-2 text-xs'
                        onClick={async () => {
                          await posApi<unknown>(`/notifications/${notification.id}/read`, {
                            method: 'PATCH',
                            body: JSON.stringify({})
                          }).catch(() => undefined);
                          setRemoteNotifications((items) =>
                            items.map((item) =>
                              item.id === notification.id ? { ...item, read: true } : item
                            )
                          );
                        }}
                      >
                        Mark read
                      </Button>
                    )}
                  </div>
                ))}
            </div>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
}
