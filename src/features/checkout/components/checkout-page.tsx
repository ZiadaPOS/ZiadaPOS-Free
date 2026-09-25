'use client';

import PageContainer from '@/components/layout/page-container';
import {
  posApi,
  unwrapList,
  type PosCategory,
  type PosCustomer,
  type PosProduct,
  type PosTransaction
} from '@/lib/pos-api';
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { CheckoutWorkspace } from '@/features/checkout/components/checkout-workspace';
import { ReceiptDialog } from '@/features/sales/components/receipt-dialog';

export default function CheckoutPage() {
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [customers, setCustomers] = useState<PosCustomer[]>([]);
  const [search, setSearch] = useState('');
  const [cart, setCart] = useState<Record<string, number>>({});
  const [paymentMethod, setPaymentMethod] = useState('Cash');
  const [customerId, setCustomerId] = useState('');
  const [discountPct, setDiscountPct] = useState('0');
  const [paymentReference, setPaymentReference] = useState('');
  const [catalogView, setCatalogView] = useState<'grid' | 'list'>('grid');
  const [mobileCartOpen, setMobileCartOpen] = useState(false);
  const [scanCode, setScanCode] = useState('');
  const [receiptTransaction, setReceiptTransaction] = useState<PosTransaction | null>(null);
  const load = useCallback(async () => {
    setError('');
    try {
      const [p, c, categoryResult] = await Promise.all([
        posApi<unknown>('/inventory/products?status=active&page=1&page_size=100'),
        posApi<unknown>('/customers?page=1&page_size=100'),
        posApi<unknown>('/inventory/categories')
      ]);
      setProducts(unwrapList<PosProduct>(p.data));
      setCustomers(unwrapList<PosCustomer>(c.data));
      setCategories(unwrapList<PosCategory>(categoryResult.data));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load data.');
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    document.documentElement.dataset.catalogView = catalogView;
    return () => {
      delete document.documentElement.dataset.catalogView;
    };
  }, [catalogView]);
  const filteredProducts = useMemo(
    () =>
      products.filter((product) =>
        `${product.name} ${product.sku} ${product.barcode}`
          .toLowerCase()
          .includes(search.toLowerCase())
      ),
    [products, search]
  );
  const subtotal = products.reduce((sum, p) => sum + (cart[p.id] || 0) * p.selling_price, 0);
  const discountAmount = Math.floor((subtotal * Number(discountPct || 0)) / 100);
  const checkoutTax = Math.floor((subtotal - discountAmount) * 0.18);

  async function scanProduct(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!scanCode.trim()) return;
    try {
      const response = await posApi<PosProduct>(
        `/inventory/products/barcode/${encodeURIComponent(scanCode.trim())}`
      );
      setCart((current) => ({
        ...current,
        [response.data.id]: (current[response.data.id] || 0) + 1
      }));
      setScanCode('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Product barcode was not found.');
    }
  }

  async function checkout(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    try {
      const lines = Object.entries(cart)
        .filter(([, qty]) => qty > 0)
        .map(([product_id, qty]) => ({ product_id, qty }));
      if (!lines.length) throw new Error('Add at least one product to the sale.');
      if (paymentMethod === 'Credit' && !customerId)
        throw new Error('Select a customer for a credit sale.');
      const customer = customers.find((item) => item.id === customerId);
      const result = await posApi<PosTransaction>('/pos/sale', {
        method: 'POST',
        body: JSON.stringify({
          lines,
          payment_method: paymentMethod,
          customer_id: customerId || undefined,
          customer_name: customer?.name || undefined,
          customer_phone: customer?.phone || undefined,
          discount_pct: Number(discountPct),
          payment_reference: paymentReference
        })
      });
      setCart({});
      setCustomerId('');
      setDiscountPct('0');
      setPaymentReference('');
      await load();
      if (result.data?.id) setReceiptTransaction(result.data);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Sale could not be completed.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageContainer>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}
        <CheckoutWorkspace
          products={products}
          filteredProducts={filteredProducts}
          categories={categories}
          customers={customers}
          cart={cart}
          setCart={setCart}
          search={search}
          setSearch={setSearch}
          scanCode={scanCode}
          setScanCode={setScanCode}
          scanProduct={scanProduct}
          checkout={checkout}
          paymentMethod={paymentMethod}
          setPaymentMethod={setPaymentMethod}
          customerId={customerId}
          setCustomerId={setCustomerId}
          discountPct={discountPct}
          setDiscountPct={setDiscountPct}
          paymentReference={paymentReference}
          setPaymentReference={setPaymentReference}
          subtotal={subtotal}
          discountAmount={discountAmount}
          checkoutTax={checkoutTax}
          busy={busy}
          catalogView={catalogView}
          setCatalogView={setCatalogView}
          mobileCartOpen={mobileCartOpen}
          setMobileCartOpen={setMobileCartOpen}
        />
        {receiptTransaction && (
          <ReceiptDialog
            transaction={receiptTransaction}
            onClose={() => setReceiptTransaction(null)}
          />
        )}
      </div>
    </PageContainer>
  );
}
