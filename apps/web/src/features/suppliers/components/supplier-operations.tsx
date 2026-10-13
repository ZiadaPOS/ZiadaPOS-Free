'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList } from '@/lib/pos-api';
import { FormEvent, useEffect, useState } from 'react';
import { PharmacyPurchaseOrders } from '@/features/pharmacy/components/purchase-orders';

export function SupplierOperations() {
  const [suppliers, setSuppliers] = useState<Record<string, unknown>[]>([]);
  const [products, setProducts] = useState<Record<string, unknown>[]>([]);
  const [isPharmacy, setIsPharmacy] = useState(false);
  const [lines, setLines] = useState([
    { product_id: '', quantity: '1', unit_cost: '', batch_number: '', expiry_date: '' }
  ]);
  const [message, setMessage] = useState('');
  const [reload, setReload] = useState(0);
  useEffect(() => {
    Promise.all([
      posApi<unknown>('/suppliers?page=1&page_size=100'),
      posApi<{ organisation?: { business_type?: string } }>('/accounts/me')
    ])
      .then(async ([supplierResult, profile]) => {
        setSuppliers(unwrapList<Record<string, unknown>>(supplierResult.data));
        const pharmacy = profile.data.organisation?.business_type === 'pharmacy';
        setIsPharmacy(pharmacy);
        if (pharmacy) {
          const productsResult = await posApi<unknown>('/inventory/products?page=1&page_size=100');
          setProducts(unwrapList<Record<string, unknown>>(productsResult.data));
        }
      })
      .catch((reason) =>
        setMessage(reason instanceof Error ? reason.message : 'Could not load suppliers.')
      );
  }, [reload]);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = Object.fromEntries(new FormData(form));
    const supplierId = String(values.supplier_id || '');
    try {
      if (values.operation === 'delivery')
        await posApi<unknown>(`/suppliers/${supplierId}/deliveries`, {
          method: 'POST',
          body: JSON.stringify({
            invoice_number: values.invoice_number,
            delivery_date: new Date(`${values.delivery_date}T00:00:00`).toISOString(),
            total_value: Number(values.total_value),
            items_count: Number(values.items_count || 1),
            lines: isPharmacy
              ? lines.map((line) => ({
                  ...line,
                  quantity: Number(line.quantity),
                  unit_cost: Number(line.unit_cost)
                }))
              : undefined,
            is_paid: false,
            notes: values.notes
          })
        });
      else
        await posApi<unknown>(`/suppliers/${supplierId}/pay`, {
          method: 'POST',
          body: JSON.stringify({
            amount: Number(values.amount),
            payment_method: values.payment_method,
            reference: values.reference,
            notes: values.notes
          })
        });
      setMessage('Supplier record saved.');
      setReload((value) => value + 1);
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not save supplier record.');
    }
  }
  return (
    <div className='space-y-5'>
      {isPharmacy && <PharmacyPurchaseOrders />}
      <Card>
        <CardHeader>
          <CardTitle>Deliveries and payments</CardTitle>
          <CardDescription>Record an invoice delivery or pay a supplier balance.</CardDescription>
        </CardHeader>
        <CardContent className='grid gap-6 lg:grid-cols-2'>
          <form onSubmit={submit} className='space-y-3'>
            <input type='hidden' name='operation' value='delivery' />
            <label className='block text-sm'>
              Supplier
              <select
                className='bg-background mt-1 h-9 w-full rounded-md border px-3'
                name='supplier_id'
                required
              >
                {suppliers.map((s) => (
                  <option value={String(s.id)} key={String(s.id)}>
                    {String(s.name)}
                  </option>
                ))}
              </select>
            </label>
            <label className='block text-sm'>
              Invoice number
              <Input name='invoice_number' />
            </label>
            <label className='block text-sm'>
              Delivery date
              <Input name='delivery_date' type='date' required />
            </label>
            <label className='block text-sm'>
              Invoice total (TZS)
              <Input name='total_value' type='number' min='1' required />
            </label>
            <label className='block text-sm'>
              Items count
              <Input name='items_count' type='number' min='1' defaultValue='1' />
            </label>
            {isPharmacy && (
              <section className='space-y-3 rounded-md border p-3'>
                <div className='flex items-center justify-between'>
                  <p className='text-sm font-medium'>Received medicine batches</p>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() =>
                      setLines((current) => [
                        ...current,
                        {
                          product_id: '',
                          quantity: '1',
                          unit_cost: '',
                          batch_number: '',
                          expiry_date: ''
                        }
                      ])
                    }
                  >
                    Add line
                  </Button>
                </div>
                {lines.map((line, index) => (
                  <div className='grid gap-2 rounded border p-2 sm:grid-cols-2' key={index}>
                    <select
                      required
                      value={line.product_id}
                      onChange={(event) =>
                        setLines((current) =>
                          current.map((item, i) =>
                            i === index ? { ...item, product_id: event.target.value } : item
                          )
                        )
                      }
                      className='bg-background h-9 rounded-md border px-2 text-sm sm:col-span-2'
                    >
                      <option value=''>Select medicine</option>
                      {products.map((product) => (
                        <option key={String(product.id)} value={String(product.id)}>
                          {String(product.name)}
                        </option>
                      ))}
                    </select>
                    <Input
                      aria-label='Quantity received'
                      type='number'
                      min='1'
                      required
                      placeholder='Quantity'
                      value={line.quantity}
                      onChange={(event) =>
                        setLines((current) =>
                          current.map((item, i) =>
                            i === index ? { ...item, quantity: event.target.value } : item
                          )
                        )
                      }
                    />
                    <Input
                      aria-label='Unit cost'
                      type='number'
                      min='0'
                      required
                      placeholder='Unit cost (TZS)'
                      value={line.unit_cost}
                      onChange={(event) =>
                        setLines((current) =>
                          current.map((item, i) =>
                            i === index ? { ...item, unit_cost: event.target.value } : item
                          )
                        )
                      }
                    />
                    <Input
                      aria-label='Batch number'
                      required
                      placeholder='Batch number'
                      value={line.batch_number}
                      onChange={(event) =>
                        setLines((current) =>
                          current.map((item, i) =>
                            i === index ? { ...item, batch_number: event.target.value } : item
                          )
                        )
                      }
                    />
                    <Input
                      aria-label='Expiry date'
                      type='date'
                      required
                      value={line.expiry_date}
                      onChange={(event) =>
                        setLines((current) =>
                          current.map((item, i) =>
                            i === index ? { ...item, expiry_date: event.target.value } : item
                          )
                        )
                      }
                    />
                    {lines.length > 1 && (
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='sm:col-span-2'
                        onClick={() => setLines((current) => current.filter((_, i) => i !== index))}
                      >
                        Remove line
                      </Button>
                    )}
                  </div>
                ))}
              </section>
            )}
            <Button>Record delivery</Button>
          </form>
          <form onSubmit={submit} className='space-y-3'>
            <input type='hidden' name='operation' value='payment' />
            <label className='block text-sm'>
              Supplier
              <select
                className='bg-background mt-1 h-9 w-full rounded-md border px-3'
                name='supplier_id'
                required
              >
                {suppliers.map((s) => (
                  <option value={String(s.id)} key={String(s.id)}>
                    {String(s.name)}
                  </option>
                ))}
              </select>
            </label>
            <label className='block text-sm'>
              Payment amount (TZS)
              <Input name='amount' type='number' min='1' required />
            </label>
            <label className='block text-sm'>
              Method
              <select
                name='payment_method'
                className='bg-background mt-1 h-9 w-full rounded-md border px-3'
              >
                <option value='cash'>Cash</option>
                <option value='bank'>Bank</option>
                <option value='mobile'>Mobile</option>
              </select>
            </label>
            <label className='block text-sm'>
              Reference
              <Input name='reference' />
            </label>
            <Button>Record payment</Button>
          </form>
          {message && (
            <p role='status' className='text-muted-foreground text-sm lg:col-span-2'>
              {message}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
