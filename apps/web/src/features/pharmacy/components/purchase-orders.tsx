'use client';

import { useEffect, useState, type FormEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { posApi, unwrapList } from '@/lib/pos-api';

type PurchaseLine = {
  id: string;
  product_id: string;
  ordered_quantity: number;
  received_quantity: number;
  unit_cost: number;
};
type PurchaseOrder = {
  id: string;
  order_number: string;
  supplier_id: string;
  status: string;
  lines: PurchaseLine[];
};
type Product = { id: string; name: string };
type Supplier = { id: string; name: string; status?: string };
type DraftLine = { product_id: string; quantity: string; unit_cost: string };
type ReorderSuggestion = {
  product_id: string;
  product_name: string;
  sku: string;
  supplier_id?: string;
  supplier_name?: string;
  sales_last_30_days: number;
  daily_velocity: number;
  lead_time_days: number;
  safety_stock: number;
  current_stock: number;
  lead_time_demand: number;
  suggested_quantity: number;
  unit_cost: number;
  reason: string;
};

export function PharmacyPurchaseOrders() {
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [suggestions, setSuggestions] = useState<ReorderSuggestion[]>([]);
  const [selectedProducts, setSelectedProducts] = useState<string[]>([]);
  const [supplierAssignments, setSupplierAssignments] = useState<Record<string, string>>({});
  const [suggestedQuantities, setSuggestedQuantities] = useState<Record<string, string>>({});
  const [unitCosts, setUnitCosts] = useState<Record<string, string>>({});
  const [bulkPending, setBulkPending] = useState(false);
  const [draftLines, setDraftLines] = useState<DraftLine[]>([
    { product_id: '', quantity: '1', unit_cost: '' }
  ]);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [reload, setReload] = useState(0);

  useEffect(() => {
    Promise.all([
      posApi<unknown>('/suppliers/purchase-orders'),
      posApi<unknown>('/inventory/products?page=1&page_size=100'),
      posApi<unknown>('/suppliers?page=1&page_size=100'),
      posApi<unknown>('/suppliers/reorder-suggestions')
    ])
      .then(([ordersResult, productsResult, suppliersResult, suggestionResult]) => {
        setOrders(unwrapList<PurchaseOrder>(ordersResult.data));
        setProducts(unwrapList<Product>(productsResult.data));
        const supplierItems = unwrapList<Supplier>(suppliersResult.data);
        setSuppliers(supplierItems);
        const reorderItems = unwrapList<ReorderSuggestion>(suggestionResult.data);
        setSuggestions(reorderItems);
        const activeSupplierIDs = new Set(
          supplierItems
            .filter((supplier) => !supplier.status || supplier.status === 'active')
            .map((supplier) => supplier.id)
        );
        setSupplierAssignments(
          Object.fromEntries(
            reorderItems.map((item) => [
              item.product_id,
              item.supplier_id && activeSupplierIDs.has(item.supplier_id) ? item.supplier_id : ''
            ])
          )
        );
        setSuggestedQuantities(
          Object.fromEntries(
            reorderItems.map((item) => [item.product_id, String(item.suggested_quantity)])
          )
        );
        setUnitCosts(
          Object.fromEntries(reorderItems.map((item) => [item.product_id, String(item.unit_cost)]))
        );
      })
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load purchasing data.')
      );
  }, [reload]);

  async function createOrder(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    setMessage('');
    const form = new FormData(event.currentTarget);
    try {
      await posApi('/suppliers/purchase-orders', {
        method: 'POST',
        body: JSON.stringify({
          supplier_id: form.get('supplier_id'),
          notes: form.get('notes'),
          lines: draftLines.map((line) => ({
            product_id: line.product_id,
            quantity: Number(line.quantity),
            unit_cost: Number(line.unit_cost)
          }))
        })
      });
      setDraftLines([{ product_id: '', quantity: '1', unit_cost: '' }]);
      setMessage('Purchase order created.');
      setReload((value) => value + 1);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create purchase order.');
    }
  }

  async function receiveOrder(event: FormEvent<HTMLFormElement>, order: PurchaseOrder) {
    event.preventDefault();
    setError('');
    setMessage('');
    const values = new FormData(event.currentTarget);
    const lines = order.lines.flatMap((line) => {
      const quantity = Number(values.get(`quantity:${line.id}`) || 0);
      if (!quantity) return [];
      return [
        {
          line_id: line.id,
          quantity,
          batch_number: values.get(`batch:${line.id}`),
          expiry_date: values.get(`expiry:${line.id}`)
        }
      ];
    });
    if (!lines.length) {
      setError('Enter a quantity to receive for at least one product.');
      return;
    }
    try {
      await posApi(`/suppliers/purchase-orders/${order.id}/receive`, {
        method: 'POST',
        body: JSON.stringify({ invoice_number: values.get('invoice_number'), lines })
      });
      setMessage(`${order.order_number} received.`);
      setReload((value) => value + 1);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not receive purchase order.');
    }
  }

  async function createSuggestedOrders() {
    setError('');
    setMessage('');
    const selected = suggestions.filter((item) => selectedProducts.includes(item.product_id));
    if (!selected.length) {
      setError('Select at least one medicine to create purchase orders.');
      return;
    }
    const groups = new Map<string, { product_id: string; quantity: number; unit_cost: number }[]>();
    for (const item of selected) {
      const supplierId = supplierAssignments[item.product_id];
      const quantity = Number(suggestedQuantities[item.product_id]);
      const unitCost = Number(unitCosts[item.product_id]);
      if (!supplierId) {
        setError(`Choose a supplier for ${item.product_name}.`);
        return;
      }
      if (
        !Number.isInteger(quantity) ||
        quantity < 1 ||
        !Number.isSafeInteger(unitCost) ||
        unitCost < 0
      ) {
        setError(`Enter a valid quantity and unit cost for ${item.product_name}.`);
        return;
      }
      const lines = groups.get(supplierId) || [];
      lines.push({ product_id: item.product_id, quantity, unit_cost: unitCost });
      groups.set(supplierId, lines);
    }
    setBulkPending(true);
    try {
      const result = await posApi<{ items: PurchaseOrder[] }>('/suppliers/purchase-orders/bulk', {
        method: 'POST',
        body: JSON.stringify({
          groups: Array.from(groups, ([supplier_id, lines]) => ({ supplier_id, lines }))
        })
      });
      setSelectedProducts([]);
      setMessage(`${result.data.items.length} supplier purchase order(s) created.`);
      setReload((value) => value + 1);
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : 'Could not create grouped purchase orders.'
      );
    } finally {
      setBulkPending(false);
    }
  }

  return (
    <section className='space-y-5'>
      <Card>
        <CardHeader>
          <CardTitle>Pharmacy purchasing</CardTitle>
          <CardDescription>
            Create supplier orders, then receive deliveries into traceable medicine batches.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={createOrder} className='space-y-4'>
            <select
              required
              name='supplier_id'
              className='bg-background h-10 w-full rounded-md border px-3 text-sm'
            >
              <option value=''>Choose supplier</option>
              {suppliers.map((supplier) => (
                <option key={supplier.id} value={supplier.id}>
                  {supplier.name}
                </option>
              ))}
            </select>
            <Input name='notes' placeholder='Order notes (optional)' />
            <div className='space-y-3'>
              {draftLines.map((line, index) => (
                <div key={index} className='grid gap-2 sm:grid-cols-[2fr_1fr_1fr_auto]'>
                  <select
                    required
                    value={line.product_id}
                    onChange={(event) =>
                      setDraftLines((current) =>
                        current.map((item, i) =>
                          i === index ? { ...item, product_id: event.target.value } : item
                        )
                      )
                    }
                    className='bg-background h-10 rounded-md border px-3 text-sm'
                  >
                    <option value=''>Choose medicine</option>
                    {products.map((product) => (
                      <option key={product.id} value={product.id}>
                        {product.name}
                      </option>
                    ))}
                  </select>
                  <Input
                    aria-label='Ordered quantity'
                    required
                    type='number'
                    min='1'
                    value={line.quantity}
                    onChange={(event) =>
                      setDraftLines((current) =>
                        current.map((item, i) =>
                          i === index ? { ...item, quantity: event.target.value } : item
                        )
                      )
                    }
                  />
                  <Input
                    aria-label='Unit cost'
                    required
                    type='number'
                    min='0'
                    placeholder='Unit cost'
                    value={line.unit_cost}
                    onChange={(event) =>
                      setDraftLines((current) =>
                        current.map((item, i) =>
                          i === index ? { ...item, unit_cost: event.target.value } : item
                        )
                      )
                    }
                  />
                  {draftLines.length > 1 && (
                    <Button
                      type='button'
                      variant='ghost'
                      onClick={() =>
                        setDraftLines((current) => current.filter((_, i) => i !== index))
                      }
                    >
                      Remove
                    </Button>
                  )}
                </div>
              ))}
            </div>
            <div className='flex flex-wrap gap-2'>
              <Button
                type='button'
                variant='outline'
                onClick={() =>
                  setDraftLines((current) => [
                    ...current,
                    { product_id: '', quantity: '1', unit_cost: '' }
                  ])
                }
              >
                Add product
              </Button>
              <Button>Create purchase order</Button>
            </div>
          </form>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Suggested reorder</CardTitle>
          <CardDescription>
            Calculated from the last 30 days of sales, supplier lead time, safety stock, and
            unexpired batch stock.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error && (
            <p role='alert' className='text-destructive mb-3 text-sm'>
              {error}
            </p>
          )}
          {message && (
            <p role='status' className='text-muted-foreground mb-3 text-sm'>
              {message}
            </p>
          )}
          {suggestions.length ? (
            <div className='overflow-x-auto rounded-lg border'>
              <table className='w-full min-w-[760px] text-left text-sm'>
                <thead className='bg-muted/50'>
                  <tr>
                    {[
                      'Select',
                      'Medicine',
                      'Supplier',
                      'Sellable',
                      'Suggested',
                      'Order quantity',
                      'Unit cost',
                      'Calculation'
                    ].map((heading) => (
                      <th key={heading} className='px-3 py-2 font-medium'>
                        {heading}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className='divide-y'>
                  {suggestions.map((suggestion) => (
                    <tr key={suggestion.product_id}>
                      <td className='px-3 py-2'>
                        <input
                          type='checkbox'
                          aria-label={`Select ${suggestion.product_name}`}
                          checked={selectedProducts.includes(suggestion.product_id)}
                          onChange={(event) =>
                            setSelectedProducts((current) =>
                              event.target.checked
                                ? [...current, suggestion.product_id]
                                : current.filter((id) => id !== suggestion.product_id)
                            )
                          }
                        />
                      </td>
                      <td className='px-3 py-2 font-medium'>
                        {suggestion.product_name}
                        {suggestion.sku ? (
                          <span className='text-muted-foreground ml-2 text-xs'>
                            {suggestion.sku}
                          </span>
                        ) : null}
                      </td>
                      <td className='px-3 py-2'>
                        <select
                          aria-label={`Supplier for ${suggestion.product_name}`}
                          value={supplierAssignments[suggestion.product_id] || ''}
                          onChange={(event) =>
                            setSupplierAssignments((current) => ({
                              ...current,
                              [suggestion.product_id]: event.target.value
                            }))
                          }
                          className='bg-background h-9 min-w-36 rounded-md border px-2'
                        >
                          <option value=''>Choose supplier</option>
                          {suppliers
                            .filter((supplier) => !supplier.status || supplier.status === 'active')
                            .map((supplier) => (
                              <option key={supplier.id} value={supplier.id}>
                                {supplier.name}
                              </option>
                            ))}
                        </select>
                      </td>
                      <td className='px-3 py-2 tabular-nums'>{suggestion.current_stock}</td>
                      <td className='px-3 py-2 font-semibold tabular-nums'>
                        {suggestion.suggested_quantity}
                      </td>
                      <td className='px-3 py-2'>
                        <Input
                          aria-label={`Order quantity for ${suggestion.product_name}`}
                          type='number'
                          min='1'
                          step='1'
                          value={suggestedQuantities[suggestion.product_id] || ''}
                          onChange={(event) =>
                            setSuggestedQuantities((current) => ({
                              ...current,
                              [suggestion.product_id]: event.target.value
                            }))
                          }
                          className='w-28'
                        />
                      </td>
                      <td className='px-3 py-2'>
                        <Input
                          aria-label={`Unit cost for ${suggestion.product_name}`}
                          type='number'
                          min='0'
                          step='1'
                          value={unitCosts[suggestion.product_id] || ''}
                          onChange={(event) =>
                            setUnitCosts((current) => ({
                              ...current,
                              [suggestion.product_id]: event.target.value
                            }))
                          }
                          className='w-32'
                        />
                      </td>
                      <td className='text-muted-foreground max-w-sm px-3 py-2 text-xs'>
                        {suggestion.reason}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className='text-muted-foreground text-sm'>No products need reordering right now.</p>
          )}
          {!!suggestions.length && (
            <div className='mt-4 flex flex-wrap items-center gap-3'>
              <Button
                type='button'
                variant='outline'
                onClick={() => setSelectedProducts(suggestions.map((item) => item.product_id))}
              >
                Select all
              </Button>
              <Button
                type='button'
                onClick={() => void createSuggestedOrders()}
                disabled={!selectedProducts.length || bulkPending}
              >
                {bulkPending
                  ? 'Creating orders…'
                  : `Generate purchase orders (${selectedProducts.length})`}
              </Button>
              <span className='text-muted-foreground text-xs'>
                Orders are grouped by supplier; quantities and unit costs are editable above.
              </span>
            </div>
          )}
        </CardContent>
      </Card>
      {orders.map((order) => (
        <Card key={order.id}>
          <CardHeader>
            <CardTitle className='text-base'>
              {order.order_number} · {order.status}
            </CardTitle>
            <CardDescription>{order.lines?.length || 0} ordered product line(s)</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={(event) => void receiveOrder(event, order)} className='space-y-4'>
              <Input name='invoice_number' placeholder='Supplier invoice number (optional)' />
              {order.lines?.map((line) => {
                const remaining = line.ordered_quantity - line.received_quantity;
                const product = products.find((item) => item.id === line.product_id);
                return (
                  <div
                    key={line.id}
                    className='grid gap-2 rounded-md border p-3 sm:grid-cols-[2fr_1fr_1fr_1fr]'
                  >
                    <div className='text-sm'>
                      <p className='font-medium'>{product?.name || 'Medicine'}</p>
                      <p className='text-muted-foreground'>
                        Received {line.received_quantity} / {line.ordered_quantity}; {remaining}{' '}
                        remaining
                      </p>
                    </div>
                    {remaining > 0 ? (
                      <>
                        <Input
                          name={`quantity:${line.id}`}
                          type='number'
                          min='0'
                          max={remaining}
                          defaultValue='0'
                          aria-label='Receive quantity'
                        />
                        <Input
                          name={`batch:${line.id}`}
                          placeholder='Batch number'
                          aria-label='Batch number'
                        />
                        <Input name={`expiry:${line.id}`} type='date' aria-label='Expiry date' />
                      </>
                    ) : (
                      <p className='text-muted-foreground self-center text-sm sm:col-span-3'>
                        Fully received
                      </p>
                    )}
                  </div>
                );
              })}
              {order.status !== 'received' && <Button>Receive selected quantities</Button>}
            </form>
          </CardContent>
        </Card>
      ))}
      {(message || error) && (
        <p
          role={error ? 'alert' : 'status'}
          className={`text-sm ${error ? 'text-destructive' : 'text-muted-foreground'}`}
        >
          {error || message}
        </p>
      )}
    </section>
  );
}
