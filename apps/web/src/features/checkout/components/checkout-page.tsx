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
import {
  FormEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type SetStateAction
} from 'react';
import { useText } from '@/lib/i18n';
import { CheckoutWorkspace } from '@/features/checkout/components/checkout-workspace';
import { ReceiptDialog } from '@/features/sales/components/receipt-dialog';

async function getSaleIdempotencyKey(
  requestRef: { current: { signature: string; idempotencyKey: string } | null },
  payload: object
): Promise<string> {
  const bytes = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(JSON.stringify(payload))
  );
  const signature = Array.from(new Uint8Array(bytes), (byte) =>
    byte.toString(16).padStart(2, '0')
  ).join('');
  if (!requestRef.current || requestRef.current.signature !== signature) {
    let saved: { signature?: string; idempotencyKey?: string } | null = null;
    try {
      saved = JSON.parse(localStorage.getItem('ziadapos_pending_sale_request_v1') || 'null');
    } catch {
      saved = null;
    }
    const idempotencyKey =
      saved?.signature === signature && saved.idempotencyKey
        ? saved.idempotencyKey
        : crypto.randomUUID();
    requestRef.current = { signature, idempotencyKey };
    try {
      localStorage.setItem('ziadapos_pending_sale_request_v1', JSON.stringify(requestRef.current));
    } catch {
      // The in-memory key still protects retries while this app session is open.
    }
  }
  return requestRef.current.idempotencyKey;
}

export default function CheckoutPage() {
  const text = useText();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [activeUserId, setActiveUserId] = useState('');
  const [activeStoreId, setActiveStoreId] = useState('');
  const [localMode, setLocalMode] = useState(false);
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
  const [checkoutViewResetKey, setCheckoutViewResetKey] = useState(0);
  const [scanCode, setScanCode] = useState('');
  const [receiptTransaction, setReceiptTransaction] = useState<PosTransaction | null>(null);
  const [cartRestored, setCartRestored] = useState(false);
  const [restoredCartStorageKey, setRestoredCartStorageKey] = useState('');
  const [productsLoaded, setProductsLoaded] = useState(false);
  const saleRequestRef = useRef<{ signature: string; idempotencyKey: string } | null>(null);
  const legacyCartStorageKey =
    activeUserId && activeStoreId ? `ziada-checkout-cart:${activeUserId}:${activeStoreId}` : '';
  const previousCartStorageKey =
    activeUserId && activeStoreId ? `ziada-checkout-cart:v3:${activeUserId}:${activeStoreId}` : '';
  const olderCartStorageKey =
    activeUserId && activeStoreId ? `ziada-checkout-cart:v2:${activeUserId}:${activeStoreId}` : '';
  const cartStorageKey =
    activeUserId && activeStoreId ? `ziada-checkout-cart:v4:${activeUserId}:${activeStoreId}` : '';
  const updateCart = useCallback((action: SetStateAction<Record<string, number>>) => {
    setError('');
    setCart(action);
  }, []);
  useEffect(() => {
    setRxDetails((current) => {
      const remaining = Object.fromEntries(
        Object.entries(current).filter(([productId]) => (cart[productId] || 0) > 0)
      );
      return Object.keys(remaining).length === Object.keys(current).length ? current : remaining;
    });
  }, [cart]);
  const load = useCallback(
    async ({ clearError = true }: { clearError?: boolean } = {}) => {
      if (clearError) setError('');
      setProductsLoaded(false);
      try {
        const [p, c, categoryResult, profile] = await Promise.all([
          posApi<unknown>('/inventory/products?status=active&page=1&page_size=100'),
          posApi<unknown>('/customers?page=1&page_size=100'),
          posApi<unknown>('/inventory/categories'),
          posApi<{
            offline_mode?: boolean;
            organisation?: { business_type?: string };
            store?: { id?: string };
            user?: { id?: string };
          }>('/accounts/me')
        ]);
        setLocalMode(Boolean(profile.data.offline_mode));
        setActiveUserId(profile.data.user?.id || '');
        setActiveStoreId(profile.data.store?.id || '');
        const type = profile.data.organisation?.business_type || 'retail';
        setBusinessType(type);
        setPriceMode(type === 'wholesale' ? 'wholesale' : 'retail');
        const pharmacy = type === 'pharmacy';
        setIsPharmacy(pharmacy);
        let productList = unwrapList<PosProduct>(p.data);
        const profileStoreId = profile.data.store?.id || '';
        const catalogStoreId = productList.find((product) => product.store_id)?.store_id || '';
        // The authenticated session is the source of truth for the active
        // store. Do not silently replace its identity with the catalog's.
        if (profileStoreId && catalogStoreId && catalogStoreId !== profileStoreId) {
          throw new Error('ACTIVE_STORE_CHANGED');
        }
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
              // Pharmacy profiles have their own database ID; sale lines must
              // keep the inventory product ID from `product` above.
              generic_name: profile?.generic_name,
              brand_name: profile?.brand_name,
              medicine_schedule: profile?.medicine_schedule,
              requires_prescription: Boolean(
                profile?.requires_prescription || profile?.medicine_schedule === 'prescription_only'
              )
            };
          });
        }
        setProducts(productList);
        setCustomers(unwrapList<PosCustomer>(c.data));
        setCategories(unwrapList<PosCategory>(categoryResult.data));
        setProductsLoaded(true);
      } catch (reason) {
        if (clearError) {
          setError(
            reason instanceof Error && reason.message === 'ACTIVE_STORE_CHANGED'
              ? text(
                  'The store changed while Checkout was loading. Refresh Checkout, then review the products for the active store.',
                  'Duka limebadilika wakati mauzo yanapakia. Pakia upya ukurasa wa mauzo, kisha kagua bidhaa za duka linalotumika.'
                )
              : reason instanceof Error
                ? reason.message
                : text('Could not load data.', 'Imeshindikana kupakia taarifa.')
          );
        }
      }
    },
    [text]
  );
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    // Previous carts can contain product IDs from another store or catalog.
    // Discard saved carts that may contain pharmacy profile IDs instead of
    // inventory product IDs.
    window.localStorage.removeItem('ziada-checkout-cart');
    if (legacyCartStorageKey) window.localStorage.removeItem(legacyCartStorageKey);
    if (previousCartStorageKey) window.localStorage.removeItem(previousCartStorageKey);
    if (olderCartStorageKey) window.localStorage.removeItem(olderCartStorageKey);
  }, [legacyCartStorageKey, olderCartStorageKey, previousCartStorageKey]);
  useEffect(() => {
    // Restore only after both the user/store identity and that store's fresh
    // catalog are known. Otherwise a saved ID from an old/deleted product can
    // briefly become the active cart and block the sale preflight.
    if (!cartStorageKey || !productsLoaded || restoredCartStorageKey === cartStorageKey) return;

    const productById = new Map(products.map((product) => [product.id, product]));
    let removedUnavailableItem = false;
    let restoredCart: Record<string, number> = {};
    try {
      const saved = window.localStorage.getItem(cartStorageKey);
      if (saved) {
        const parsed: unknown = JSON.parse(saved);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          for (const [productId, quantity] of Object.entries(parsed)) {
            const product = productById.get(productId);
            if (!product || !Number.isInteger(quantity) || (quantity as number) <= 0) {
              removedUnavailableItem = true;
              continue;
            }

            const safeQuantity = product.is_service
              ? (quantity as number)
              : Math.min(quantity as number, product.stock);
            if (safeQuantity > 0) restoredCart[productId] = safeQuantity;
            if (safeQuantity !== quantity) removedUnavailableItem = true;
          }
        } else {
          removedUnavailableItem = true;
        }
      }
    } catch {
      removedUnavailableItem = true;
    }

    setCart(restoredCart);
    setRestoredCartStorageKey(cartStorageKey);
    setCartRestored(true);
    if (removedUnavailableItem) {
      setError(
        text(
          'The saved cart was updated to match the products and stock currently available at this store. Review it before continuing.',
          'Kikapu kilichohifadhiwa kimerekebishwa kulingana na bidhaa na stoo inayopatikana sasa katika duka hili. Kihakiki kabla ya kuendelea.'
        )
      );
    }
  }, [cartStorageKey, products, productsLoaded, restoredCartStorageKey, text]);
  useEffect(() => {
    if (
      !cartRestored ||
      !productsLoaded ||
      !cartStorageKey ||
      restoredCartStorageKey !== cartStorageKey
    )
      return;
    if (Object.keys(cart).length) {
      window.localStorage.setItem(cartStorageKey, JSON.stringify(cart));
    } else {
      window.localStorage.removeItem(cartStorageKey);
    }
  }, [cart, cartRestored, cartStorageKey, productsLoaded, restoredCartStorageKey]);
  useEffect(() => {
    if (
      !cartRestored ||
      !productsLoaded ||
      !cartStorageKey ||
      restoredCartStorageKey !== cartStorageKey
    )
      return;

    // Catalogs can change while a checkout tab is open (for example, when a
    // product is archived in another tab). Reconcile the in-memory cart after
    // every successful refresh instead of checking only on the initial restore.
    const productById = new Map(products.map((product) => [product.id, product]));
    let changed = false;
    const currentCart: Record<string, number> = {};
    for (const [productId, quantity] of Object.entries(cart)) {
      const product = productById.get(productId);
      if (!product || !Number.isInteger(quantity) || quantity <= 0) {
        changed = true;
        continue;
      }
      const safeQuantity = product.is_service ? quantity : Math.min(quantity, product.stock);
      if (safeQuantity > 0) currentCart[productId] = safeQuantity;
      if (safeQuantity !== quantity) changed = true;
    }

    if (changed) {
      setCart(currentCart);
      setMobileCartOpen(false);
      setCheckoutViewResetKey((key) => key + 1);
      setError(
        text(
          'The cart was refreshed to match products and stock available at this store. Review it before continuing.',
          'Kikapu kimepangwa upya kulingana na bidhaa na stoo inayopatikana katika duka hili. Kihakiki kabla ya kuendelea.'
        )
      );
    }
  }, [cart, cartRestored, cartStorageKey, products, productsLoaded, restoredCartStorageKey, text]);
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
    if (!productsLoaded) {
      setError(
        text(
          'Products are still loading. Try again shortly.',
          'Bidhaa bado zinapakia. Jaribu tena baada ya muda mfupi.'
        )
      );
      return;
    }
    if (!Object.values(cart).some((quantity) => quantity > 0)) {
      setError(
        text('Add at least one product to the sale.', 'Ongeza bidhaa angalau moja kwenye mauzo.')
      );
      return;
    }
    setBusy(true);
    setError('');
    try {
      // A checkout tab may stay open after products are deleted or replaced.
      // Validate against an uncached catalog so the UI cannot submit an old ID
      // just because it is still present in this tab's in-memory product list.
      const cartProductIds = Object.keys(cart).filter((productId) => cart[productId] > 0);
      const [catalog, session] = await Promise.all([
        posApi<unknown>(
          `/inventory/products?status=active&fresh=true&ids=${encodeURIComponent(cartProductIds.join(','))}`
        ),
        posApi<{ store?: { id?: string } }>('/accounts/me')
      ]);
      const currentStoreId = session.data.store?.id || '';
      if (!activeStoreId || !currentStoreId) {
        throw new Error(
          text(
            'Could not confirm the active store. Reload checkout and try again.',
            'Imeshindikana kuthibitisha duka linalotumika. Pakia upya ukurasa wa mauzo kisha ujaribu tena.'
          )
        );
      }
      if (currentStoreId !== activeStoreId) {
        throw new Error('ACTIVE_STORE_CHANGED');
      }
      const latestProducts = unwrapList<PosProduct>(catalog.data);
      const available = new Set(latestProducts.map((product) => product.id));
      const staleProductIds = Object.keys(cart).filter((productId) => !available.has(productId));
      if (staleProductIds.length) {
        const currentCart = Object.fromEntries(
          Object.entries(cart).filter(([productId]) => available.has(productId))
        );
        setCart(currentCart);
        setMobileCartOpen(false);
        setCheckoutViewResetKey((key) => key + 1);
        void load({ clearError: false });
        setError(
          text(
            'A product in this cart is no longer available in the active store. It was removed; add the current product and try again.',
            'Bidhaa moja kwenye kikapu haipatikani tena kwenye duka linalotumika. Imeondolewa; ongeza bidhaa iliyopo kisha ujaribu tena.'
          )
        );
        return;
      }
      const lines = Object.entries(cart)
        .filter(([, qty]) => qty > 0)
        .map(([product_id, qty]) => ({ product_id, qty, ...rxDetails[product_id] }));
      if (paymentMethod === 'Credit' && !customerId)
        throw new Error(
          text('Select a customer for a credit sale.', 'Chagua mteja kwa mauzo ya mkopo.')
        );
      const customer = customers.find((item) => item.id === customerId);
      const salePayload = {
        expected_store_id: activeStoreId,
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
      };
      const idempotencyKey = await getSaleIdempotencyKey(saleRequestRef, salePayload);
      const result = await posApi<PosTransaction>('/pos/sale', {
        method: 'POST',
        body: JSON.stringify({ ...salePayload, idempotency_key: idempotencyKey })
      });
      saleRequestRef.current = null;
      try {
        localStorage.removeItem('ziadapos_pending_sale_request_v1');
      } catch {
        // A local receipt is authoritative even if browser storage is unavailable.
      }
      setCart({});
      setRxDetails({});
      setMobileCartOpen(false);
      setCheckoutViewResetKey((key) => key + 1);
      if (cartStorageKey) window.localStorage.removeItem(cartStorageKey);
      setCustomerId('');
      setDiscountPct('0');
      setPaymentReference('');
      await load();
      if (result.data?.id) setReceiptTransaction(result.data);
    } catch (reason) {
      if (
        reason instanceof Error &&
        (reason.message === 'ACTIVE_STORE_CHANGED' ||
          reason.message === 'Active store changed; refresh checkout before completing the sale')
      ) {
        await load({ clearError: false });
        setError(
          text(
            'The active store changed in another tab. Checkout has refreshed; review the updated cart before completing the sale.',
            'Duka linalotumika limebadilishwa kwenye kichupo kingine. Mauzo yamepakiwa upya; kagua kikapu kabla ya kukamilisha.'
          )
        );
      } else if (
        reason instanceof Error &&
        reason.message === 'One or more products were not found in this store'
      ) {
        await load({ clearError: false });
        setError(
          text(
            'The store or product list changed during checkout. Unavailable items were removed from the cart; review it and complete the sale again.',
            'Duka au orodha ya bidhaa imebadilika wakati wa mauzo. Bidhaa zisizopatikana zimeondolewa kwenye kikapu; kagua kisha kamilisha mauzo tena.'
          )
        );
      } else {
        setError(
          reason instanceof Error
            ? reason.message
            : text('Sale could not be completed.', 'Mauzo hayakuweza kukamilika.')
        );
      }
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
        {localMode && (
          <div className='rounded-md border border-amber-500/40 bg-amber-500/5 p-3 text-sm'>
            <strong>Local-only POS.</strong> Cash sales and products are saved on this computer.
            They are not synced to an online account.
          </div>
        )}
        <CheckoutWorkspace
          products={products}
          localMode={localMode}
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
          setCart={updateCart}
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
          checkoutViewResetKey={checkoutViewResetKey}
        />
        {receiptTransaction && (
          <ReceiptDialog
            transaction={receiptTransaction}
            localMode={localMode}
            onClose={() => setReceiptTransaction(null)}
          />
        )}
      </div>
    </PageContainer>
  );
}
