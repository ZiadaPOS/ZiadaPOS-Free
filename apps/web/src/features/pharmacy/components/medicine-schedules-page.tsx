'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { BusinessKpiCard } from '@/components/business/business-kpi-card';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList, type PosMedicineProfile, type PosProduct } from '@/lib/pos-api';

type MedicineSchedule = 'unscheduled' | 'otc' | 'addo_permitted' | 'prescription_only';
type MedicineRow = { product: PosProduct; profile?: PosMedicineProfile };

const schedules: { value: MedicineSchedule; label: string; description: string }[] = [
  {
    value: 'unscheduled',
    label: 'Unclassified',
    description: 'No schedule label has been selected for this medicine.'
  },
  {
    value: 'otc',
    label: 'OTC',
    description:
      'A shop-selected OTC label. This label alone does not request prescription details at checkout.'
  },
  {
    value: 'addo_permitted',
    label: 'ADDO permitted',
    description:
      'A shop-selected ADDO label. The app does not verify legal eligibility from this selection.'
  },
  {
    value: 'prescription_only',
    label: 'Prescription only',
    description:
      'Checkout requires prescription-register details and prescriber authenticity confirmation.'
  }
];

function scheduleLabel(value?: string) {
  return schedules.find((schedule) => schedule.value === value)?.label || 'Unclassified';
}

export default function MedicineSchedulesPage() {
  const [rows, setRows] = useState<MedicineRow[]>([]);
  const [search, setSearch] = useState('');
  const [scheduleFilter, setScheduleFilter] = useState('all');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [productsResult, profilesResult] = await Promise.all([
        posApi<unknown>('/inventory/products?page=1&page_size=200'),
        posApi<unknown>('/pharmacy/products')
      ]);
      const products = unwrapList<PosProduct>(productsResult.data).filter(
        (product) => !product.is_service
      );
      const profiles = new Map(
        unwrapList<PosMedicineProfile>(profilesResult.data).map((profile) => [
          profile.product_id,
          profile
        ])
      );
      setRows(products.map((product) => ({ product, profile: profiles.get(product.id) })));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load medicine schedules.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const visibleRows = useMemo(() => {
    const query = search.trim().toLowerCase();
    return rows.filter(({ product, profile }) => {
      const schedule = profile?.medicine_schedule || 'unscheduled';
      const matchesSearch =
        !query ||
        `${product.name} ${profile?.generic_name || ''} ${profile?.brand_name || ''}`
          .toLowerCase()
          .includes(query);
      return matchesSearch && (scheduleFilter === 'all' || schedule === scheduleFilter);
    });
  }, [rows, search, scheduleFilter]);

  const prescriptionCount = rows.filter(
    ({ profile }) =>
      profile?.requires_prescription || profile?.medicine_schedule === 'prescription_only'
  ).length;
  const unclassifiedCount = rows.filter(
    ({ profile }) =>
      !profile || !profile.medicine_schedule || profile.medicine_schedule === 'unscheduled'
  ).length;

  return (
    <PageContainer
      pageTitle='Medicine schedules'
      pageDescription='Review schedule labels and understand how they affect pharmacy checkout.'
      pageHeaderAction={
        <Button className='w-full sm:w-auto' onClick={() => (window.location.href = '/inventory')}>
          Manage medicines
        </Button>
      }
    >
      <div className='space-y-5'>
        <div className='grid grid-cols-2 gap-3 xl:grid-cols-3'>
          <BusinessKpiCard
            label='Medicines'
            value={String(rows.length)}
            helper='Stock products in this store'
            icon={Icons.product}
          />
          <BusinessKpiCard
            label='Prescription capture'
            value={String(prescriptionCount)}
            helper='Require prescription details at checkout'
            icon={Icons.fileTypePdf}
          />
          <BusinessKpiCard
            label='Unclassified'
            value={String(unclassifiedCount)}
            helper='Review the schedule label'
            icon={Icons.info}
          />
        </div>

        <Card>
          <CardHeader>
            <CardTitle>How schedules affect checkout</CardTitle>
            <CardDescription>
              These are the labels currently available in the medicine form and the behavior the app
              applies.
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            {schedules.map((schedule) => (
              <div
                key={schedule.value}
                className='grid gap-1 rounded-lg border p-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-start sm:gap-4'
              >
                <p className='font-medium'>{schedule.label}</p>
                <p className='text-muted-foreground text-sm'>{schedule.description}</p>
              </div>
            ))}
            <div className='rounded-lg border border-primary/25 bg-primary/5 p-4'>
              <p className='font-medium'>Prescription setting</p>
              <p className='text-muted-foreground mt-1 text-sm'>
                The separate “Require a prescription” option also makes checkout collect
                prescription-register details, even when the schedule isn’t “Prescription only.” For
                an ADDO outlet, the API blocks a medicine marked “Prescription only.”
              </p>
            </div>
            <p className='text-muted-foreground text-xs'>
              Schedule labels are entered by your shop; the app does not verify legal classification
              or dispensing permissions. Confirm classifications with an authoritative Tanzanian
              source.
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Medicine schedule register</CardTitle>
            <CardDescription>
              Find a medicine and open its inventory record to review or change its dispensing
              details.
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-4'>
            <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_14rem]'>
              <Input
                aria-label='Search medicines'
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder='Search medicine, generic, or brand…'
              />
              <select
                aria-label='Filter by medicine schedule'
                value={scheduleFilter}
                onChange={(event) => setScheduleFilter(event.target.value)}
                className='bg-background h-11 w-full rounded-lg border border-foreground/20 px-3 text-sm'
              >
                <option value='all'>All schedules</option>
                {schedules.map((schedule) => (
                  <option key={schedule.value} value={schedule.value}>
                    {schedule.label}
                  </option>
                ))}
              </select>
            </div>
            {error && (
              <p role='alert' className='text-destructive text-sm'>
                {error}
              </p>
            )}
            <div className='overflow-x-auto rounded-lg border'>
              <table className='w-full min-w-[640px] text-left text-sm'>
                <thead className='bg-muted/40'>
                  <tr>
                    <th className='px-4 py-3 font-medium'>Medicine</th>
                    <th className='px-4 py-3 font-medium'>Schedule</th>
                    <th className='px-4 py-3 font-medium'>Checkout</th>
                    <th className='px-4 py-3 text-right font-medium'>Action</th>
                  </tr>
                </thead>
                <tbody className='divide-y'>
                  {visibleRows.map(({ product, profile }) => {
                    const prescriptionRequired = Boolean(
                      profile?.requires_prescription ||
                      profile?.medicine_schedule === 'prescription_only'
                    );
                    return (
                      <tr key={product.id} className='hover:bg-muted/30'>
                        <td className='px-4 py-3'>
                          <p className='font-medium'>{product.name}</p>
                          <p className='text-muted-foreground text-xs'>
                            {[profile?.generic_name, profile?.brand_name]
                              .filter(Boolean)
                              .join(' · ') || 'No generic or brand name'}
                          </p>
                        </td>
                        <td className='px-4 py-3'>{scheduleLabel(profile?.medicine_schedule)}</td>
                        <td className='px-4 py-3'>
                          {profile
                            ? prescriptionRequired
                              ? 'Prescription details required'
                              : 'No prescription capture'
                            : 'Profile not set up'}
                        </td>
                        <td className='px-4 py-3 text-right'>
                          <Button
                            variant='outline'
                            size='sm'
                            onClick={() => (window.location.href = `/inventory/${product.id}`)}
                          >
                            Review
                          </Button>
                        </td>
                      </tr>
                    );
                  })}
                  {!visibleRows.length && (
                    <tr>
                      <td colSpan={4} className='text-muted-foreground px-4 py-10 text-center'>
                        {loading
                          ? 'Loading medicines…'
                          : rows.length
                            ? 'No medicines match these filters.'
                            : 'No medicine products found.'}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            <p className='text-muted-foreground text-xs'>
              Showing {visibleRows.length} of {rows.length} medicines.
            </p>
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  );
}
