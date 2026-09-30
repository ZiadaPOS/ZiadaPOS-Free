'use client';

import PageContainer from '@/components/layout/page-container';
import {
  posApi,
  unwrapList,
  type PosCategory,
  type PosCustomer,
  type PosMedicineBatch,
  type PosMedicineProfile,
  type PosPrescriptionDetails,
  type PosProduct,
  type PosTransaction
} from '@/lib/pos-api';
import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useText } from '@/lib/i18n';
import { CheckoutWorkspace } from '@/features/checkout/components/checkout-workspace';
import { ReceiptDialog } from '@/features/sales/components/receipt-dialog';

export default function CheckoutPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [isPharmacy, setIsPharmacy] = useState(false);
  const [businessType, setBusinessType] = useState('retail');
  const [priceMode, setPriceMode] = useState<'retail' | 'wholesale'>('retail');
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [customers, setCustomers] = useState<PosCustomer[]>([]);
  const [search, setSearch] = useState('');
  const [cart, setCart] = useState<Record<string, number>>({});
  const [rxDetails, setRxDetails] = useState<Record<string, PosPrescriptionDetails>>({});
  const [paymentMethod, setPaymentMethod] = useState('Cash');
  const [customerId, setCustomerId] = useState('');
  const [discountPct, setDiscountPct] = useState('0');
  const [paymentReference, setPaymentReference] = useState('');
  const [catalogView, setCatalogView] = useState<'grid' | 'list'>('grid');
  const [mobileCartOpen, setMobileCartOpen] = useState(false);
  const [scanCode, setScanCode] = useState('');
  const [receiptTransaction, setReceiptTransaction] = useState<PosTransaction | null>(null);
  const cartRestored = useRef(false);
  useEffect(() => {
    setRxDetails((current) => {
      const remaining = Object.fromEntries(
        Object.entries(current).filter(([productId]) => (cart[productId] || 0) > 0)
      );
      return Object.keys(remaining).length === Object.keys(current).length ? current : remaining;
    });
  }, [cart]);
  const load = useCallback(async () => {
    setError('');
    try {
      const [p, c, categoryResult, profile] = await Promise.all([
        posApi<unknown>('/inventory/products?status=active&page=1&page_size=100'),
        posApi<unknown>('/customers?page=1&page_size=100'),
        posApi<unknown>('/inventory/categories'),
        posApi<{ organisation?: { business_type?: string } }>('/accounts/me')
      ]);
      const type = profile.data.organisation?.business_type || 'retail';
      setBusinessType(type);
      setPriceMode(type === 'wholesale' ? 'wholesale' : 'retail');
      const pharmacy = type === 'pharmacy';
      setIsPharmacy(pharmacy);
      let productList = unwrapList<PosProduct>(p.data);
      if (pharmacy) {
        const [batchResult, profileResult] = await Promise.all([
          posApi<unknown>('/pharmacy/batches'),
          posApi<unknown>('/pharmacy/products')
        ]);
        const profiles = new Map(
          unwrapList<PosMedicineProfile>(profileResult.data).map((profile) => [
            profile.product_id,
            profile
          ])
        );
        const today = new Date();
        const batches = unwrapList<PosMedicineBatch>(batchResult.data).filter((batch) => {
          const expiry = new Date(`${batch.expiry_date.slice(0, 10)}T00:00:00`);
          return (
            batch.quantity > 0 &&
            expiry >= new Date(today.getFullYear(), today.getMonth(), today.getDate())
          );
        });
        const stockByProduct = new Map<string, { quantity: number; earliest: string }>();
        for (const batch of batches) {
          const current = stockByProduct.get(batch.product_id);
          stockByProduct.set(batch.product_id, {
            quantity: (current?.quantity || 0) + batch.quantity,
            earliest:
              !current || batch.expiry_date < current.earliest
                ? batch.expiry_date
                : current.earliest
          });
        }
        productList = productList.map((product) => {
          const stock = stockByProduct.get(product.id);
          const profile = profiles.get(product.id);
          return {
            ...product,
            stock: stock?.quantity || 0,
            next_expiry_date: stock?.earliest,
            medicine_profile_configured: Boolean(profile),
            ...profile,
            requires_prescription: Boolean(
              profile?.requires_prescription || profile?.medicine_schedule === 'prescription_only'
            )
          };
        });
      }
      setProducts(productList);
      setCustomers(unwrapList<PosCustomer>(c.data));
      setCategories(unwrapList<PosCategory>(categoryResult.data));
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : text('Could not load data.', 'Imeshindikana kupakia taarifa.')
      );
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    try {
      const saved = window.localStorage.getItem('ziada-checkout-cart');
      if (saved) {
        const parsed = JSON.parse(saved) as Record<string, unknown>;
        setCart(
          Object.fromEntries(
            Object.entries(parsed).filter(
              ([, quantity]) => typeof quantity === 'number' && quantity > 0
            )
          ) as Record<string, number>
        );
      }
    } catch {
      window.localStorage.removeItem('ziada-checkout-cart');
    } finally {
      cartRestored.current = true;
    }
  }, []);
  useEffect(() => {
    if (!cartRestored.current) return;
    if (Object.keys(cart).length) {
      window.localStorage.setItem('ziada-checkout-cart', JSON.stringify(cart));
    } else {
      window.localStorage.removeItem('ziada-checkout-cart');
    }
  }, [cart]);
  useEffect(() => {
    if (!products.length) return;
    const available = new Set(products.map((product) => product.id));
    setCart((current) => {
      const next = Object.fromEntries(
        Object.entries(current).filter(([productId]) => available.has(productId))
      ) as Record<string, number>;
      return Object.keys(next).length === Object.keys(current).length ? current : next;
    });
  }, [products]);
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
  const unitPrice = (product: PosProduct) =>
    priceMode === 'wholesale' && product.wholesale_price
      ? product.wholesale_price
      : product.selling_price;
  const subtotal = products.reduce((sum, p) => sum + (cart[p.id] || 0) * unitPrice(p), 0);
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
      setError(
        reason instanceof Error
          ? reason.message
          : text('Product barcode was not found.', 'Barcode ya bidhaa haikupatikana.')
      );
    }
  }

  async function checkout(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    try {
      const lines = Object.entries(cart)
        .filter(([, qty]) => qty > 0)
        .map(([product_id, qty]) => ({ product_id, qty, ...(rxDetails[product_id] || {}) }));
      if (!lines.length)
        throw new Error(
          text('Add at least one product to the sale.', 'Ongeza bidhaa angalau moja kwenye mauzo.')
        );
      if (paymentMethod === 'Credit' && !customerId)
        throw new Error(
          text('Select a customer for a credit sale.', 'Chagua mteja kwa mauzo ya mkopo.')
        );
      const customer = customers.find((item) => item.id === customerId);
      const result = await posApi<PosTransaction>('/pos/sale', {
        method: 'POST',
        body: JSON.stringify({
          lines,
          payment_method: paymentMethod,
          price_mode:
            businessType === 'wholesale' || businessType === 'retail_wholesale'
              ? priceMode
              : 'retail',
          customer_id: customerId || undefined,
          customer_name: customer?.name || undefined,
          customer_phone: customer?.phone || undefined,
          discount_pct: Number(discountPct),
          payment_reference: paymentReference
        })
      });
      setCart({});
      setRxDetails({});
      window.localStorage.removeItem('ziada-checkout-cart');
      window.localStorage.removeItem('ziada-checkout-step');
      setCustomerId('');
      setDiscountPct('0');
      setPaymentReference('');
      await load();
      if (result.data?.id) setReceiptTransaction(result.data);
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : text('Sale could not be completed.', 'Mauzo hayakuweza kukamilika.')
      );
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
          isPharmacy={isPharmacy}
          priceMode={priceMode}
          setPriceMode={setPriceMode}
          canChooseWholesale={businessType === 'retail_wholesale'}
          unitPrice={unitPrice}
          rxDetails={rxDetails}
          setRxDetails={setRxDetails}
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
