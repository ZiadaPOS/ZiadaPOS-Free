'use client';

import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  posApi,
  money,
  unwrapList,
  type PosCategory,
  type PosCustomer,
  type PosCreditTab,
  type PosProduct,
  type PosTransaction
} from '@/lib/pos-api';
import { ThemeModeToggle } from '@/components/themes/theme-mode-toggle';
import { ThemeSelector } from '@/components/themes/theme-selector';
import { Icons } from '@/components/icons';
import { usePathname, useRouter } from 'next/navigation';
import {
  Dispatch,
  FormEvent,
  SetStateAction,
  useCallback,
  useEffect,
  useMemo,
  useState
} from 'react';

const today = () => new Date().toLocaleDateString('en-CA');

export default function PosWorkspace() {
  const pathname = usePathname();
  const router = useRouter();
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [products, setProducts] = useState<PosProduct[]>([]);
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [customers, setCustomers] = useState<PosCustomer[]>([]);
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const [creditTabs, setCreditTabs] = useState<PosCreditTab[]>([]);
  const [summary, setSummary] = useState<Record<string, number>>({});
  const [search, setSearch] = useState('');
  const [cart, setCart] = useState<Record<string, number>>({});
  const [paymentMethod, setPaymentMethod] = useState('Cash');
  const [customerId, setCustomerId] = useState('');
  const [discountPct, setDiscountPct] = useState('0');
  const [paymentReference, setPaymentReference] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [catalogView, setCatalogView] = useState<'grid' | 'list'>('grid');
  const [mobileCartOpen, setMobileCartOpen] = useState(false);
  const [scanCode, setScanCode] = useState('');
  const [receiptTransaction, setReceiptTransaction] = useState<PosTransaction | null>(null);
  const [settingsData, setSettingsData] = useState<Record<string, unknown> | null>(null);
  const [user, setUser] = useState<{ full_name?: string; role?: string } | null>(null);

  const load = useCallback(async () => {
    setError('');
    try {
      if (pathname.includes('/checkout')) {
        const [p, c, categoryResult] = await Promise.all([
          posApi<unknown>('/inventory/products?status=active&page=1&page_size=100'),
          posApi<unknown>('/customers?page=1&page_size=100'),
          posApi<unknown>('/inventory/categories')
        ]);
        setProducts(unwrapList<PosProduct>(p.data));
        setCustomers(unwrapList<PosCustomer>(c.data));
        setCategories(unwrapList<PosCategory>(categoryResult.data));
      } else if (pathname.includes('/sales')) {
        const result = await posApi<unknown>('/pos/transactions?page=1&page_size=50');
        setTransactions(unwrapList<PosTransaction>(result.data));
      } else if (pathname.includes('/overview')) {
        const [s, t, p] = await Promise.allSettled([
          posApi<Record<string, number>>(`/pos/summary?date_from=${today()}&date_to=${today()}`),
          posApi<unknown>('/pos/transactions?page=1&page_size=6'),
          posApi<unknown>('/inventory/low-stock')
        ]);
        if (s.status === 'fulfilled') setSummary(s.value.data);
        if (t.status === 'fulfilled') setTransactions(unwrapList<PosTransaction>(t.value.data));
        if (p.status === 'fulfilled') setProducts(unwrapList<PosProduct>(p.value.data));
      } else if (pathname.includes('/customers')) {
        const result = await posApi<unknown>('/customers?page=1&page_size=100');
        setCustomers(unwrapList<PosCustomer>(result.data));
      } else if (pathname.includes('/inventory')) {
        const result = await posApi<unknown>('/inventory/products?page=1&page_size=100');
        setProducts(unwrapList<PosProduct>(result.data));
      } else if (pathname.includes('/credits')) {
        const [credit, customerResult] = await Promise.all([
          posApi<unknown>('/credits?page=1&page_size=100'),
          posApi<unknown>('/customers?page=1&page_size=100')
        ]);
        setCreditTabs(unwrapList<PosCreditTab>(credit.data));
        setCustomers(unwrapList<PosCustomer>(customerResult.data));
      } else if (pathname.includes('/settings')) {
        const me = await posApi<unknown>('/accounts/me');
        setSettingsData(me.data as Record<string, unknown>);
      }
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load data.');
    }
  }, [pathname]);

  useEffect(() => {
    try {
      const saved = localStorage.getItem('pos_user');
      if (saved) setUser(JSON.parse(saved));
    } catch {
      /* ignore invalid old local profile */
    }
    void fetch('/api/pos-session')
      .then((response) => {
        if (!response.ok) router.replace('/auth/sign-in');
      })
      .catch(() => undefined);
    void load();
  }, [load, router]);

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

  const pageMeta = pathname.includes('/overview')
    ? { title: 'Business overview', description: 'Today’s performance and store activity.' }
    : pathname.includes('/checkout')
      ? {
          title: 'Checkout',
          description: 'Scan or search products, select payment, and complete the sale.'
        }
      : pathname.includes('/sales')
        ? { title: 'Sales', description: 'Transaction ledger, receipts, refunds, and voids.' }
        : pathname.includes('/inventory')
          ? { title: 'Inventory', description: 'Products and current stock for this store.' }
          : pathname.includes('/customers')
            ? {
                title: 'Customers',
                description: 'Customer profiles, segments, spend, and outstanding credit.'
              }
            : pathname.includes('/credits')
              ? {
                  title: 'Credit accounts',
                  description: 'Outstanding customer tabs and payment follow-up.'
                }
              : {
                  title:
                    pathname
                      .split('/')
                      .filter(Boolean)
                      .at(-1)
                      ?.replace(/^./, (letter) => letter.toUpperCase()) || 'Workspace',
                  description: 'Manage this area for the active store.'
                };

  const usesOwnHeader = [
    '/checkout',
    '/inventory',
    '/customers',
    '/suppliers',
    '/expenses',
    '/staff',
    '/stores',
    '/reports'
  ].some((route) => pathname.includes(route));

  return (
    <PageContainer
      pageTitle={usesOwnHeader ? undefined : pageMeta.title}
      pageDescription={usesOwnHeader ? undefined : pageMeta.description}
    >
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-5'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}
        {pathname.includes('/checkout') && (
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
        )}

        {pathname.includes('/overview') && (
          <>
            <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-4'>
              {[
                [
                  'Sales today',
                  money(summary.gross_sales || summary.net_sales || 0),
                  'Gross sales recorded today'
                ],
                ['Transactions', String(summary.transactions || 0), 'Completed receipts'],
                ['Profit', money(summary.profit || 0), 'Based on recorded product costs'],
                ['Low stock', String(products.length), 'Products to review']
              ].map(([label, value, detail]) => (
                <Card key={label}>
                  <CardHeader className='pb-2'>
                    <CardDescription>{label}</CardDescription>
                    <CardTitle className='text-2xl'>{value}</CardTitle>
                  </CardHeader>
                  <CardContent className='text-muted-foreground text-xs'>{detail}</CardContent>
                </Card>
              ))}
            </div>
            <Card>
              <CardHeader>
                <CardTitle>Recent sales</CardTitle>
                <CardDescription>Latest transactions for the active store.</CardDescription>
              </CardHeader>
              <CardContent>
                <TransactionRows transactions={transactions} />
              </CardContent>
            </Card>
          </>
        )}

        {false && pathname.includes('/checkout') && (
          <>
            <div>
              <h1 className='text-2xl font-bold tracking-tight'>Checkout</h1>
              <p className='text-muted-foreground'>
                Scan or search products, select payment, and complete the sale.
              </p>
            </div>
            <div className='grid min-h-0 flex-1 gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,420px)]'>
              <Card className='min-w-0'>
                <CardHeader>
                  <CardTitle>Products</CardTitle>
                  <div className='pos-search-row grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]'>
                    <Input
                      aria-label='Search products'
                      placeholder='Search name, SKU, or barcode…'
                      value={search}
                      onChange={(event) => setSearch(event.target.value)}
                    />
                    <form onSubmit={scanProduct} className='flex gap-2'>
                      <Input
                        aria-label='Scan barcode'
                        placeholder='Scan barcode…'
                        value={scanCode}
                        onChange={(event) => setScanCode(event.target.value)}
                      />
                      <Button type='submit' variant='outline'>
                        Find
                      </Button>
                    </form>
                  </div>
                </CardHeader>
                <CardContent className='grid max-h-[calc(100svh-18rem)] gap-3 overflow-y-auto sm:grid-cols-2 2xl:grid-cols-3'>
                  {filteredProducts.length ? (
                    filteredProducts.map((p) => (
                      <button
                        key={p.id}
                        disabled={!p.is_service && p.stock <= 0}
                        onClick={() =>
                          setCart((current) => ({
                            ...current,
                            [p.id]: Math.min(current[p.id] || 0, p.stock - 1) + 1
                          }))
                        }
                        className='hover:bg-muted/50 focus-visible:ring-ring rounded-lg border p-3 text-left transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50'
                      >
                        <p className='font-medium'>{p.name}</p>
                        <p className='text-muted-foreground mt-1 text-xs'>
                          {p.sku || p.barcode || 'No SKU'} · stock{' '}
                          {p.is_service ? 'service' : p.stock}
                        </p>
                        <p className='mt-3 font-semibold'>{money(p.selling_price)}</p>
                      </button>
                    ))
                  ) : (
                    <div className='text-muted-foreground flex min-h-40 items-center justify-center text-center text-sm sm:col-span-2 2xl:col-span-3'>
                      {filteredProducts.length
                        ? 'No products in this category.'
                        : products.length
                          ? 'No products match your search.'
                          : 'No products are available at this store yet.'}
                    </div>
                  )}
                </CardContent>
              </Card>
              <Card className='xl:sticky xl:top-20 xl:self-start'>
                <CardHeader>
                  <CardTitle>Current sale</CardTitle>
                  <CardDescription>
                    {Object.values(cart).reduce((a, b) => a + b, 0)} items
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <form onSubmit={checkout} className='space-y-4'>
                    <div className='max-h-44 space-y-3 overflow-y-auto'>
                      {products
                        .filter((p) => cart[p.id])
                        .map((p) => (
                          <div
                            key={p.id}
                            className='flex items-center justify-between gap-2 text-sm'
                          >
                            <span className='min-w-0 truncate'>
                              {cart[p.id]} × {p.name}
                            </span>
                            <span className='shrink-0'>{money(cart[p.id] * p.selling_price)}</span>
                            <div className='flex shrink-0 gap-1'>
                              <Button
                                type='button'
                                size='sm'
                                variant='outline'
                                aria-label={`Remove one ${p.name}`}
                                onClick={() =>
                                  setCart((v) => ({
                                    ...v,
                                    [p.id]: Math.max(0, (v[p.id] || 0) - 1)
                                  }))
                                }
                              >
                                −
                              </Button>
                              <Button
                                type='button'
                                size='sm'
                                variant='ghost'
                                aria-label={`Remove ${p.name} from sale`}
                                onClick={() =>
                                  setCart((v) => {
                                    const next = { ...v };
                                    delete next[p.id];
                                    return next;
                                  })
                                }
                              >
                                ×
                              </Button>
                            </div>
                          </div>
                        ))}
                    </div>
                    <div>
                      <label className='mb-1 block text-sm font-medium' htmlFor='customer'>
                        Customer
                      </label>
                      <select
                        id='customer'
                        className='bg-background h-9 w-full rounded-md border px-3 text-sm'
                        value={customerId}
                        onChange={(event) => setCustomerId(event.target.value)}
                      >
                        <option value=''>Walk-in customer</option>
                        {customers.map((c) => (
                          <option key={c.id} value={c.id}>
                            {c.name}
                            {c.phone ? ` · ${c.phone}` : ''}
                          </option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className='mb-1 block text-sm font-medium' htmlFor='payment'>
                        Payment method
                      </label>
                      <select
                        id='payment'
                        className='bg-background h-9 w-full rounded-md border px-3 text-sm'
                        value={paymentMethod}
                        onChange={(event) => setPaymentMethod(event.target.value)}
                      >
                        {['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit'].map((method) => (
                          <option key={method}>{method}</option>
                        ))}
                      </select>
                    </div>
                    <div className='grid grid-cols-2 gap-2'>
                      <label className='text-sm'>
                        Discount %
                        <Input
                          min='0'
                          max='100'
                          type='number'
                          value={discountPct}
                          onChange={(e) => setDiscountPct(e.target.value)}
                        />
                      </label>
                      <label className='text-sm'>
                        Reference
                        <Input
                          value={paymentReference}
                          onChange={(e) => setPaymentReference(e.target.value)}
                          placeholder='Optional'
                        />
                      </label>
                    </div>
                    <div className='border-t pt-3 text-sm'>
                      <div className='flex justify-between'>
                        <span>Subtotal</span>
                        <span>{money(subtotal)}</span>
                      </div>
                      <div className='flex justify-between'>
                        <span>Discount</span>
                        <span>{money(discountAmount)}</span>
                      </div>
                      <div className='flex justify-between'>
                        <span>Tax (18%)</span>
                        <span>{money(checkoutTax)}</span>
                      </div>
                      <div className='mt-2 flex justify-between font-semibold'>
                        <span>Total</span>
                        <span>{money(subtotal - discountAmount + checkoutTax)}</span>
                      </div>
                    </div>
                    <Button
                      className='w-full'
                      isLoading={busy}
                      disabled={Object.values(cart).every((quantity) => quantity === 0)}
                    >
                      Complete sale
                    </Button>
                  </form>
                </CardContent>
              </Card>
            </div>
          </>
        )}

        {pathname.includes('/sales') && (
          <SalesTable transactions={transactions} onUpdated={() => void load()} />
        )}
        {pathname.includes('/inventory') && (
          <>
            <div className='flex items-start justify-between'>
              <div>
                <h1 className='text-2xl font-bold tracking-tight'>Inventory</h1>
                <p className='text-muted-foreground'>Products and current stock for this store.</p>
              </div>
              <div className='flex gap-2'>
                <CategoryManager />
                <Button onClick={() => setShowCreate((v) => !v)}>
                  {showCreate ? 'Close' : 'Add product'}
                </Button>
              </div>
            </div>
            {showCreate && (
              <CreateProduct
                onCreated={() => {
                  setShowCreate(false);
                  void load();
                }}
              />
            )}
            <InventoryWorkspace products={products} onUpdated={() => void load()} />
          </>
        )}
        {pathname.includes('/customers') && (
          <>
            <div className='flex items-start justify-between'>
              <div>
                <h1 className='text-2xl font-bold tracking-tight'>Customers</h1>
                <p className='text-muted-foreground'>
                  Customer profiles, segments, spend, and outstanding credit.
                </p>
              </div>
              <Button onClick={() => setShowCreate((v) => !v)}>
                {showCreate ? 'Close' : 'Add customer'}
              </Button>
            </div>
            {showCreate && (
              <CreateCustomer
                onCreated={() => {
                  setShowCreate(false);
                  void load();
                }}
              />
            )}
            <DataTable
              columns={['Customer', 'Phone', 'Segment', 'Lifetime spend', 'Open credit']}
              rows={customers.map((c) => [
                c.name,
                c.phone || '—',
                c.segment,
                money(c.total_spent),
                money(c.open_credit)
              ])}
            />
          </>
        )}
        {pathname.includes('/credits') && (
          <CreditTable tabs={creditTabs} customers={customers} onUpdated={() => void load()} />
        )}
        {['suppliers', 'expenses', 'staff', 'stores', 'reports'].some((section) =>
          pathname.includes(`/${section}`)
        ) && <ResourcePage pathname={pathname} />}
        {pathname.includes('/suppliers') && <SupplierOperations />}
        {receiptTransaction && (
          <ReceiptDialog
            transaction={receiptTransaction}
            onClose={() => setReceiptTransaction(null)}
          />
        )}
        {pathname.includes('/settings') && (
          <>
            <Card>
              <CardHeader>
                <CardTitle>Appearance</CardTitle>
                <CardDescription>
                  Choose the Ziada or Supabase workspace style and light or dark mode.
                </CardDescription>
              </CardHeader>
              <CardContent className='flex flex-wrap items-center gap-3'>
                <ThemeSelector />
                <ThemeModeToggle />
              </CardContent>
            </Card>
            <StoreSwitcher />
            {(user?.role === 'owner' || user?.role === 'admin') && (
              <BusinessSettings
                initialValue={settingsData?.organisation as Record<string, unknown> | undefined}
              />
            )}
          </>
        )}
      </div>
    </PageContainer>
  );
}

function CheckoutWorkspace({
  products,
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
  setMobileCartOpen
}: {
  products: PosProduct[];
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
}) {
  const [activeCategory, setActiveCategory] = useState('');
  const visibleProducts = activeCategory
    ? filteredProducts.filter(
        (product) =>
          product.category_id === activeCategory || product.category?.id === activeCategory
      )
    : filteredProducts;
  const itemCount = Object.values(cart).reduce((sum, quantity) => sum + quantity, 0);
  const saleForm = (
    <form onSubmit={checkout} className='space-y-4'>
      <div className='max-h-44 space-y-3 overflow-y-auto'>
        {products
          .filter((product) => cart[product.id])
          .map((product) => (
            <div key={product.id} className='flex items-center justify-between gap-2 text-sm'>
              <span className='min-w-0 truncate'>
                {cart[product.id]} × {product.name}
              </span>
              <span>{money(cart[product.id] * product.selling_price)}</span>
              <div className='flex gap-1'>
                <Button
                  type='button'
                  size='sm'
                  variant='outline'
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
                  size='sm'
                  variant='ghost'
                  onClick={() =>
                    setCart((current) => {
                      const next = { ...current };
                      delete next[product.id];
                      return next;
                    })
                  }
                >
                  ×
                </Button>
              </div>
            </div>
          ))}
      </div>
      <label className='block text-sm font-medium'>
        Customer
        <select
          className='bg-background mt-1 h-11 w-full rounded-md border px-3 text-sm'
          value={customerId}
          onChange={(event) => setCustomerId(event.target.value)}
        >
          <option value=''>Walk-in customer</option>
          {customers.map((customer) => (
            <option key={customer.id} value={customer.id}>
              {customer.name}
            </option>
          ))}
        </select>
      </label>
      <label className='block text-sm font-medium'>
        Payment method
        <select
          className='bg-background mt-1 h-11 w-full rounded-md border px-3 text-sm'
          value={paymentMethod}
          onChange={(event) => setPaymentMethod(event.target.value)}
        >
          {['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit'].map((method) => (
            <option key={method}>{method}</option>
          ))}
        </select>
      </label>
      <div className='grid grid-cols-2 gap-2'>
        <label className='text-sm'>
          Discount %
          <Input
            className='mt-1 h-11'
            min='0'
            max='100'
            type='number'
            value={discountPct}
            onChange={(event) => setDiscountPct(event.target.value)}
          />
        </label>
        <label className='text-sm'>
          Reference
          <Input
            className='mt-1 h-11'
            value={paymentReference}
            onChange={(event) => setPaymentReference(event.target.value)}
            placeholder='Optional'
          />
        </label>
      </div>
      <div className='border-t pt-3 text-sm'>
        <div className='flex justify-between'>
          <span>Subtotal</span>
          <span>{money(subtotal)}</span>
        </div>
        <div className='flex justify-between'>
          <span>Discount</span>
          <span>{money(discountAmount)}</span>
        </div>
        <div className='flex justify-between'>
          <span>Tax (18%)</span>
          <span>{money(checkoutTax)}</span>
        </div>
        <div className='mt-2 flex justify-between font-semibold'>
          <span>Total</span>
          <span>{money(subtotal - discountAmount + checkoutTax)}</span>
        </div>
      </div>
      <Button className='h-11 w-full' isLoading={busy} disabled={!itemCount}>
        Complete sale
      </Button>
    </form>
  );
  const controls = (
    <div className='flex shrink-0 gap-1'>
      <Button
        size='icon'
        variant={catalogView === 'grid' ? 'default' : 'outline'}
        aria-label='Grid view'
        onClick={() => setCatalogView('grid')}
      >
        <Icons.dashboard className='h-4 w-4' />
      </Button>
      <Button
        size='icon'
        variant={catalogView === 'list' ? 'default' : 'outline'}
        aria-label='List view'
        onClick={() => setCatalogView('list')}
      >
        <Icons.page className='h-4 w-4' />
      </Button>
      <Button
        className='pos-title-cart relative xl:hidden'
        size='icon'
        variant='outline'
        aria-label={`Open current sale, ${itemCount} items`}
        onClick={() => setMobileCartOpen(true)}
      >
        <Icons.cart className='h-4 w-4' />
        {itemCount > 0 && (
          <span className='bg-primary text-primary-foreground absolute -right-1 -top-1 min-w-4 rounded-full px-1 text-[10px]'>
            {itemCount}
          </span>
        )}
      </Button>
    </div>
  );
  return (
    <>
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>Checkout</h1>
        <p className='text-muted-foreground'>
          Scan or search products, select payment, and complete the sale.
        </p>
      </div>
      <div className='grid min-h-0 flex-1 gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(360px,420px)]'>
        <Card className='min-w-0'>
          <CardHeader>
            <div className='flex items-center justify-between gap-3'>
              <CardTitle>Products</CardTitle>
              {controls}
            </div>
            <div className='pos-search-row grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]'>
              <Input
                aria-label='Search products'
                placeholder='Search name, SKU, or barcode…'
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
              <form onSubmit={scanProduct} className='pos-scan-form flex min-w-0 gap-2'>
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
                className='pos-row-cart relative'
                size='icon'
                variant='outline'
                aria-label={`Open current sale, ${itemCount} items`}
                onClick={() => setMobileCartOpen(true)}
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
          <CardContent className='pos-product-catalog grid max-h-[calc(100svh-18rem)] gap-3 overflow-y-auto'>
            {visibleProducts.length ? (
              visibleProducts.map((product) => (
                <button
                  key={product.id}
                  disabled={!product.is_service && product.stock <= 0}
                  onClick={() =>
                    setCart((current) => ({
                      ...current,
                      [product.id]: Math.min(current[product.id] || 0, product.stock - 1) + 1
                    }))
                  }
                  className='hover:bg-muted/50 rounded-lg border p-3 text-left transition-colors disabled:opacity-50'
                >
                  <div className='pos-product-image'>
                    {product.image ? (
                      <img src={product.image} alt='' className='h-full w-full object-cover' />
                    ) : (
                      <span>{product.name.slice(0, 1).toUpperCase()}</span>
                    )}
                  </div>
                  <p className='font-medium'>{product.name}</p>
                  <p className='text-muted-foreground mt-1 text-xs'>
                    {product.sku || product.barcode || 'No SKU'} · stock{' '}
                    {product.is_service ? 'service' : product.stock}
                  </p>
                  <p className='mt-3 font-semibold'>{money(product.selling_price)}</p>
                </button>
              ))
            ) : (
              <div className='text-muted-foreground flex min-h-40 items-center justify-center text-center text-sm'>
                {products.length
                  ? 'No products match your search.'
                  : 'No products are available at this store yet.'}
              </div>
            )}
          </CardContent>
        </Card>
        <Card className='hidden xl:sticky xl:top-20 xl:block xl:self-start'>
          <CardHeader>
            <CardTitle>Current sale</CardTitle>
            <CardDescription>{itemCount} items</CardDescription>
          </CardHeader>
          <CardContent>{saleForm}</CardContent>
        </Card>
      </div>
      {mobileCartOpen && (
        <div
          className='fixed inset-0 z-50 bg-black/50 xl:hidden'
          onClick={() => setMobileCartOpen(false)}
        >
          <section
            role='dialog'
            aria-modal='true'
            className='bg-background absolute inset-x-0 bottom-0 max-h-[85svh] overflow-y-auto rounded-t-2xl border p-4 shadow-2xl'
            onClick={(event) => event.stopPropagation()}
          >
            <div className='mb-4 flex items-center justify-between'>
              <div>
                <h2 className='text-lg font-semibold'>Current sale</h2>
                <p className='text-muted-foreground text-sm'>{itemCount} items</p>
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

function TransactionRows({ transactions }: { transactions: PosTransaction[] }) {
  return (
    <div className='divide-y'>
      {transactions.length ? (
        transactions.map((t) => (
          <div
            key={t.id}
            className='flex flex-wrap items-center justify-between gap-2 py-3 text-sm'
          >
            <div>
              <p className='font-medium'>
                {t.txn_number} · {t.customer_name || 'Walk-in'}
              </p>
              <p className='text-muted-foreground text-xs'>
                {new Date(t.created_at).toLocaleString()} · {t.payment_method} · {t.status}
              </p>
            </div>
            <p className='font-semibold'>{money(t.total)}</p>
          </div>
        ))
      ) : (
        <p className='text-muted-foreground py-6 text-center text-sm'>No transactions found.</p>
      )}
    </div>
  );
}

function InventoryWorkspace({
  products,
  onUpdated
}: {
  products: PosProduct[];
  onUpdated: () => void;
}) {
  const router = useRouter();
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('all');
  const [status, setStatus] = useState('all');
  const [view, setView] = useState<'table' | 'grid'>('table');
  const categories = Array.from(
    new Map(
      products.map((product) => [
        product.category_id || product.category?.id || 'uncategorized',
        product.category?.name || 'Uncategorized'
      ])
    ).entries()
  );
  const filtered = products.filter((product) => {
    const text = `${product.name} ${product.sku} ${product.barcode}`.toLowerCase();
    const matchesQuery = text.includes(query.toLowerCase());
    const matchesCategory =
      category === 'all' ||
      (product.category_id || product.category?.id || 'uncategorized') === category;
    const matchesStatus =
      status === 'all' ||
      (status === 'active'
        ? product.is_active && !product.low_stock
        : status === 'low'
          ? product.low_stock && product.stock > 0
          : product.stock === 0);
    return matchesQuery && matchesCategory && matchesStatus;
  });
  const totalValue = products.reduce((sum, product) => sum + product.stock * product.cost_price, 0);
  const lowStock = products.filter((product) => product.low_stock && product.stock > 0).length;
  const outOfStock = products.filter((product) => product.stock === 0).length;
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          [
            'Total SKU',
            String(products.length),
            `${products.filter((product) => product.is_active).length} active`
          ],
          ['Inventory value', money(totalValue), 'At cost value'],
          ['Low stock', String(lowStock), lowStock ? 'Needs restock' : 'All above reorder point'],
          ['Out of stock', String(outOfStock), outOfStock ? 'Needs restock now' : 'Nothing out yet']
        ].map(([label, value, detail]) => (
          <Card key={label} className='min-w-0'>
            <CardHeader className='p-3 pb-2 sm:p-6 sm:pb-2'>
              <CardDescription className='truncate text-xs sm:text-sm'>{label}</CardDescription>
              <CardTitle className='truncate text-lg sm:text-2xl'>{value}</CardTitle>
            </CardHeader>
            <CardContent className='truncate p-3 pt-0 text-[11px] text-muted-foreground sm:p-6 sm:pt-0 sm:text-xs'>
              {detail}
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <div>
              <CardTitle>Products</CardTitle>
              <CardDescription>
                Search, filter, and manage every product in this store.
              </CardDescription>
            </div>
            <div className='inventory-view-controls flex gap-1'>
              <Button
                size='icon'
                variant={view === 'table' ? 'default' : 'outline'}
                aria-label='Table view'
                onClick={() => setView('table')}
              >
                <Icons.page className='h-4 w-4' />
              </Button>
              <Button
                size='icon'
                variant={view === 'grid' ? 'default' : 'outline'}
                aria-label='Grid view'
                onClick={() => setView('grid')}
              >
                <Icons.dashboard className='h-4 w-4' />
              </Button>
            </div>
          </div>
          <div className='inventory-filter-row grid gap-2 md:grid-cols-[minmax(0,1fr)_160px_160px]'>
            <Input
              className='h-11'
              aria-label='Search inventory'
              placeholder='Search name, SKU, or barcode…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <select
              aria-label='Filter by category'
              className='bg-background h-11 min-w-0 rounded-md border px-3 text-sm'
              value={category}
              onChange={(event) => setCategory(event.target.value)}
            >
              <option value='all'>All categories</option>
              {categories.map(([id, name]) => (
                <option key={id} value={id}>
                  {name}
                </option>
              ))}
            </select>
            <select
              aria-label='Filter by stock status'
              className='bg-background h-11 min-w-0 rounded-md border px-3 text-sm'
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value='all'>All stock</option>
              <option value='active'>Active</option>
              <option value='low'>Low stock</option>
              <option value='out'>Out of stock</option>
            </select>
            <div className='inventory-mobile-view-controls inventory-view-controls flex gap-1'>
              <Button
                size='icon'
                variant={view === 'table' ? 'default' : 'outline'}
                aria-label='Table view'
                onClick={() => setView('table')}
              >
                <Icons.page className='h-4 w-4' />
              </Button>
              <Button
                size='icon'
                variant={view === 'grid' ? 'default' : 'outline'}
                aria-label='Grid view'
                onClick={() => setView('grid')}
              >
                <Icons.dashboard className='h-4 w-4' />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {view === 'table' ? (
            <InventoryTable products={filtered} onUpdated={onUpdated} />
          ) : (
            <div className='grid grid-cols-2 gap-3 lg:grid-cols-3 xl:grid-cols-4'>
              {filtered.map((product) => (
                <Card
                  key={product.id}
                  className='cursor-pointer overflow-hidden transition-shadow hover:shadow-md'
                  onClick={() => router.push(`/dashboard/inventory/${product.id}`)}
                >
                  <div className='bg-muted/40 flex aspect-[4/3] items-center justify-center text-3xl font-semibold text-primary'>
                    {product.image ? (
                      <img src={product.image} alt='' className='h-full w-full object-cover' />
                    ) : (
                      product.name.slice(0, 1).toUpperCase()
                    )}
                  </div>
                  <CardContent className='space-y-2 p-3'>
                    <p className='truncate font-medium'>{product.name}</p>
                    <p className='text-muted-foreground truncate text-xs'>
                      {product.sku || product.barcode || 'No SKU'}
                    </p>
                    <div className='flex items-center justify-between text-sm'>
                      <span className='font-semibold'>{money(product.selling_price)}</span>
                      <span
                        className={
                          product.stock === 0
                            ? 'text-destructive'
                            : product.low_stock
                              ? 'text-amber-500'
                              : 'text-muted-foreground'
                        }
                      >
                        {product.stock} in stock
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))}
              {!filtered.length && (
                <p className='text-muted-foreground col-span-full py-8 text-center text-sm'>
                  No products match these filters.
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function InventoryTable({
  products,
  onUpdated
}: {
  products: PosProduct[];
  onUpdated: () => void;
}) {
  const router = useRouter();
  const [error, setError] = useState('');
  const [adjustId, setAdjustId] = useState('');
  async function adjust(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<unknown>(`/inventory/products/${adjustId}/adjust`, {
        method: 'POST',
        body: JSON.stringify({
          qty_change: Number(values.qty_change),
          reason: values.reason,
          note: values.note
        })
      });
      setAdjustId('');
      setError('');
      onUpdated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Stock adjustment failed.');
    }
  }
  async function updatePrice(product: PosProduct) {
    const value = window.prompt(
      `New selling price for ${product.name}`,
      String(product.selling_price)
    );
    if (value === null) return;
    const amount = Number(value);
    if (!Number.isFinite(amount) || amount < 0) {
      setError('Enter a valid selling price.');
      return;
    }
    try {
      await posApi<unknown>(`/inventory/products/${product.id}`, {
        method: 'PATCH',
        body: JSON.stringify({ selling_price: amount })
      });
      onUpdated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Product update failed.');
    }
  }
  async function deactivate(product: PosProduct) {
    if (!window.confirm(`Deactivate ${product.name}?`)) return;
    try {
      await posApi<unknown>(`/inventory/products/${product.id}`, { method: 'DELETE' });
      onUpdated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not deactivate product.');
    }
  }
  return (
    <>
      <div className='overflow-x-auto rounded-lg border'>
        <table className='w-full text-left text-sm'>
          <thead className='bg-muted/50'>
            <tr>
              {['Product', 'SKU / barcode', 'Price', 'Cost', 'Stock', 'Status', 'Actions'].map(
                (label) => (
                  <th key={label} className='px-4 py-3 font-medium'>
                    {label}
                  </th>
                )
              )}
            </tr>
          </thead>
          <tbody className='divide-y'>
            {products.length ? (
              products.map((product) => (
                <tr
                  key={product.id}
                  className='cursor-pointer hover:bg-muted/30'
                  onClick={() => router.push(`/dashboard/inventory/${product.id}`)}
                >
                  <td className='px-4 py-3 font-medium'>{product.name}</td>
                  <td className='px-4 py-3'>{product.sku || product.barcode || '—'}</td>
                  <td className='px-4 py-3'>{money(product.selling_price)}</td>
                  <td className='px-4 py-3'>{money(product.cost_price)}</td>
                  <td className='px-4 py-3'>
                    {product.stock}{' '}
                    {product.low_stock && <span className='text-destructive'>Low</span>}
                  </td>
                  <td className='px-4 py-3'>{product.is_active ? 'Active' : 'Inactive'}</td>
                  <td className='px-4 py-3'>
                    <div className='flex gap-1'>
                      <Button size='sm' variant='outline' onClick={() => setAdjustId(product.id)}>
                        Adjust stock
                      </Button>
                      <Button size='sm' variant='ghost' onClick={() => void updatePrice(product)}>
                        Price
                      </Button>
                      <Button size='sm' variant='ghost' onClick={() => void deactivate(product)}>
                        Deactivate
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} className='text-muted-foreground px-4 py-8 text-center'>
                  No products found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {error && (
        <p role='alert' className='text-destructive text-sm'>
          {error}
        </p>
      )}
      {adjustId && (
        <div className='fixed inset-0 z-50 grid place-items-center bg-black/40 p-4'>
          <Card className='w-full max-w-md'>
            <CardHeader>
              <CardTitle>Adjust stock</CardTitle>
              <CardDescription>
                {products.find((product) => product.id === adjustId)?.name}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={adjust} className='space-y-3'>
                <label className='block text-sm'>
                  Quantity change (use a negative number to remove stock)
                  <Input name='qty_change' type='number' required />
                </label>
                <label className='block text-sm'>
                  Reason
                  <select
                    className='bg-background mt-1 h-9 w-full rounded-md border px-3'
                    name='reason'
                  >
                    <option value='restock'>Restock</option>
                    <option value='damage'>Damage</option>
                    <option value='correction'>Correction</option>
                    <option value='return'>Return</option>
                  </select>
                </label>
                <label className='block text-sm'>
                  Note
                  <Input name='note' />
                </label>
                <div className='flex justify-end gap-2'>
                  <Button type='button' variant='outline' onClick={() => setAdjustId('')}>
                    Cancel
                  </Button>
                  <Button type='submit'>Save adjustment</Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      )}
    </>
  );
}

function CategoryManager() {
  const [categories, setCategories] = useState<PosCategory[]>([]);
  const [name, setName] = useState('');
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  useEffect(() => {
    posApi<unknown>('/inventory/categories')
      .then((result) => setCategories(unwrapList<PosCategory>(result.data)))
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load categories.')
      );
  }, [reload]);
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      await posApi<PosCategory>('/inventory/categories', {
        method: 'POST',
        body: JSON.stringify({ name })
      });
      setName('');
      setReload((value) => value + 1);
      setError('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create category.');
    }
  }
  const visibleCategories = categories.filter((category) =>
    category.name.toLowerCase().includes(query.toLowerCase())
  );
  return (
    <div className='relative'>
      <Button variant='outline' onClick={() => setOpen((value) => !value)}>
        Categories ({categories.length})
      </Button>
      {open && (
        <Card className='absolute right-0 top-11 z-20 w-80'>
          <CardHeader>
            <CardTitle>Product categories</CardTitle>
          </CardHeader>
          <CardContent className='space-y-3'>
            <Input
              aria-label='Search categories'
              placeholder='Search categories…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <div className='max-h-40 space-y-1 overflow-y-auto text-sm'>
              {visibleCategories.map((category) => (
                <p key={category.id} className='rounded border px-2 py-1'>
                  {category.name}
                </p>
              ))}
              {!visibleCategories.length && (
                <p className='text-muted-foreground py-3 text-center text-xs'>
                  No categories found.
                </p>
              )}
            </div>
            <form onSubmit={create} className='flex gap-2'>
              <Input
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder='New category'
                required
              />
              <Button size='sm'>Add</Button>
            </form>
            {error && (
              <p role='alert' className='text-destructive text-xs'>
                {error}
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function CreateProduct({ onCreated }: { onCreated: () => void }) {
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const [categories, setCategories] = useState<PosCategory[]>([]);
  useEffect(() => {
    posApi<unknown>('/inventory/categories')
      .then((result) => setCategories(unwrapList<PosCategory>(result.data)))
      .catch(() => setCategories([]));
  }, []);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const raw = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<PosProduct>('/inventory/products', {
        method: 'POST',
        body: JSON.stringify({
          name: raw.name,
          sku: raw.sku,
          barcode: raw.barcode,
          category_id: raw.category_id || undefined,
          selling_price: Number(raw.selling_price),
          cost_price: Number(raw.cost_price || 0),
          stock: Number(raw.stock || 0),
          reorder_point: Number(raw.reorder_point || 0),
          description: raw.description,
          is_service: raw.is_service === 'on',
          is_active: true
        })
      });
      onCreated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create product.');
    } finally {
      setPending(false);
    }
  }
  return (
    <div className='fixed inset-0 z-50 grid place-items-center overflow-y-auto bg-black/60 p-4'>
      <Card className='my-4 w-full max-w-3xl shadow-2xl'>
        <CardHeader>
          <CardTitle>New product</CardTitle>
          <CardDescription>Amounts are entered in Tanzanian shillings.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className='grid gap-3 sm:grid-cols-2'>
            {[
              ['name', 'Name', 'text'],
              ['sku', 'SKU', 'text'],
              ['barcode', 'Barcode', 'text'],
              ['selling_price', 'Selling price (TZS)', 'number'],
              ['cost_price', 'Cost price (TZS)', 'number'],
              ['stock', 'Opening stock', 'number'],
              ['reorder_point', 'Low stock alert at', 'number'],
              ['description', 'Description', 'text']
            ].map(([name, label, type]) => (
              <label key={name} className='text-sm'>
                {label}
                <Input
                  name={name}
                  type={type}
                  required={name === 'name' || name === 'selling_price'}
                  min={type === 'number' ? 0 : undefined}
                />
              </label>
            ))}
            <label className='text-sm'>
              Category
              <select
                name='category_id'
                className='bg-background mt-1 h-9 w-full rounded-md border px-3'
              >
                <option value=''>Uncategorized</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
            </label>
            <label className='flex items-center gap-2 text-sm'>
              <input type='checkbox' name='is_service' /> Service (no stock tracking)
            </label>
            {error && (
              <p role='alert' className='text-destructive sm:col-span-2'>
                {error}
              </p>
            )}
            <div className='sm:col-span-2'>
              <Button isLoading={pending}>Create product</Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

function CreateCustomer({ onCreated }: { onCreated: () => void }) {
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      await posApi<PosCustomer>('/customers', {
        method: 'POST',
        body: JSON.stringify({
          name: values.name,
          phone: values.phone,
          email: values.email,
          segment: values.segment,
          credit_limit: values.credit_limit === '' ? undefined : Number(values.credit_limit),
          notes: values.notes
        })
      });
      onCreated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create customer.');
    } finally {
      setPending(false);
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>New customer</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className='grid gap-3 sm:grid-cols-2'>
          <label className='text-sm'>
            Full name
            <Input name='name' required />
          </label>
          <label className='text-sm'>
            Phone
            <Input name='phone' />
          </label>
          <label className='text-sm'>
            Email
            <Input type='email' name='email' />
          </label>
          <label className='text-sm'>
            Segment
            <select name='segment' className='bg-background mt-1 h-9 w-full rounded-md border px-3'>
              <option>New</option>
              <option>Regular</option>
              <option>VIP</option>
              <option>Occasional</option>
            </select>
          </label>
          <label className='text-sm'>
            Credit limit (TZS, -1 for unlimited)
            <Input type='number' name='credit_limit' defaultValue='0' />
          </label>
          <label className='text-sm'>
            Notes
            <Input name='notes' />
          </label>
          {error && <p className='text-destructive sm:col-span-2'>{error}</p>}
          <div className='sm:col-span-2'>
            <Button isLoading={pending}>Save customer</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function SupplierOperations() {
  const [suppliers, setSuppliers] = useState<Record<string, unknown>[]>([]);
  const [message, setMessage] = useState('');
  const [reload, setReload] = useState(0);
  useEffect(() => {
    posApi<unknown>('/suppliers?page=1&page_size=100')
      .then((result) => setSuppliers(unwrapList<Record<string, unknown>>(result.data)))
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
  );
}

function SalesTable({
  transactions,
  onUpdated
}: {
  transactions: PosTransaction[];
  onUpdated: () => void;
}) {
  const router = useRouter();
  const [message, setMessage] = useState('');
  const [query, setQuery] = useState('');
  const [status, setStatus] = useState('all');
  const [payment, setPayment] = useState('all');
  const [range, setRange] = useState('all');
  const now = Date.now();
  const filtered = transactions.filter((transaction) => {
    const haystack =
      `${transaction.txn_number} ${transaction.customer_name || ''} ${transaction.payment_method}`.toLowerCase();
    const matchesQuery = haystack.includes(query.toLowerCase());
    const matchesStatus = status === 'all' || transaction.status.toLowerCase() === status;
    const matchesPayment =
      payment === 'all' || transaction.payment_method.toLowerCase() === payment.toLowerCase();
    const age = now - new Date(transaction.created_at).getTime();
    const matchesRange = range === 'all' || (range === 'today' ? age < 86400000 : age < 604800000);
    return matchesQuery && matchesStatus && matchesPayment && matchesRange;
  });
  const revenue = filtered.reduce((sum, transaction) => sum + Number(transaction.total || 0), 0);
  const paid = filtered.filter((transaction) => transaction.status.toLowerCase() === 'paid').length;
  const average = filtered.length ? revenue / filtered.length : 0;
  async function action(id: string, actionName: 'refund' | 'void') {
    const reason = window.prompt(`Reason for ${actionName}:`);
    if (reason === null) return;
    try {
      await posApi<unknown>(`/pos/transactions/${id}/${actionName}`, {
        method: 'POST',
        body: JSON.stringify({ reason })
      });
      setMessage(`Transaction ${actionName} completed.`);
      onUpdated();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : `Could not ${actionName} transaction.`);
    }
  }
  function clearFilters() {
    setQuery('');
    setStatus('all');
    setPayment('all');
    setRange('all');
  }
  return (
    <div className='space-y-4'>
      <div className='grid grid-cols-2 gap-3 xl:grid-cols-4'>
        {[
          ['Sales revenue', money(revenue), 'From matching sales'],
          ['Transactions', String(filtered.length), `${paid} paid`],
          ['Average sale', money(average), 'Per transaction'],
          [
            'Refund / void ready',
            String(filtered.filter((item) => item.status === 'paid').length),
            'Paid sales available'
          ]
        ].map(([label, value, detail]) => (
          <Card key={label} className='min-w-0'>
            <CardHeader className='p-3 pb-2 sm:p-6 sm:pb-2'>
              <CardDescription className='truncate text-xs sm:text-sm'>{label}</CardDescription>
              <CardTitle className='truncate text-lg sm:text-2xl'>{value}</CardTitle>
            </CardHeader>
            <CardContent className='truncate p-3 pt-0 text-[11px] text-muted-foreground sm:p-6 sm:pt-0 sm:text-xs'>
              {detail}
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardHeader>
          <div className='flex flex-wrap items-start justify-between gap-3'>
            <div>
              <CardTitle>Sales history</CardTitle>
              <CardDescription>
                Search receipts, review payment details, and manage completed sales.
              </CardDescription>
            </div>
            <Button variant='outline' className='hidden sm:inline-flex' onClick={clearFilters}>
              Clear filters
            </Button>
          </div>
          <div className='sales-filter-toolbar grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(520px,auto)]'>
            <Input
              className='h-11'
              aria-label='Search sales'
              placeholder='Search receipt, customer, or payment…'
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <div className='sales-filter-row'>
              <select
                aria-label='Filter by status'
                className='bg-background h-11 rounded-md border px-3 text-sm'
                value={status}
                onChange={(event) => setStatus(event.target.value)}
              >
                <option value='all'>All statuses</option>
                <option value='paid'>Paid</option>
                <option value='refunded'>Refunded</option>
                <option value='voided'>Voided</option>
              </select>
              <select
                aria-label='Filter by payment'
                className='bg-background h-11 rounded-md border px-3 text-sm'
                value={payment}
                onChange={(event) => setPayment(event.target.value)}
              >
                <option value='all'>All payments</option>
                {['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit'].map((method) => (
                  <option key={method}>{method}</option>
                ))}
              </select>
              <select
                aria-label='Filter by date'
                className='bg-background h-11 rounded-md border px-3 text-sm'
                value={range}
                onChange={(event) => setRange(event.target.value)}
              >
                <option value='all'>All dates</option>
                <option value='today'>Today</option>
                <option value='7d'>Last 7 days</option>
              </select>
              <Button
                type='button'
                size='icon'
                variant='outline'
                className='sm:hidden'
                aria-label='Clear filters'
                title='Clear filters'
                onClick={clearFilters}
              >
                <Icons.close className='h-4 w-4' />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[760px] text-left text-sm'>
              <thead className='bg-muted/40'>
                <tr>
                  {['Receipt', 'Customer', 'Date', 'Payment', 'Status', 'Total', 'Actions'].map(
                    (heading) => (
                      <th key={heading} className='px-3 py-3 font-medium'>
                        {heading}
                      </th>
                    )
                  )}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {filtered.length ? (
                  filtered.map((transaction) => (
                    <tr key={transaction.id} className='hover:bg-muted/30'>
                      <td className='px-3 py-3 font-medium'>{transaction.txn_number}</td>
                      <td className='px-3 py-3'>
                        {transaction.customer_name || 'Walk-in customer'}
                      </td>
                      <td className='text-muted-foreground px-3 py-3'>
                        {new Date(transaction.created_at).toLocaleString()}
                      </td>
                      <td className='px-3 py-3'>{transaction.payment_method}</td>
                      <td className='px-3 py-3 capitalize'>{transaction.status}</td>
                      <td className='px-3 py-3 font-semibold'>{money(transaction.total)}</td>
                      <td className='px-3 py-3'>
                        <div className='flex gap-1'>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => router.push(`/dashboard/sales/${transaction.id}`)}
                          >
                            View
                          </Button>
                          {transaction.status === 'paid' && (
                            <>
                              <Button
                                size='sm'
                                variant='ghost'
                                onClick={() => void action(transaction.id, 'refund')}
                              >
                                Refund
                              </Button>
                              <Button
                                size='sm'
                                variant='ghost'
                                onClick={() => void action(transaction.id, 'void')}
                              >
                                Void
                              </Button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={7} className='text-muted-foreground px-3 py-10 text-center'>
                      No sales match these filters.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {message && (
            <p role='status' className='text-muted-foreground pt-3 text-sm'>
              {message}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function CreditTable({
  tabs,
  customers,
  onUpdated
}: {
  tabs: PosCreditTab[];
  customers: PosCustomer[];
  onUpdated: () => void;
}) {
  const [amounts, setAmounts] = useState<Record<string, string>>({});
  const [message, setMessage] = useState('');
  async function pay(tab: PosCreditTab) {
    const amount = Number(amounts[tab.customer_id]);
    if (!amount || amount <= 0) {
      setMessage('Enter a payment amount greater than zero.');
      return;
    }
    try {
      await posApi<unknown>(`/credits/customers/${tab.customer_id}/payments`, {
        method: 'POST',
        body: JSON.stringify({ amount, method: 'Cash' })
      });
      setMessage('Payment recorded.');
      setAmounts({});
      onUpdated();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Payment could not be recorded.');
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Outstanding tabs</CardTitle>
      </CardHeader>
      <CardContent>
        <div className='overflow-x-auto'>
          <table className='w-full text-left text-sm'>
            <thead>
              <tr className='border-b'>
                {['Customer', 'Original', 'Paid', 'Balance', 'Status', 'Record payment'].map(
                  (item) => (
                    <th className='px-2 py-3' key={item}>
                      {item}
                    </th>
                  )
                )}
              </tr>
            </thead>
            <tbody className='divide-y'>
              {tabs.map((tab) => (
                <tr key={tab.id}>
                  <td className='px-2 py-3'>
                    {customers.find((customer) => customer.id === tab.customer_id)?.name ||
                      tab.customer_id.slice(0, 8)}
                  </td>
                  <td className='px-2 py-3'>{money(tab.amount)}</td>
                  <td className='px-2 py-3'>{money(tab.amount_paid)}</td>
                  <td className='px-2 py-3 font-semibold'>{money(tab.balance)}</td>
                  <td className='px-2 py-3'>
                    {tab.status}
                    {tab.is_overdue ? ' · overdue' : ''}
                  </td>
                  <td className='px-2 py-3'>
                    <div className='flex gap-2'>
                      <Input
                        className='w-28'
                        type='number'
                        min='1'
                        placeholder='Amount'
                        value={amounts[tab.customer_id] || ''}
                        onChange={(event) =>
                          setAmounts((current) => ({
                            ...current,
                            [tab.customer_id]: event.target.value
                          }))
                        }
                      />
                      <Button size='sm' onClick={() => void pay(tab)}>
                        Record
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {!tabs.length && (
                <tr>
                  <td colSpan={6} className='text-muted-foreground px-2 py-8 text-center'>
                    No credit tabs found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {message && (
          <p role='status' className='text-muted-foreground pt-3 text-sm'>
            {message}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function ReceiptDialog({
  transaction,
  onClose
}: {
  transaction: PosTransaction;
  onClose: () => void;
}) {
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState('');
  async function send() {
    try {
      await posApi<unknown>(`/pos/transactions/${transaction.id}/receipt`, {
        method: 'POST',
        body: JSON.stringify({ email })
      });
      setStatus('Receipt emailed.');
    } catch (reason) {
      setStatus(reason instanceof Error ? reason.message : 'Email failed.');
    }
  }
  return (
    <div className='fixed inset-0 z-50 grid place-items-center bg-black/40 p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <CardTitle>Sale complete · {transaction.txn_number}</CardTitle>
          <CardDescription>
            Receipt total {money(transaction.total)}. Print this receipt or email a copy.
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-3'>
          <Button className='w-full' onClick={() => window.print()}>
            Print receipt
          </Button>
          <div className='flex gap-2'>
            <Input
              type='email'
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder='Customer email'
            />
            <Button variant='outline' onClick={() => void send()}>
              Email
            </Button>
          </div>
          {status && (
            <p role='status' className='text-sm'>
              {status}
            </p>
          )}
          <Button className='w-full' variant='ghost' onClick={onClose}>
            Close
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function StoreSwitcher() {
  const [stores, setStores] = useState<{ id: string; name: string }[]>([]);
  const [error, setError] = useState('');
  useEffect(() => {
    posApi<unknown>('/accounts/me')
      .then((result) => {
        const payload = result.data as {
          stores?: { id: string; name: string }[];
          store?: { id: string };
        };
        setStores(
          (payload.stores || []).map((store) => ({
            ...store,
            name: `${store.name}${store.id === payload.store?.id ? ' (current)' : ''}`
          }))
        );
      })
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load branches.')
      );
  }, []);
  async function change(store_id: string) {
    try {
      const session = await fetch('/api/pos-session', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ store_id })
      });
      if (!session.ok) {
        const data = await session.json();
        throw new Error(data.message || 'Could not switch branch.');
      }
      window.location.reload();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not switch branch.');
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Active branch</CardTitle>
        <CardDescription>Branch context is embedded in your API access token.</CardDescription>
      </CardHeader>
      <CardContent className='space-y-2'>
        <select
          className='bg-background h-9 w-full rounded-md border px-3'
          onChange={(event) => void change(event.target.value)}
          defaultValue=''
        >
          <option value='' disabled>
            Select branch
          </option>
          {stores.map((store) => (
            <option value={store.id} key={store.id}>
              {store.name}
            </option>
          ))}
        </select>
        {error && <p className='text-destructive text-sm'>{error}</p>}
      </CardContent>
    </Card>
  );
}

function DataTable({ columns, rows }: { columns: string[]; rows: string[][] }) {
  return (
    <div className='overflow-x-auto rounded-lg border'>
      <table className='w-full text-left text-sm'>
        <thead className='bg-muted/50'>
          <tr>
            {columns.map((column) => (
              <th key={column} className='px-4 py-3 font-medium'>
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className='divide-y'>
          {rows.length ? (
            rows.map((row, i) => (
              <tr key={i} className='hover:bg-muted/30'>
                {row.map((cell, j) => (
                  <td key={`${i}-${j}`} className='px-4 py-3'>
                    {cell}
                  </td>
                ))}
              </tr>
            ))
          ) : (
            <tr>
              <td className='text-muted-foreground px-4 py-8 text-center' colSpan={columns.length}>
                No records found.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function ResourcePage({ pathname }: { pathname: string }) {
  const endpoint = pathname.includes('/suppliers')
    ? '/suppliers?page=1&page_size=50'
    : pathname.includes('/expenses')
      ? '/expenses?page=1&page_size=50'
      : pathname.includes('/staff')
        ? '/staff?page=1&page_size=50'
        : pathname.includes('/stores')
          ? '/stores'
          : pathname.includes('/reports')
            ? '/reports/exports?page=1&page_size=50'
            : '/accounts/organisation';
  const title =
    pathname
      .split('/')
      .filter(Boolean)
      .at(-1)
      ?.replace(/^./, (letter) => letter.toUpperCase()) || 'Workspace';
  const [records, setRecords] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [pending, setPending] = useState(false);
  const [reload, setReload] = useState(0);
  useEffect(() => {
    let active = true;
    posApi<unknown>(endpoint)
      .then((res) => {
        if (active) setRecords(unwrapList<Record<string, unknown>>(res.data));
      })
      .catch((reason) => {
        if (active) setError(reason instanceof Error ? reason.message : 'Request failed');
      });
    return () => {
      active = false;
    };
  }, [endpoint, reload]);
  const fields = pathname.includes('/suppliers')
    ? [
        ['name', 'Supplier name'],
        ['phone', 'Phone'],
        ['category', 'Category']
      ]
    : pathname.includes('/expenses')
      ? [
          ['title', 'Expense title'],
          ['amount', 'Amount (TZS)'],
          ['category', 'Category'],
          ['payment_method', 'Payment method']
        ]
      : pathname.includes('/staff')
        ? [
            ['first_name', 'First name'],
            ['last_name', 'Last name'],
            ['phone', 'Phone'],
            ['email', 'Email'],
            ['role', 'Role'],
            ['shift', 'Shift'],
            ['pin', '4 digit POS PIN']
          ]
        : pathname.includes('/stores')
          ? [
              ['name', 'Branch name'],
              ['code', 'Branch code'],
              ['area', 'Area']
            ]
          : [];
  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const values = Object.fromEntries(new FormData(event.currentTarget));
    try {
      const payload = Object.fromEntries(
        Object.entries(values).map(([key, value]) => [
          key,
          ['amount', 'payment_terms'].includes(key) ? Number(value) : value
        ])
      );
      await posApi<unknown>(endpoint.split('?')[0], {
        method: 'POST',
        body: JSON.stringify(payload)
      });
      setShowForm(false);
      setReload((n) => n + 1);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not save the record.');
    } finally {
      setPending(false);
    }
  }
  const keys = records.length
    ? Object.keys(records[0])
        .filter(
          (key) => !['id', 'organisation_id', 'store_id', 'created_at', 'updated_at'].includes(key)
        )
        .slice(0, 6)
    : [];
  return (
    <>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h1 className='text-2xl font-bold tracking-tight'>{title}</h1>
          <p className='text-muted-foreground'>
            Manage {title.toLowerCase()} for the active store.
          </p>
        </div>
        {fields.length > 0 && (
          <Button onClick={() => setShowForm((value) => !value)}>
            {showForm ? 'Close form' : `Add ${title.slice(0, -1)}`}
          </Button>
        )}
        {pathname.includes('/reports') && (
          <ReportGenerator onGenerated={() => setReload((n) => n + 1)} />
        )}
      </div>
      {error && (
        <p className='text-destructive text-sm' role='alert'>
          {error}
        </p>
      )}
      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle>New {title.slice(0, -1)}</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={create} className='grid gap-4 sm:grid-cols-2'>
              {fields.map(([name, label]) => (
                <div key={name} className='space-y-1'>
                  <label className='text-sm font-medium' htmlFor={name}>
                    {label}
                  </label>
                  <Input
                    id={name}
                    name={name}
                    required={name === 'name' || name === 'title' || name === 'first_name'}
                    type={name === 'amount' ? 'number' : 'text'}
                  />
                </div>
              ))}
              <div className='sm:col-span-2'>
                <Button isLoading={pending}>Save</Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
      {pathname.includes('/staff') ? (
        <StaffTable records={records} onUpdated={() => setReload((n) => n + 1)} />
      ) : (
        <DataTable
          columns={keys.map((key) =>
            key.replaceAll('_', ' ').replace(/^./, (letter) => letter.toUpperCase())
          )}
          rows={records.map((record) =>
            keys.map((key) =>
              typeof record[key] === 'object'
                ? JSON.stringify(record[key])
                : String(record[key] ?? '—')
            )
          )}
        />
      )}
    </>
  );
}

function StaffTable({
  records,
  onUpdated
}: {
  records: Record<string, unknown>[];
  onUpdated: () => void;
}) {
  const [message, setMessage] = useState('');
  async function togglePermission(
    record: Record<string, unknown>,
    permission: 'can_refund' | 'can_discount' | 'can_view_reports'
  ) {
    const id = String(record.id || record.user_id || '');
    try {
      await posApi<unknown>(`/staff/${id}/permissions`, {
        method: 'PATCH',
        body: JSON.stringify({ [permission]: !record[permission] })
      });
      setMessage('Staff permissions updated.');
      onUpdated();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not update permissions.');
    }
  }
  async function deactivate(record: Record<string, unknown>) {
    const id = String(record.id || record.user_id || '');
    if (
      !window.confirm(
        `Deactivate ${String(record.full_name || record.first_name || 'this staff member')}?`
      )
    )
      return;
    try {
      await posApi<unknown>(`/staff/${id}`, { method: 'DELETE' });
      setMessage('Staff account deactivated.');
      onUpdated();
    } catch (reason) {
      setMessage(reason instanceof Error ? reason.message : 'Could not deactivate staff.');
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Team members</CardTitle>
      </CardHeader>
      <CardContent>
        <div className='overflow-x-auto'>
          <table className='w-full text-left text-sm'>
            <thead>
              <tr className='border-b'>
                {['Name', 'Phone', 'Role', 'Shift', 'POS permissions', 'Actions'].map((heading) => (
                  <th key={heading} className='px-3 py-3'>
                    {heading}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className='divide-y'>
              {records.map((record) => (
                <tr key={String(record.id)}>
                  <td className='px-3 py-3 font-medium'>
                    {String(
                      record.full_name || `${record.first_name || ''} ${record.last_name || ''}`
                    )}
                  </td>
                  <td className='px-3 py-3'>{String(record.phone || '—')}</td>
                  <td className='px-3 py-3'>{String(record.role || 'staff')}</td>
                  <td className='px-3 py-3'>
                    {String(record.shift_display || record.shift || '—')}
                  </td>
                  <td className='px-3 py-3'>
                    <div className='flex flex-wrap gap-1'>
                      {(['can_refund', 'can_discount', 'can_view_reports'] as const).map(
                        (permission) => (
                          <Button
                            key={permission}
                            size='sm'
                            variant={record[permission] ? 'default' : 'outline'}
                            onClick={() => void togglePermission(record, permission)}
                          >
                            {permission.replace('can_', '').replaceAll('_', ' ')}:{' '}
                            {record[permission] ? 'yes' : 'no'}
                          </Button>
                        )
                      )}
                    </div>
                  </td>
                  <td className='px-3 py-3'>
                    <Button size='sm' variant='ghost' onClick={() => void deactivate(record)}>
                      Deactivate
                    </Button>
                  </td>
                </tr>
              ))}
              {!records.length && (
                <tr>
                  <td colSpan={6} className='text-muted-foreground px-3 py-8 text-center'>
                    No staff found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        {message && (
          <p role='status' className='text-muted-foreground mt-3 text-sm'>
            {message}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function ReportGenerator({ onGenerated }: { onGenerated: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function generate() {
    setBusy(true);
    setError('');
    try {
      const response = await fetch('/api/pos/reports/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ report_type: 'sales', format: 'csv', range: 'today' })
      });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new Error(payload.message || 'Could not generate report.');
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'ziada-sales-report.csv';
      link.click();
      URL.revokeObjectURL(url);
      onGenerated();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not generate report.');
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className='flex items-center gap-2'>
      <Button isLoading={busy} onClick={() => void generate()}>
        Download sales CSV
      </Button>
      {error && <span className='text-destructive text-xs'>{error}</span>}
    </div>
  );
}

function BusinessSettings({ initialValue }: { initialValue?: Record<string, unknown> }) {
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setSaved(false);
    const fields = Object.fromEntries(new FormData(event.currentTarget));
    const payload = Object.fromEntries(
      Object.entries(fields).map(([key, value]) => [key, value || undefined])
    );
    try {
      await posApi<unknown>('/accounts/organisation', {
        method: 'PATCH',
        body: JSON.stringify(payload)
      });
      setSaved(true);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not update business profile.');
    } finally {
      setBusy(false);
    }
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Business profile</CardTitle>
        <CardDescription>
          Update the organisation details used in your POS workspace.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className='grid gap-3 sm:grid-cols-2'>
          {[
            ['name', 'Business name'],
            ['tin', 'TIN'],
            ['phone', 'Business phone'],
            ['email', 'Business email'],
            ['region', 'Region'],
            ['district', 'District'],
            ['ward', 'Ward'],
            ['street', 'Street'],
            ['address', 'Address']
          ].map(([name, label]) => (
            <label key={name} className='text-sm'>
              {label}
              <Input name={name} defaultValue={String(initialValue?.[name] || '')} />
            </label>
          ))}
          {error && (
            <p role='alert' className='text-destructive sm:col-span-2'>
              {error}
            </p>
          )}
          {saved && (
            <p role='status' className='text-sm text-green-700 sm:col-span-2'>
              Business profile saved.
            </p>
          )}
          <div className='sm:col-span-2'>
            <Button isLoading={busy}>Save business profile</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
