'use client';

import Link from 'next/link';
import Image from 'next/image';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  money,
  type PosCategory,
  type PosCustomer,
  type PosPrescriptionDetails,
  type PosProduct
} from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { Dispatch, FormEvent, SetStateAction, useEffect, useState } from 'react';
import { useText } from '@/lib/i18n';

const prescriptionFields: {
  key: Exclude<keyof PosPrescriptionDetails, 'prescriber_authenticity_confirmed'>;
  label: string;
  maxLength: number;
}[] = [
  { key: 'patient_name', label: 'Patient name', maxLength: 200 },
  { key: 'patient_address', label: 'Patient address', maxLength: 500 },
  { key: 'prescription_reference', label: 'Prescription serial / reference', maxLength: 200 },
  { key: 'prescriber_name', label: 'Prescriber name', maxLength: 200 },
  { key: 'prescriber_address', label: 'Prescriber address', maxLength: 500 },
  {
    key: 'prescriber_registration_number',
    label: 'Prescriber registration number',
    maxLength: 100
  },
  { key: 'prescribing_facility_name', label: 'Prescribing facility', maxLength: 200 },
  { key: 'prescribing_facility_address', label: 'Facility address', maxLength: 500 },
  { key: 'medicine_strength', label: 'Medicine strength', maxLength: 100 },
  { key: 'dosage_form', label: 'Dosage form', maxLength: 100 }
];

function ProductImage({ image, name }: { image?: string | null; name: string }) {
  const [failed, setFailed] = useState(false);

  return (
    <div className='pos-product-image'>
      {image && !failed ? (
        <Image
          src={image}
          alt=''
          width={64}
          height={64}
          unoptimized
          className='h-full w-full object-cover'
          onError={() => setFailed(true)}
        />
      ) : (
        <span>{name.slice(0, 1).toUpperCase()}</span>
      )}
    </div>
  );
}

export function CheckoutWorkspace({
  products,
  localMode,
  isPharmacy,
  priceMode,
  setPriceMode,
  canChooseWholesale,
  unitPrice,
  rxDetails,
  setRxDetails,
  filteredProducts,
  categories,
  customers,
  cart,
  setCart,
  search,
  setSearch,
  scanCode,
  setScanCode,
  scanProduct,
  checkout,
  paymentMethod,
  setPaymentMethod,
  customerId,
  setCustomerId,
  discountPct,
  setDiscountPct,
  paymentReference,
  setPaymentReference,
  subtotal,
  discountAmount,
  checkoutTax,
  busy,
  catalogView,
  setCatalogView,
  mobileCartOpen,
  setMobileCartOpen,
  checkoutViewResetKey
}: {
  products: PosProduct[];
  localMode: boolean;
  isPharmacy: boolean;
  priceMode: 'retail' | 'wholesale';
  setPriceMode: Dispatch<SetStateAction<'retail' | 'wholesale'>>;
  canChooseWholesale: boolean;
  unitPrice: (product: PosProduct) => number;
  rxDetails: Record<string, PosPrescriptionDetails>;
  setRxDetails: Dispatch<SetStateAction<Record<string, PosPrescriptionDetails>>>;
  filteredProducts: PosProduct[];
  categories: PosCategory[];
  customers: PosCustomer[];
  cart: Record<string, number>;
  setCart: Dispatch<SetStateAction<Record<string, number>>>;
  search: string;
  setSearch: Dispatch<SetStateAction<string>>;
  scanCode: string;
  setScanCode: Dispatch<SetStateAction<string>>;
  scanProduct: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  checkout: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  paymentMethod: string;
  setPaymentMethod: Dispatch<SetStateAction<string>>;
  customerId: string;
  setCustomerId: Dispatch<SetStateAction<string>>;
  discountPct: string;
  setDiscountPct: Dispatch<SetStateAction<string>>;
  paymentReference: string;
  setPaymentReference: Dispatch<SetStateAction<string>>;
  subtotal: number;
  discountAmount: number;
  checkoutTax: number;
  busy: boolean;
  catalogView: 'grid' | 'list';
  setCatalogView: Dispatch<SetStateAction<'grid' | 'list'>>;
  mobileCartOpen: boolean;
  setMobileCartOpen: Dispatch<SetStateAction<boolean>>;
  checkoutViewResetKey: number;
}) {
  const text = useText();
  const [showCheckoutDetails, setShowCheckoutDetails] = useState(false);
  useEffect(() => {
    // Do not restore payment details from a previous checkout/session. A
    // restored cart must always open at the products review step.
    window.localStorage.removeItem('ziada-checkout-step');
  }, []);
  useEffect(() => {
    if (checkoutViewResetKey === 0) return;
    setShowCheckoutDetails(false);
    setMobileCartOpen(false);
  }, [checkoutViewResetKey, setMobileCartOpen]);
  const visibleProducts = filteredProducts;
  const itemCount = Object.values(cart).reduce((sum, quantity) => sum + quantity, 0);
  function updatePrescriptionField(
    productId: string,
    key: Exclude<keyof PosPrescriptionDetails, 'prescriber_authenticity_confirmed'>,
    value: string
  ) {
    setRxDetails((current) => {
      const previous = current[productId] || {
        prescriber_name: '',
        prescriber_address: '',
        prescriber_registration_number: '',
        prescriber_authenticity_confirmed: false,
        prescribing_facility_name: '',
        prescribing_facility_address: '',
        prescription_reference: '',
        patient_name: '',
        patient_address: '',
        medicine_strength: '',
        dosage_form: ''
      };
      return { ...current, [productId]: { ...previous, [key]: value } };
    });
  }
  const saleForm = (
    <form onSubmit={checkout} className='space-y-4'>
      {canChooseWholesale && (
        <label className='text-sm'>
          Sell at
          <select
            value={priceMode}
            onChange={(event) => setPriceMode(event.target.value as 'retail' | 'wholesale')}
            className='bg-background ml-2 h-9 rounded-md border px-3'
          >
            <option value='retail'>Retail price</option>
            <option value='wholesale'>Wholesale price</option>
          </select>
        </label>
      )}
      {(!showCheckoutDetails ||
        products.some(
          (product) => cart[product.id] && isPharmacy && product.requires_prescription
        )) && (
        <div className='space-y-3 border-b pb-4'>
          {products
            .filter((product) => cart[product.id])
            .map((product) => (
              <div key={product.id} className='space-y-2'>
                {isPharmacy && product.requires_prescription && (
                  <div className='grid gap-2 rounded-md border p-3 sm:grid-cols-2'>
                    <p className='text-sm font-medium sm:col-span-2'>
                      Prescription register · {product.name}
                    </p>
                    {prescriptionFields.map((field) => (
                      <Input
                        key={field.key}
                        aria-label={field.label}
                        placeholder={field.label}
                        maxLength={field.maxLength}
                        required
                        value={rxDetails[product.id]?.[field.key] || ''}
                        onChange={(event) =>
                          updatePrescriptionField(product.id, field.key, event.target.value)
                        }
                      />
                    ))}
                    <label className='flex items-start gap-2 text-sm sm:col-span-2'>
                      <input
                        type='checkbox'
                        aria-label='Confirm prescription authenticity check'
                        required
                        checked={rxDetails[product.id]?.prescriber_authenticity_confirmed || false}
                        onChange={(event) =>
                          setRxDetails((current) => ({
                            ...current,
                            [product.id]: {
                              ...(current[product.id] || {
                                prescriber_name: '',
                                prescriber_address: '',
                                prescriber_registration_number: '',
                                prescribing_facility_name: '',
                                prescribing_facility_address: '',
                                prescription_reference: '',
                                patient_name: '',
                                patient_address: '',
                                medicine_strength: '',
                                dosage_form: ''
                              }),
                              prescriber_authenticity_confirmed: event.target.checked
                            }
                          }))
                        }
                      />
                      I checked the prescriber’s identity/registration and prescription
                      authenticity.
                    </label>
                  </div>
                )}
                {!showCheckoutDetails && (
                  <div className='grid grid-cols-[minmax(0,1fr)_auto_auto_auto_auto] items-center gap-2 text-sm'>
                    <span className='min-w-0 break-words'>
                      {cart[product.id]} × {product.name}
                    </span>
                    <span>{money(cart[product.id] * unitPrice(product))}</span>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='outline'
                      aria-label={`Decrease ${product.name}`}
                      onClick={() =>
                        setCart((current) => ({
                          ...current,
                          [product.id]: Math.max(0, (current[product.id] || 0) - 1)
                        }))
                      }
                    >
                      −
                    </Button>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='outline'
                      aria-label={`Increase ${product.name}`}
                      onClick={() =>
                        setCart((current) => ({
                          ...current,
                          [product.id]: (current[product.id] || 0) + 1
                        }))
                      }
                    >
                      <Icons.add className='size-3.5' />
                    </Button>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      aria-label={`Remove ${product.name}`}
                      onClick={() =>
                        setCart((current) => {
                          const next = { ...current };
                          delete next[product.id];
                          return next;
                        })
                      }
                    >
                      <Icons.trash className='size-3.5' />
                    </Button>
                  </div>
                )}
              </div>
            ))}
        </div>
      )}
      {!showCheckoutDetails && (
        <div className='flex items-center justify-between text-sm font-semibold'>
          <span>{text('Items total', 'Jumla ya bidhaa')}</span>
          <span>{money(subtotal)}</span>
        </div>
      )}
      {!showCheckoutDetails ? (
        <Button
          type='button'
          className='h-11 w-full'
          disabled={!itemCount}
          onClick={() => setShowCheckoutDetails(true)}
        >
          {text('Continue', 'Endelea')}
        </Button>
      ) : (
        <>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            className='-ml-2 w-fit justify-start px-2'
            onClick={() => setShowCheckoutDetails(false)}
          >
            <Icons.chevronLeft className='size-4' />
            {text('Back to items', 'Rudi kwenye bidhaa')}
          </Button>
          <label className='block text-sm font-medium'>
            {text('Customer', 'Mteja')}
            <select
              className='bg-background mt-1 h-11 w-full rounded-md border px-3 text-sm'
              value={customerId}
              onChange={(event) => setCustomerId(event.target.value)}
            >
              <option value=''>{text('Walk-in customer', 'Mteja wa kawaida')}</option>
              {customers.map((customer) => (
                <option key={customer.id} value={customer.id}>
                  {customer.name}
                </option>
              ))}
            </select>
          </label>
          <label className='block text-sm font-medium'>
            {text('Payment method', 'Njia ya malipo')}
            <select
              className='bg-background mt-1 h-11 w-full rounded-md border px-3 text-sm'
              value={paymentMethod}
              onChange={(event) => setPaymentMethod(event.target.value)}
            >
              {(localMode ? ['Cash'] : ['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit']).map(
                (method) => (
                  <option key={method}>{method}</option>
                )
              )}
            </select>
          </label>
          <div className='grid grid-cols-2 gap-2'>
            <label className='text-sm'>
              {text('Discount %', 'Punguzo %')}
              <Input
                className='mt-1 h-11'
                min='0'
                max='100'
                type='number'
                value={discountPct}
                onChange={(event) => setDiscountPct(event.target.value)}
              />
            </label>
            {!localMode && (
              <label className='text-sm'>
                {text('Reference', 'Kumbukumbu')}
                <Input
                  className='mt-1 h-11'
                  value={paymentReference}
                  onChange={(event) => setPaymentReference(event.target.value)}
                  placeholder='Optional'
                />
              </label>
            )}
          </div>
          <div className='border-t pt-3 text-sm'>
            <div className='flex justify-between'>
              <span>{text('Subtotal', 'Jumla ndogo')}</span>
              <span>{money(subtotal)}</span>
            </div>
            <div className='flex justify-between'>
              <span>Discount</span>
              <span>{money(discountAmount)}</span>
            </div>
            <div className='flex justify-between'>
              <span>{text('Tax (18%)', 'Kodi (18%)')}</span>
              <span>{money(checkoutTax)}</span>
            </div>
            <div className='mt-2 flex justify-between font-semibold'>
              <span>{text('Total', 'Jumla')}</span>
              <span>{money(subtotal - discountAmount + checkoutTax)}</span>
            </div>
          </div>
          <Button type='submit' className='h-11 w-full' isLoading={busy} disabled={!itemCount}>
            {text('Complete sale', 'Kamilisha mauzo')}
          </Button>
        </>
      )}
    </form>
  );
  const controls = (
    <div className='flex shrink-0 gap-1'>
      <Button
        size='icon'
        variant={catalogView === 'grid' ? 'default' : 'outline'}
        aria-label={text('Grid view', 'Mwonekano wa gridi')}
        onClick={() => setCatalogView('grid')}
      >
        <Icons.dashboard className='h-4 w-4' />
      </Button>
      <Button
        size='icon'
        variant={catalogView === 'list' ? 'default' : 'outline'}
        aria-label={text('List view', 'Mwonekano wa orodha')}
        onClick={() => setCatalogView('list')}
      >
        <Icons.page className='h-4 w-4' />
      </Button>
    </div>
  );
  return (
    <>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0 flex-1'>
          <h1 className='text-2xl font-bold tracking-tight'>{text('Products', 'Bidhaa')}</h1>
          <p className='text-muted-foreground'>
            Scan or search products, select payment, and complete the sale.
          </p>
        </div>
        <div className='shrink-0'>{controls}</div>
      </div>
      <div className='grid min-h-0 flex-1 gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,420px)]'>
        <Card className='min-w-0'>
          <CardHeader>
            <div className='pos-search-row flex items-center gap-2'>
              <Input
                className='min-w-0 flex-1'
                aria-label={text('Search products', 'Tafuta bidhaa')}
                placeholder='Search name, SKU, or barcode…'
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
              <form onSubmit={scanProduct} className='pos-scan-form flex shrink-0 gap-2'>
                <Input
                  className='hidden'
                  aria-label='Scan barcode'
                  placeholder='Scan barcode…'
                  value={scanCode}
                  onChange={(event) => setScanCode(event.target.value)}
                />
                <Button type='submit' variant='outline' size='icon' aria-label='Scan barcode'>
                  <Icons.barcode className='h-4 w-4' />
                </Button>
              </form>
              <Button
                className='relative xl:hidden'
                size='icon'
                variant='outline'
                aria-label={`Open current sale, ${itemCount} items`}
                onClick={() => {
                  // Always show the selected products first when reopening the
                  // mobile cart, even if payment details were cached earlier.
                  setShowCheckoutDetails(false);
                  setMobileCartOpen(true);
                }}
              >
                <Icons.cart className='h-4 w-4' />
                {itemCount > 0 && (
                  <span className='bg-primary text-primary-foreground absolute -right-1 -top-1 min-w-4 rounded-full px-1 text-[10px]'>
                    {itemCount}
                  </span>
                )}
              </Button>
            </div>
            <div className='pos-category-rail' aria-label='Product categories'>
              <button type='button' className='pos-category-chip pos-category-chip-active'>
                All
              </button>
              {categories.map((category) => (
                <button type='button' key={category.id} className='pos-category-chip'>
                  {category.name}
                </button>
              ))}
            </div>
          </CardHeader>
          <CardContent
            data-catalog-view={catalogView}
            className='pos-product-catalog grid max-h-[calc(100svh-18rem)] gap-3 overflow-y-auto'
          >
            {visibleProducts.length ? (
              visibleProducts.map((product) => {
                const canAddToSale =
                  (!isPharmacy || product.is_service || product.medicine_profile_configured) &&
                  (product.is_service || product.stock > 0);
                const addToSale = () =>
                  setCart((current) => ({
                    ...current,
                    [product.id]: product.is_service
                      ? (current[product.id] || 0) + 1
                      : Math.min(current[product.id] || 0, product.stock - 1) + 1
                  }));

                return (
                  <div key={product.id} className='pos-product-card'>
                    {canAddToSale ? (
                      <button
                        type='button'
                        className='pos-product-card-button'
                        aria-label={`${text('Add', 'Ongeza')} ${product.name} ${text('to sale', 'kwenye mauzo')}`}
                        onClick={addToSale}
                      >
                        <ProductImage image={product.image} name={product.name} />
                        <div className='pos-product-info'>
                          <span className='pos-product-name font-medium'>{product.name}</span>
                          <span className='pos-product-meta text-muted-foreground'>
                            {product.sku || product.barcode || text('No SKU', 'Hakuna SKU')} ·{' '}
                            {text('stock', 'stoo')}{' '}
                            {product.is_service ? text('service', 'huduma') : product.stock}
                          </span>
                          <span className='pos-product-price font-semibold'>
                            {money(unitPrice(product))}
                          </span>
                        </div>
                      </button>
                    ) : (
                      <div className='pos-product-card-unavailable'>
                        <ProductImage image={product.image} name={product.name} />
                        <div className='pos-product-info'>
                          <span className='pos-product-name font-medium'>{product.name}</span>
                          <span className='pos-product-meta text-muted-foreground'>
                            {product.sku || product.barcode || text('No SKU', 'Hakuna SKU')} ·{' '}
                            {text('stock', 'stoo')}{' '}
                            {product.is_service ? text('service', 'huduma') : product.stock}
                          </span>
                          <span className='pos-product-price font-semibold'>
                            {money(unitPrice(product))}
                          </span>
                        </div>
                        {isPharmacy &&
                          !product.is_service &&
                          !product.medicine_profile_configured && (
                            <Link
                              className='text-primary text-xs underline underline-offset-4'
                              href={`/inventory/${product.id}`}
                            >
                              Configure medicine details
                            </Link>
                          )}
                      </div>
                    )}
                  </div>
                );
              })
            ) : (
              <div className='text-muted-foreground flex min-h-40 items-center justify-center text-center text-sm'>
                {products.length
                  ? text(
                      'No products match your search.',
                      'Hakuna bidhaa zinazolingana na utafutaji wako.'
                    )
                  : text(
                      'No products are available at this store yet.',
                      'Hakuna bidhaa katika stoo hii bado.'
                    )}
              </div>
            )}
          </CardContent>
        </Card>
        <Card className='hidden xl:sticky xl:top-20 xl:block xl:max-h-[calc(100svh-6rem)] xl:self-start xl:overflow-y-auto'>
          <CardHeader>
            <div className='flex items-center justify-between gap-3'>
              <CardTitle>{text('Current sale', 'Mauzo ya sasa')}</CardTitle>
              <CardDescription>
                {itemCount} {text('items', 'bidhaa')}
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent>{saleForm}</CardContent>
        </Card>
      </div>
      {mobileCartOpen && (
        <div className='fixed inset-0 z-50 xl:hidden'>
          <button
            type='button'
            aria-label={text('Close current sale', 'Funga mauzo ya sasa')}
            className='absolute inset-0 h-full w-full bg-black/50'
            onClick={() => setMobileCartOpen(false)}
          />
          <section
            role='dialog'
            aria-modal='true'
            aria-label={text('Current sale', 'Mauzo ya sasa')}
            className='bg-background absolute inset-x-0 bottom-0 max-h-[90svh] overflow-y-auto rounded-t-2xl border p-4 shadow-2xl'
          >
            <div className='mb-4 flex items-center justify-between'>
              <div className='flex min-w-0 items-center gap-3'>
                <h2 className='text-lg font-semibold'>{text('Current sale', 'Mauzo ya sasa')}</h2>
                <p className='text-muted-foreground text-sm whitespace-nowrap'>
                  {itemCount} {text('items', 'bidhaa')}
                </p>
              </div>
              <Button size='icon' variant='ghost' onClick={() => setMobileCartOpen(false)}>
                <Icons.close className='h-5 w-5' />
              </Button>
            </div>
            {saleForm}
          </section>
        </div>
      )}
    </>
  );
}
