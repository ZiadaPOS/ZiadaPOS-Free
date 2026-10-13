'use client';

import { useCallback, useEffect, useState, type FormEvent } from 'react';
import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList } from '@/lib/pos-api';

type PrescriptionRecord = {
  id: string;
  register_number: string;
  transaction_number: string;
  transaction_status: string;
  product_name: string;
  quantity: number;
  prescriber_name: string;
  prescriber_address: string;
  prescriber_registration_number: string;
  prescriber_authenticity_confirmed: boolean;
  prescribing_facility_name: string;
  prescribing_facility_address: string;
  prescription_reference: string;
  patient_name: string;
  patient_address: string;
  medicine_strength: string;
  dosage_form: string;
  batch_numbers: string;
  recorded_by: string;
  recorded_at: string;
  sale_at: string;
};

type PrescriptionPageResponse = {
  items: PrescriptionRecord[];
  total: number;
  page: number;
  page_size: number;
};

const pageSize = 50;

export default function PharmacyPrescriptionsPage() {
  const [records, setRecords] = useState<PrescriptionRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [appliedFrom, setAppliedFrom] = useState('');
  const [appliedTo, setAppliedTo] = useState('');
  const [page, setPage] = useState(1);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    if (appliedSearch.trim()) query.set('search', appliedSearch.trim());
    if (appliedFrom) query.set('from', appliedFrom);
    if (appliedTo) query.set('to', appliedTo);
    try {
      const result = await posApi<PrescriptionPageResponse>(`/pharmacy/prescriptions?${query}`);
      setRecords(unwrapList<PrescriptionRecord>(result.data));
      setTotal(result.data.total || 0);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load prescription records.');
    } finally {
      setLoading(false);
    }
  }, [appliedFrom, appliedSearch, appliedTo, page]);

  useEffect(() => {
    void load();
  }, [load]);

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPage(1);
    setAppliedSearch(search);
    setAppliedFrom(from);
    setAppliedTo(to);
  }

  const pageCount = Math.max(1, Math.ceil(total / pageSize));

  return (
    <PageContainer
      pageTitle='Prescription records'
      pageDescription='Review prescription details recorded with completed pharmacy sales.'
    >
      <Card>
        <CardHeader>
          <CardTitle>Dispensing audit</CardTitle>
          <CardDescription>
            The dispensing record captures staff-entered prescription details and the batch used.
            The authenticity checkbox records a staff action; it does not electronically validate a
            prescription or prescriber.
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <form
            onSubmit={applyFilters}
            className='grid gap-2 sm:grid-cols-[minmax(0,1fr)_170px_170px_auto]'
          >
            <Input
              aria-label='Search prescription records'
              placeholder='Search patient, prescriber, reference, medicine, or sale…'
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
            <Input
              aria-label='From date'
              type='date'
              value={from}
              onChange={(event) => setFrom(event.target.value)}
            />
            <Input
              aria-label='To date'
              type='date'
              value={to}
              onChange={(event) => setTo(event.target.value)}
            />
            <Button type='submit' variant='outline'>
              Apply filters
            </Button>
          </form>
          {error && (
            <p role='alert' className='text-destructive text-sm'>
              {error}
            </p>
          )}
          <div className='overflow-x-auto rounded-lg border'>
            <table className='w-full min-w-[1000px] text-left text-sm'>
              <thead className='bg-muted/50'>
                <tr>
                  {[
                    'Register no.',
                    'Sale date',
                    'Patient',
                    'Medicine',
                    'Qty',
                    'Prescriber',
                    'Prescription reference',
                    'Sale',
                    'Sale status',
                    'Recorded'
                  ].map((heading) => (
                    <th key={heading} className='px-3 py-2 font-medium'>
                      {heading}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {records.map((record) => (
                  <tr key={record.id}>
                    <td className='px-3 py-2 font-mono'>{record.register_number || '—'}</td>
                    <td className='px-3 py-2 whitespace-nowrap'>
                      {new Date(record.sale_at).toLocaleString()}
                    </td>
                    <td className='px-3 py-2'>
                      <details>
                        <summary className='cursor-pointer'>{record.patient_name}</summary>
                        <dl className='mt-2 grid min-w-64 gap-1 text-xs'>
                          <dt className='font-medium'>Patient address</dt>
                          <dd>{record.patient_address}</dd>
                          <dt className='font-medium'>Prescriber address / registration</dt>
                          <dd>
                            {record.prescriber_address} · {record.prescriber_registration_number}
                          </dd>
                          <dt className='font-medium'>Prescribing facility</dt>
                          <dd>
                            {record.prescribing_facility_name} ·{' '}
                            {record.prescribing_facility_address}
                          </dd>
                          <dt className='font-medium'>Medicine strength / form</dt>
                          <dd>
                            {record.medicine_strength} · {record.dosage_form}
                          </dd>
                          <dt className='font-medium'>Dispensed batch number(s)</dt>
                          <dd>{record.batch_numbers || 'No batch allocation recorded'}</dd>
                          <dt className='font-medium'>Staff authenticity check</dt>
                          <dd>
                            {record.prescriber_authenticity_confirmed
                              ? 'Confirmed by staff'
                              : 'Not confirmed'}
                          </dd>
                        </dl>
                      </details>
                    </td>
                    <td className='px-3 py-2'>{record.product_name}</td>
                    <td className='px-3 py-2 tabular-nums'>{record.quantity}</td>
                    <td className='px-3 py-2'>{record.prescriber_name}</td>
                    <td className='px-3 py-2'>{record.prescription_reference}</td>
                    <td className='px-3 py-2'>{record.transaction_number}</td>
                    <td className='px-3 py-2 capitalize'>{record.transaction_status}</td>
                    <td className='px-3 py-2 whitespace-nowrap'>
                      {new Date(record.recorded_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
                {!loading && !records.length && (
                  <tr>
                    <td colSpan={10} className='text-muted-foreground px-3 py-10 text-center'>
                      No prescription records found for these filters.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <p className='text-muted-foreground text-sm'>
              {loading ? 'Loading records…' : `${total} record(s)`}
            </p>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                disabled={page <= 1 || loading}
                onClick={() => setPage((current) => current - 1)}
              >
                Previous
              </Button>
              <span className='text-sm'>
                Page {page} of {pageCount}
              </span>
              <Button
                variant='outline'
                disabled={page >= pageCount || loading}
                onClick={() => setPage((current) => current + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </PageContainer>
  );
}
