'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, type PosMedicineProfile } from '@/lib/pos-api';
import { FormEvent, useEffect, useState } from 'react';

const empty: PosMedicineProfile = {
  product_id: '',
  generic_name: '',
  brand_name: '',
  unit: '',
  pack_size: '',
  requires_prescription: false,
  medicine_schedule: 'unscheduled',
  registration_number: ''
};

export function MedicineProfileEditor({ productId }: { productId: string }) {
  const [profile, setProfile] = useState<PosMedicineProfile>(empty);
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    posApi<PosMedicineProfile>(`/pharmacy/products/${productId}`)
      .then((result) => setProfile(result.data))
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load medicine details.')
      );
  }, [productId]);
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setSaved(false);
    const data = new FormData(event.currentTarget);
    try {
      const result = await posApi<PosMedicineProfile>(`/pharmacy/products/${productId}`, {
        method: 'PUT',
        body: JSON.stringify({
          generic_name: data.get('generic_name'),
          brand_name: data.get('brand_name'),
          unit: data.get('unit'),
          pack_size: data.get('pack_size'),
          medicine_schedule: data.get('medicine_schedule'),
          registration_number: data.get('registration_number'),
          requires_prescription: data.get('requires_prescription') === 'on'
        })
      });
      setProfile(result.data);
      setSaved(true);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not save medicine details.');
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Medicine details</CardTitle>
        <CardDescription>
          Classify this product for dispensing and capture prescription requirements.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={save} className='grid gap-3 sm:grid-cols-2'>
          <label className='space-y-1 text-sm'>
            Generic name
            <Input
              name='generic_name'
              value={profile.generic_name}
              onChange={(event) => setProfile({ ...profile, generic_name: event.target.value })}
            />
          </label>
          <label className='space-y-1 text-sm'>
            Brand name
            <Input
              name='brand_name'
              value={profile.brand_name}
              onChange={(event) => setProfile({ ...profile, brand_name: event.target.value })}
            />
          </label>
          <label className='space-y-1 text-sm'>
            Unit
            <Input
              name='unit'
              value={profile.unit}
              onChange={(event) => setProfile({ ...profile, unit: event.target.value })}
            />
          </label>
          <label className='space-y-1 text-sm'>
            Pack size
            <Input
              name='pack_size'
              value={profile.pack_size}
              onChange={(event) => setProfile({ ...profile, pack_size: event.target.value })}
            />
          </label>
          <label className='space-y-1 text-sm'>
            Medicine schedule
            <select
              name='medicine_schedule'
              value={profile.medicine_schedule || 'unscheduled'}
              onChange={(event) =>
                setProfile({ ...profile, medicine_schedule: event.target.value })
              }
              className='bg-background h-10 w-full rounded-md border px-3'
            >
              <option value='unscheduled'>Unclassified</option>
              <option value='otc'>OTC</option>
              <option value='addo_permitted'>ADDO permitted</option>
              <option value='prescription_only'>Prescription only</option>
            </select>
          </label>
          <label className='space-y-1 text-sm'>
            Registration number
            <Input
              name='registration_number'
              value={profile.registration_number}
              onChange={(event) =>
                setProfile({ ...profile, registration_number: event.target.value })
              }
            />
          </label>
          <label className='flex items-center gap-2 text-sm sm:col-span-2'>
            <input
              type='checkbox'
              name='requires_prescription'
              checked={profile.requires_prescription}
              onChange={(event) =>
                setProfile({ ...profile, requires_prescription: event.target.checked })
              }
            />{' '}
            Require prescription details at checkout
          </label>
          <p className='text-muted-foreground text-xs sm:col-span-2'>
            Schedule selection is shop-entered scaffolding and is not a verified legal
            classification.
          </p>
          {error && (
            <p role='alert' className='text-destructive text-sm sm:col-span-2'>
              {error}
            </p>
          )}
          {saved && (
            <p role='status' className='text-sm sm:col-span-2'>
              Medicine details saved.
            </p>
          )}
          <div className='sm:col-span-2'>
            <Button disabled={busy}>{busy ? 'Saving…' : 'Save medicine details'}</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
