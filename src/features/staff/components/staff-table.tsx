'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { posApi } from '@/lib/pos-api';
import { useState } from 'react';

export function StaffTable({
  records,
  onUpdated
}: {
  records: Record<string, unknown>[];
  onUpdated: () => void;
}) {
  const [message, setMessage] = useState('');
  async function togglePermission(
    record: Record<string, unknown>,
    permission: 'can_refund' | 'can_discount' | 'can_view_reports'
  ) {
    const id = String(record.id || record.user_id || '');
    try {
      await posApi<unknown>(`/staff/${id}/permissions`, {
        method: 'PATCH',
        body: JSON.stringify({ [permission]: !record[permission] })
      });
      setMessage('Staff permissions updated.');
      onUpdated();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not update permissions.');
    }
  }
  async function deactivate(record: Record<string, unknown>) {
    const id = String(record.id || record.user_id || '');
    if (
      !window.confirm(
        `Deactivate ${String(record.full_name || record.first_name || 'this staff member')}?`
      )
    )
      return;
    try {
      await posApi<unknown>(`/staff/${id}`, { method: 'DELETE' });
      setMessage('Staff account deactivated.');
      onUpdated();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not deactivate staff.');
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Team members</CardTitle>
      </CardHeader>
      <CardContent>
        <div className='overflow-x-auto'>
          <table className='w-full text-left text-sm'>
            <thead>
              <tr className='border-b'>
                {['Name', 'Phone', 'Role', 'Shift', 'POS permissions', 'Actions'].map((heading) => (
                  <th key={heading} className='px-3 py-3'>
                    {heading}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className='divide-y'>
              {records.map((record) => (
                <tr key={String(record.id)}>
                  <td className='px-3 py-3 font-medium'>
                    {String(
                      record.full_name || `${record.first_name || ''} ${record.last_name || ''}`
                    )}
                  </td>
                  <td className='px-3 py-3'>{String(record.phone || '—')}</td>
                  <td className='px-3 py-3'>{String(record.role || 'staff')}</td>
                  <td className='px-3 py-3'>
                    {String(record.shift_display || record.shift || '—')}
                  </td>
                  <td className='px-3 py-3'>
                    <div className='flex flex-wrap gap-1'>
                      {(['can_refund', 'can_discount', 'can_view_reports'] as const).map(
                        (permission) => (
                          <Button
                            key={permission}
                            size='sm'
                            variant={record[permission] ? 'default' : 'outline'}
                            onClick={() => void togglePermission(record, permission)}
                          >
                            {permission.replace('can_', '').replaceAll('_', ' ')}:{' '}
                            {record[permission] ? 'yes' : 'no'}
                          </Button>
                        )
                      )}
                    </div>
                  </td>
                  <td className='px-3 py-3'>
                    <Button size='sm' variant='ghost' onClick={() => void deactivate(record)}>
                      Deactivate
                    </Button>
                  </td>
                </tr>
              ))}
              {!records.length && (
                <tr>
                  <td colSpan={6} className='text-muted-foreground px-3 py-8 text-center'>
                    No staff found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {message && (
          <p role='status' className='text-muted-foreground mt-3 text-sm'>
            {message}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
