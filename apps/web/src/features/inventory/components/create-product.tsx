'use client';

import Link from 'next/link';
import { Button, buttonVariants } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { posApi, unwrapList, type PosCategory, type PosProduct, type PosUnit } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { FormEvent, useEffect, useRef, useState } from 'react';

const PRODUCT_DRAFT_KEY = 'ziada-pos:inventory:create-product:draft:v1';

type ProductDraft = {
  values: Record<string, string | boolean>;
  openingBatchNumber: string;
  openDetails?: boolean[];
  createdProductId?: string;
};

function RequiredMark() {
  return (
    <span aria-hidden='true' className='text-destructive'>
      {'*'}
    </span>
  );
}

function generateOpeningBatchNumber() {
  const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
  const randomBytes = new Uint8Array(8);
  window.crypto.getRandomValues(randomBytes);
  const suffix = Array.from(randomBytes, (byte) => alphabet[byte & 31]).join('');
  return `OPN-${suffix}`;
}

export function CreateProduct({
  onCreated,
  isPharmacy = false,
  businessType = 'retail',
  offlineMode = false
}: {
  onCreated: () => void;
  isPharmacy?: boolean;
  businessType?: string;
  offlineMode?: boolean;
}) {
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [createdProductId, setCreatedProductId] = useState('');
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [units, setUnits] = useState<PosUnit[]>([]);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState('');
  const [openingQuantity, setOpeningQuantity] = useState('');
  const [openingBatchNumber, setOpeningBatchNumber] = useState('');
  const [catalogsLoaded, setCatalogsLoaded] = useState(false);
  const [draftReady, setDraftReady] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  useEffect(() => {
    let active = true;
    void Promise.allSettled([
      posApi<unknown>('/inventory/categories').then((result) => {
        if (active) setCategories(unwrapList<PosCategory>(result.data));
      }),
      posApi<unknown>('/inventory/units').then((result) => {
        if (active)
          setUnits(unwrapList<PosUnit>(result.data).filter((unit) => unit.is_active !== false));
      })
    ]).then(() => {
      if (active) setCatalogsLoaded(true);
    });
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    if (!imageFile) {
      setImagePreview('');
      return;
    }
    const url = URL.createObjectURL(imageFile);
    setImagePreview(url);
    return () => URL.revokeObjectURL(url);
  }, [imageFile]);
  useEffect(() => {
    if (!catalogsLoaded || !formRef.current) return;
    let draft: ProductDraft | null = null;
    try {
      const saved = localStorage.getItem(PRODUCT_DRAFT_KEY);
      if (saved) draft = JSON.parse(saved) as ProductDraft;
    } catch {
      try {
        localStorage.removeItem(PRODUCT_DRAFT_KEY);
      } catch {
        // Ignore malformed saved drafts when storage cleanup is unavailable.
      }
    }

    const form = formRef.current;
    if (draft) {
      for (const [name, value] of Object.entries(draft.values)) {
        const field = form.elements.namedItem(name);
        if (field instanceof HTMLInputElement) {
          if (field.type === 'checkbox') field.checked = Boolean(value);
          else if (field.type !== 'file') field.value = String(value);
        } else if (field instanceof HTMLTextAreaElement || field instanceof HTMLSelectElement) {
          field.value = String(value);
        }
      }
      form.querySelectorAll('details').forEach((details, index) => {
        details.open = draft?.openDetails?.[index] ?? false;
      });
      setOpeningQuantity(String(draft.values.stock ?? ''));
      setOpeningBatchNumber(draft.openingBatchNumber || generateOpeningBatchNumber());
      setCreatedProductId(draft.createdProductId ?? '');
    } else {
      setOpeningBatchNumber(generateOpeningBatchNumber());
    }
    setDraftReady(true);
  }, [catalogsLoaded]);

  function persistDraft(form: HTMLFormElement) {
    if (!draftReady) return;
    const values: ProductDraft['values'] = {};
    for (const field of Array.from(form.elements)) {
      if (
        !(field instanceof HTMLInputElement) &&
        !(field instanceof HTMLTextAreaElement) &&
        !(field instanceof HTMLSelectElement)
      )
        continue;
      if (!field.name) continue;
      if (field instanceof HTMLInputElement) {
        if (field.type !== 'file')
          values[field.name] = field.type === 'checkbox' ? field.checked : field.value;
      } else if (field instanceof HTMLTextAreaElement || field instanceof HTMLSelectElement) {
        values[field.name] = field.value;
      }
    }
    try {
      localStorage.setItem(
        PRODUCT_DRAFT_KEY,
        JSON.stringify({
          values,
          openingBatchNumber:
            (form.elements.namedItem('initial_batch_number') as HTMLInputElement | null)?.value ||
            openingBatchNumber,
          openDetails: Array.from(form.querySelectorAll('details'), (details) => details.open),
          createdProductId: createdProductId || undefined
        } satisfies ProductDraft)
      );
    } catch {
      // Keep form editing available when browser storage is unavailable.
    }
  }

  function closeAndDiscardDraft() {
    try {
      localStorage.removeItem(PRODUCT_DRAFT_KEY);
    } catch {
      // Closing the dialog must work even if browser storage is unavailable.
    }
    onCreated();
  }

  function persistCreatedProduct(id: string) {
    try {
      const saved = localStorage.getItem(PRODUCT_DRAFT_KEY);
      const draft = saved
        ? (JSON.parse(saved) as ProductDraft)
        : { values: {}, openingBatchNumber };
      localStorage.setItem(PRODUCT_DRAFT_KEY, JSON.stringify({ ...draft, createdProductId: id }));
    } catch {
      // The product has already been created; storage failure must not block setup.
    }
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    const raw = Object.fromEntries(new FormData(event.currentTarget));
    const requiresBatchForSubmit = isPharmacy && raw.is_service !== 'on' && Number(raw.stock) > 0;
    if (requiresBatchForSubmit && !String(raw.initial_expiry_date ?? '').trim()) {
      setError('Add an expiry date for the opening medicine quantity.');
      return;
    }
    setPending(true);
    try {
      let image: string | undefined;
      if (imageFile) {
        const upload = new FormData();
        upload.append('file', imageFile);
        upload.append('kind', 'image');
        upload.append('folder', 'products');
        const uploaded = await posApi<{ url: string }>('/uploads', {
          method: 'POST',
          body: upload
        });
        image = uploaded.data.url;
      }
      const created = await posApi<PosProduct>('/inventory/products', {
        method: 'POST',
        body: JSON.stringify({
          name: raw.name,
          unit: raw.unit,
          sku: raw.sku,
          barcode: raw.barcode,
          category_id: raw.category_id || undefined,
          selling_price: Number(raw.selling_price),
          wholesale_price: raw.wholesale_price ? Number(raw.wholesale_price) : undefined,
          wholesale_minimum: Number(raw.wholesale_minimum || 1),
          cost_price: Number(raw.cost_price || 0),
          stock: Number(raw.stock || 0),
          initial_batch_number: openingBatchNumber,
          initial_expiry_date: raw.initial_expiry_date,
          reorder_point: Number(raw.reorder_point || 0),
          description: raw.description,
          image,
          is_service: raw.is_service === 'on',
          is_active: true
        })
      });
      setCreatedProductId(created.data.id);
      persistCreatedProduct(created.data.id);
      if (isPharmacy) {
        await posApi(`/pharmacy/products/${created.data.id}`, {
          method: 'PUT',
          body: JSON.stringify({
            generic_name: raw.generic_name,
            brand_name: raw.brand_name,
            unit: raw.unit,
            pack_size: raw.pack_size,
            requires_prescription: raw.requires_prescription === 'on',
            medicine_schedule: raw.medicine_schedule || 'otc',
            registration_number: raw.registration_number
          })
        });
      }
      try {
        localStorage.removeItem(PRODUCT_DRAFT_KEY);
      } catch {
        // Successful product creation should not depend on browser storage.
      }
      onCreated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create product.');
    } finally {
      setPending(false);
    }
  }
  const needsOpeningBatch = isPharmacy && Number(openingQuantity) > 0;
  return (
    <Dialog open onOpenChange={(open) => !open && closeAndDiscardDraft()}>
      <DialogContent
        className='max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-3xl overflow-hidden p-0 sm:max-h-[calc(100dvh-2rem)] sm:max-w-5xl'
        showCloseButton={false}
      >
        <div className='flex max-h-[calc(100dvh-1rem)] min-h-0 flex-col sm:max-h-[calc(100dvh-2rem)]'>
          <DialogHeader className='flex shrink-0 flex-row items-start justify-between gap-4 border-b bg-card px-5 py-4 text-left'>
            <div className='space-y-1'>
              <DialogTitle>{isPharmacy ? 'Add medicine' : 'Add product'}</DialogTitle>
              <DialogDescription className='hidden sm:block'>
                Enter the product details{isPharmacy ? ' and medicine information' : ''}. Fields
                marked * are required; other details are optional. Prices are in Tanzanian shillings
                (TZS).
              </DialogDescription>
            </div>
            <Button
              type='button'
              variant='ghost'
              size='icon'
              aria-label='Close new product'
              onClick={closeAndDiscardDraft}
            >
              <Icons.close className='size-5' />
            </Button>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-y-auto px-5 py-4'>
            <form
              ref={formRef}
              onChange={(event) => persistDraft(event.currentTarget)}
              onSubmit={submit}
              className='space-y-4'
            >
              <section className='space-y-3 rounded-xl border p-4'>
                <div>
                  <h3 className='font-medium'>Product details</h3>
                  <p className='text-muted-foreground text-sm'>
                    Name the item and choose where it belongs.
                  </p>
                </div>
                <div className='grid grid-cols-[4rem_minmax(0,1fr)] items-start gap-3 sm:grid-cols-[7rem_minmax(0,1fr)] sm:gap-4'>
                  <label className='text-sm sm:row-span-2'>
                    <span className='sr-only'>Product image</span>
                    <span className='bg-muted/30 border-border/70 relative flex aspect-square w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-dashed transition-colors hover:bg-muted/60 sm:w-28'>
                      {imagePreview ? (
                        <img
                          src={imagePreview}
                          alt='Product preview'
                          className='size-full object-cover'
                        />
                      ) : (
                        <span className='text-muted-foreground flex flex-col items-center gap-2 text-center text-xs'>
                          <span className='bg-muted flex size-10 items-center justify-center rounded-full'>
                            <Icons.media className='size-5' />
                          </span>
                          Add photo
                        </span>
                      )}
                      <Input
                        className='absolute inset-0 cursor-pointer opacity-0'
                        type='file'
                        accept='image/jpeg,image/png,image/webp,image/gif'
                        onChange={(event) => setImageFile(event.target.files?.[0] || null)}
                      />
                    </span>
                  </label>
                  <label className='grid gap-1.5 text-sm font-medium'>
                    <span>
                      Medicine or product name <RequiredMark />
                    </span>
                    <Input name='name' required />
                  </label>
                  <div className='col-span-2 grid grid-cols-2 gap-3 sm:col-span-1 sm:col-start-2'>
                    {!offlineMode && (
                      <label className='grid gap-1.5 text-sm font-medium'>
                        <span>
                          Category <RequiredMark />
                        </span>
                        <select
                          name='category_id'
                          required
                          className='bg-background h-10 w-full rounded-md border px-3 font-normal'
                        >
                          <option value=''>Choose a category</option>
                          {categories.map((category) => (
                            <option key={category.id} value={category.id}>
                              {category.name}
                            </option>
                          ))}
                        </select>
                      </label>
                    )}
                    <label
                      className={`grid gap-1.5 text-sm font-medium ${offlineMode ? 'col-span-2' : ''}`}
                    >
                      <span>
                        Unit <RequiredMark />
                      </span>
                      {offlineMode ? (
                        <>
                          <Input name='unit' required defaultValue='piece' maxLength={40} />
                          <span className='text-muted-foreground text-xs'>
                            Use a short unit label such as piece, kg, or bag.
                          </span>
                        </>
                      ) : (
                        <>
                          <select
                            name='unit'
                            defaultValue=''
                            required
                            className='bg-background h-10 w-full rounded-md border px-3 font-normal'
                          >
                            <option value=''>Choose a unit</option>
                            {units.map((unit) => (
                              <option key={unit.id} value={unit.name}>
                                {unit.name}
                              </option>
                            ))}
                          </select>
                          {!units.length && (
                            <span className='text-muted-foreground text-xs'>
                              Add units from the Units page first.
                            </span>
                          )}
                        </>
                      )}
                    </label>
                    <div className='col-span-2 space-y-3 border-t pt-3'>
                      <div className='grid grid-cols-2 gap-3'>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          <span>
                            Cost price (TZS) <RequiredMark />
                          </span>
                          <Input name='cost_price' type='number' min='0' step='1' required />
                        </label>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          <span>
                            Selling price (TZS) <RequiredMark />
                          </span>
                          <Input name='selling_price' type='number' min='0' step='1' required />
                        </label>
                        {(businessType === 'wholesale' || businessType === 'retail_wholesale') && (
                          <>
                            <label className='grid gap-1.5 text-sm font-medium'>
                              Wholesale unit price (TZS)
                              <Input name='wholesale_price' type='number' min='1' step='1' />
                            </label>
                            <label className='grid gap-1.5 text-sm font-medium'>
                              Minimum wholesale quantity
                              <Input
                                name='wholesale_minimum'
                                type='number'
                                min='1'
                                step='1'
                                defaultValue='1'
                              />
                            </label>
                          </>
                        )}
                      </div>
                    </div>
                    <div className='col-span-2 space-y-3 border-t pt-3'>
                      <div className='grid grid-cols-2 gap-3'>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          <span>
                            {isPharmacy ? 'Opening quantity' : 'Opening stock'} <RequiredMark />
                          </span>
                          <Input
                            name='stock'
                            type='number'
                            min='0'
                            step='1'
                            required
                            value={openingQuantity}
                            onChange={(event) => {
                              const quantity = event.target.value;
                              setOpeningQuantity(quantity);
                            }}
                          />
                        </label>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          <span>
                            Low stock alert at <RequiredMark />
                          </span>
                          <Input name='reorder_point' type='number' min='0' step='1' required />
                        </label>
                      </div>
                    </div>
                    {isPharmacy && (
                      <div className='col-span-2 space-y-3 border-t pt-3'>
                        <div className='grid grid-cols-2 gap-3'>
                          <label className='grid gap-1.5 text-sm font-medium'>
                            <span>
                              Batch number{' '}
                              <span className='hidden sm:inline'>(generated automatically)</span>
                            </span>
                            <Input
                              name='initial_batch_number'
                              value={openingBatchNumber}
                              readOnly
                            />
                          </label>
                          <label className='grid gap-1.5 text-sm font-medium'>
                            <span>Expiry date {needsOpeningBatch && <RequiredMark />}</span>
                            <Input name='initial_expiry_date' type='date' />
                          </label>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </section>

              <details
                className='group rounded-xl border bg-muted/20 p-4'
                onToggle={(event) => {
                  const form = event.currentTarget.closest('form');
                  if (form) persistDraft(form);
                }}
              >
                <summary className='cursor-pointer list-none font-medium [&::-webkit-details-marker]:hidden'>
                  <span className='flex items-center justify-between'>
                    More product details
                    <Icons.chevronDown className='size-4 transition-transform group-open:rotate-180' />
                  </span>
                </summary>
                <div className='mt-4 grid gap-3 sm:grid-cols-2'>
                  <label className='grid gap-1.5 text-sm font-medium'>
                    SKU <Input name='sku' />
                  </label>
                  <label className='grid gap-1.5 text-sm font-medium'>
                    Barcode <Input name='barcode' />
                  </label>
                  <label className='grid gap-1.5 text-sm font-medium sm:col-span-2'>
                    Description <Textarea name='description' rows={3} />
                  </label>
                  <label className='flex items-center gap-2 text-sm sm:col-span-2'>
                    <input type='checkbox' name='is_service' /> Service (no stock tracking)
                  </label>
                  {isPharmacy && (
                    <div className='space-y-3 border-t pt-4 sm:col-span-2'>
                      <div>
                        <h3 className='font-medium'>Medicine information</h3>
                        <p className='text-muted-foreground text-sm'>
                          Add identification and dispensing details.
                        </p>
                      </div>
                      <div className='grid gap-3 sm:grid-cols-2'>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          Generic name <Input name='generic_name' />
                        </label>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          Brand name <Input name='brand_name' />
                        </label>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          Pack size <Input name='pack_size' placeholder='e.g. 10 tablets' />
                        </label>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          Medicine schedule
                          <select
                            name='medicine_schedule'
                            defaultValue='otc'
                            className='bg-background h-10 w-full rounded-md border px-3 font-normal'
                          >
                            <option value='unscheduled'>Unclassified</option>
                            <option value='otc'>OTC</option>
                            <option value='addo_permitted'>ADDO permitted</option>
                            <option value='prescription_only'>Prescription only</option>
                          </select>
                        </label>
                        <label className='grid gap-1.5 text-sm font-medium'>
                          Registration number <Input name='registration_number' />
                        </label>
                        <label className='flex items-center gap-2 text-sm sm:col-span-2'>
                          <input type='checkbox' name='requires_prescription' /> Require a
                          prescription reference at checkout
                        </label>
                        <p className='text-muted-foreground text-xs sm:col-span-2'>
                          The medicine schedule is entered by your shop. Confirm the classification
                          against an authoritative source.
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              </details>

              <div className='sticky bottom-0 -mx-5 flex flex-col gap-2 border-t bg-card px-3 py-2 sm:flex-row sm:items-center sm:justify-between sm:px-5 sm:py-4'>
                {error ? (
                  <p role='alert' className='text-destructive min-w-0 flex-1 text-sm'>
                    {error}
                  </p>
                ) : (
                  <p className='text-muted-foreground hidden min-w-0 flex-1 text-xs sm:block'>
                    You can edit these details later.
                  </p>
                )}
                <div className='grid w-full shrink-0 grid-cols-2 gap-2 sm:flex sm:w-auto'>
                  {createdProductId ? (
                    <>
                      <Button
                        type='button'
                        variant='outline'
                        className='w-full sm:w-auto'
                        onClick={closeAndDiscardDraft}
                      >
                        Close
                      </Button>
                      <Link
                        className={buttonVariants({ className: 'w-full sm:w-auto' })}
                        href={`/inventory/${createdProductId}`}
                      >
                        Finish medicine setup
                      </Link>
                    </>
                  ) : (
                    <>
                      <Button
                        type='button'
                        variant='outline'
                        className='w-full sm:w-auto'
                        onClick={closeAndDiscardDraft}
                      >
                        Cancel
                      </Button>
                      <Button type='submit' className='w-full sm:w-auto' isLoading={pending}>
                        {isPharmacy ? 'Create medicine' : 'Create product'}
                      </Button>
                    </>
                  )}
                </div>
              </div>
            </form>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
