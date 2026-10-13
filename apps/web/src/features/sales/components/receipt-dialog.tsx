'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, money, type PosTransaction } from '@/lib/pos-api';
import { useState } from 'react';

export function ReceiptDialog({
  transaction,
  localMode = false,
  onClose
}: {
  transaction: PosTransaction;
  localMode?: boolean;
  onClose: () => void;
}) {
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState('');
  async function send() {
    try {
      await posApi<unknown>(`/pos/transactions/${transaction.id}/receipt`, {
        method: 'POST',
        body: JSON.stringify({ email })
      });
      setStatus('Receipt emailed.');
    } catch (reason) {
      setStatus(reason instanceof Error ? reason.message : 'Email failed.');
    }
  }
  return (
    <div className='fixed inset-0 z-50 grid place-items-center bg-black/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <CardTitle>Sale complete · {transaction.txn_number}</CardTitle>
          <CardDescription>
            {localMode
              ? `Receipt total ${money(transaction.total)}. This sale is saved on this computer and is not synced online.`
              : `Receipt total ${money(transaction.total)}. Print this receipt or email a copy.`}
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-3'>
          <Button className='w-full' onClick={() => window.print()}>
            Print receipt
          </Button>
          {!localMode && (
            <div className='flex gap-2'>
              <Input
                type='email'
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder='Customer email'
              />
              <Button variant='outline' onClick={() => void send()}>
                Email
              </Button>
            </div>
          )}
          {status && (
            <p role='status' className='text-sm'>
              {status}
            </p>
          )}
          <Button className='w-full' variant='ghost' onClick={onClose}>
            Close
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
