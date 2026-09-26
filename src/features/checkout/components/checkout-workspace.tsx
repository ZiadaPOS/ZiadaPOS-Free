'use client';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { money, type PosCategory, type PosCustomer, type PosProduct } from '@/lib/pos-api';
import { Icons } from '@/components/icons';
import { Dispatch, FormEvent, SetStateAction, useState } from 'react';
import { useText } from '@/lib/i18n';

export function CheckoutWorkspace({
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
  const text = useText();
  const [activeCategory, setActiveCategory] = useState('');
  const [showCheckoutDetails, setShowCheckoutDetails] = useState(false);
  const visibleProducts = activeCategory
    ? filteredProducts.filter(
        (product) =>
          product.category_id === activeCategory || product.category?.id === activeCategory
      )
    : filteredProducts;
  const itemCount = Object.values(cart).reduce((sum, quantity) => sum + quantity, 0);
  const saleForm = (
    <form onSubmit={checkout} className='space-y-4'>
      {!showCheckoutDetails && (
        <div className='space-y-3 border-b pb-4'>
          {products
            .filter((product) => cart[product.id])
            .map((product) => (
              <div
                key={product.id}
                className='grid grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-2 text-sm'
              >
                <span className='min-w-0 break-words'>
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
      )}
      <div className='flex items-center justify-between text-sm font-semibold'>
        <span>{text('Items total', 'Jumla ya bidhaa')}</span>
        <span>{money(subtotal)}</span>
      </div>
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
              {['Cash', 'M-Pesa', 'Tigo Pesa', 'Bank', 'Credit'].map((method) => (
                <option key={method}>{method}</option>
              ))}
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
            <label className='text-sm'>
              {text('Reference', 'Kumbukumbu')}
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
          <Button className='h-11 w-full' isLoading={busy} disabled={!itemCount}>
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
        <h1 className='text-2xl font-bold tracking-tight'>{text('Checkout', 'Malipo')}</h1>
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
                aria-label={text('Search products', 'Tafuta bidhaa')}
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
                    {product.sku || product.barcode || text('No SKU', 'Hakuna SKU')} ·{' '}
                    {text('stock', 'stoo')}{' '}
                    {product.is_service ? text('service', 'huduma') : product.stock}
                  </p>
                  <p className='mt-3 font-semibold'>{money(product.selling_price)}</p>
                </button>
              ))
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
            <CardTitle>{text('Current sale', 'Mauzo ya sasa')}</CardTitle>
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
            className='bg-background absolute inset-x-0 bottom-0 max-h-[90svh] overflow-y-auto rounded-t-2xl border p-4 shadow-2xl'
            onClick={(event) => event.stopPropagation()}
          >
            <div className='mb-4 flex items-center justify-between'>
              <div>
                <h2 className='text-lg font-semibold'>{text('Current sale', 'Mauzo ya sasa')}</h2>
                <p className='text-muted-foreground text-sm'>
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
