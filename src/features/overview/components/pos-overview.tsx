'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  XAxis,
  YAxis,
  Bar,
  BarChart,
  Pie,
  PieChart,
  LabelList,
  Legend
} from 'recharts';
import { useRouter } from 'next/navigation';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle
} from '@/components/ui/card';
import {
  ChartContainer,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig
} from '@/components/ui/chart';
import PageContainer from '@/components/layout/page-container';
import { Icons } from '@/components/icons';
import { money, posApi, unwrapList, type PosProduct, type PosTransaction } from '@/lib/pos-api';
import { useText } from '@/lib/i18n';

type SalesSummary = {
  gross_sales?: number;
  net_sales?: number;
  profit?: number;
  transactions?: number;
  by_method?: Record<string, number>;
  top_products?: Array<{ product_name?: string; qty_sold?: number; sales?: number }>;
};

type DailyPoint = { date: string; label: string; sales: number; profit: number };
type ProductPoint = { name: string; sales: number };
type PaymentPoint = { method: string; amount: number; fill: string; key: string };

const paymentColors = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)'
];

function dayKey(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function recordOf(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' ? (value as Record<string, unknown>) : {};
}

function amountOf(value: unknown) {
  const amount = Number(value);
  return Number.isFinite(amount) ? amount : 0;
}

function paymentKey(value: string) {
  return (
    value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_|_$/g, '') || 'other'
  );
}

const trendConfig = {
  sales: { label: 'Sales', color: 'var(--chart-1)' },
  profit: { label: 'Profit', color: 'var(--chart-2)' }
} satisfies ChartConfig;

export default function PosOverview() {
  const text = useText();
  const router = useRouter();
  const [summary, setSummary] = useState<SalesSummary>({});
  const [transactions, setTransactions] = useState<PosTransaction[]>([]);
  const [lowStock, setLowStock] = useState<PosProduct[]>([]);
  const [trend, setTrend] = useState<DailyPoint[]>([]);
  const [topProducts, setTopProducts] = useState<ProductPoint[]>([]);
  const [userName, setUserName] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [smsBalance, setSmsBalance] = useState<{ balance: number; configured: boolean } | null>(
    null
  );

  const loadDashboard = useCallback(async () => {
    setError('');
    setLoading(true);

    const today = new Date();
    const todayKey = dayKey(today);
    const dates = Array.from({ length: 7 }, (_, index) => {
      const date = new Date(today);
      date.setDate(today.getDate() - (6 - index));
      return date;
    });

    try {
      const [coreResults, dailyResults] = await Promise.all([
        Promise.allSettled([
          posApi<SalesSummary>(`/pos/summary?date_from=${todayKey}&date_to=${todayKey}`),
          posApi<unknown>('/pos/transactions?page=1&page_size=6'),
          posApi<unknown>('/inventory/low-stock'),
          posApi<{ balance: number; configured: boolean }>('/accounts/sms-balance')
        ]),
        Promise.allSettled(
          dates.slice(0, -1).map((date) => {
            const dateString = dayKey(date);
            return posApi<SalesSummary>(
              `/pos/summary?date_from=${dateString}&date_to=${dateString}`
            );
          })
        )
      ]);
      const [todayResult, transactionResult, lowStockResult] = coreResults;

      if (todayResult.status === 'fulfilled') setSummary(todayResult.value.data);
      else throw todayResult.reason;

      if (transactionResult.status === 'fulfilled') {
        setTransactions(unwrapList<PosTransaction>(transactionResult.value.data));
      }
      if (lowStockResult.status === 'fulfilled') {
        setLowStock(unwrapList<PosProduct>(lowStockResult.value.data));
      }
      const smsResult = coreResults[3];
      if (smsResult.status === 'fulfilled') setSmsBalance(smsResult.value.data);

      const resolvedDaily = dailyResults.flatMap((result, index) => {
        if (result.status !== 'fulfilled') return [];
        const values = result.value.data;
        return [
          {
            date: dayKey(dates[index]),
            label: new Intl.DateTimeFormat('en', { weekday: 'short' }).format(dates[index]),
            sales: amountOf(values.gross_sales ?? values.net_sales),
            profit: amountOf(values.profit)
          }
        ];
      });
      setTrend([
        ...resolvedDaily,
        {
          date: todayKey,
          label: new Intl.DateTimeFormat('en', { weekday: 'short' }).format(today),
          sales: amountOf(todayResult.value.data.gross_sales ?? todayResult.value.data.net_sales),
          profit: amountOf(todayResult.value.data.profit)
        }
      ]);
      const products = Array.isArray(todayResult.value.data.top_products)
        ? todayResult.value.data.top_products
        : [];
      setTopProducts(
        products.slice(0, 5).map((product) => ({
          name: product.product_name || 'Product',
          sales: amountOf(product.sales)
        }))
      );
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Dashboard data could not be loaded.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    try {
      const profile = recordOf(JSON.parse(localStorage.getItem('pos_user') || '{}'));
      setUserName(String(profile.full_name || '').split(' ')[0]);
    } catch {
      setUserName('');
    }
    void loadDashboard();
  }, [loadDashboard]);

  const paymentData = useMemo<PaymentPoint[]>(() => {
    const methods = Object.entries(summary.by_method || {}).filter(
      ([, amount]) => amountOf(amount) > 0
    );
    return methods.map(([method, amount], index) => ({
      method,
      amount: amountOf(amount),
      fill: paymentColors[index % paymentColors.length],
      key: paymentKey(method)
    }));
  }, [summary.by_method]);

  const paymentConfig = useMemo<ChartConfig>(
    () =>
      Object.fromEntries(
        paymentData.map((entry) => [entry.key, { label: entry.method, color: entry.fill }])
      ),
    [paymentData]
  );

  const stats = [
    {
      label: 'Sales today',
      value: money(summary.gross_sales ?? summary.net_sales),
      note: 'Gross sales recorded today',
      detail: 'Today',
      icon: Icons.trendingUp
    },
    {
      label: 'Transactions',
      value: loading ? '—' : String(amountOf(summary.transactions)),
      note: 'Completed and credit receipts',
      detail: 'Today',
      icon: Icons.post
    },
    {
      label: 'Profit today',
      value: money(summary.profit),
      note: 'From recorded product costs',
      detail: 'Today',
      icon: Icons.billing
    },
    {
      label: 'Low stock',
      value: loading ? '—' : String(lowStock.length),
      note: 'Products needing a stock review',
      detail: 'Inventory',
      icon: Icons.product
    }
  ];

  const customerCount = new Set(
    transactions.map((transaction) => transaction.customer_id).filter(Boolean)
  ).size;
  const creditSales = transactions
    .filter(
      (transaction) => transaction.status === 'credit' || transaction.payment_method === 'Credit'
    )
    .reduce((total, transaction) => total + amountOf(transaction.total), 0);
  const averageTicket = transactions.length
    ? transactions.reduce((total, transaction) => total + amountOf(transaction.total), 0) /
      transactions.length
    : 0;
  const criticalStock = lowStock.filter((product) => product.stock <= 0).length;

  return (
    <PageContainer>
      <div className='flex min-h-[calc(100svh-6rem)] min-w-0 flex-1 flex-col gap-4'>
        <div className='flex flex-col items-stretch justify-between gap-3 sm:flex-row sm:items-center'>
          <h2 className='text-xl font-bold tracking-tight sm:text-2xl'>
            Hi{userName ? `, ${userName}` : ''}! Welcome back 👋
          </h2>
          <div className='grid w-full grid-cols-[3fr_7fr] items-center gap-2 sm:flex sm:w-auto sm:justify-end'>
            <Badge variant='outline' className='hidden sm:inline-flex'>
              {text('Today · TZS', 'Leo · TZS')}
            </Badge>
            <Button
              className='w-full sm:w-auto'
              variant='outline'
              onClick={() => void loadDashboard()}
              isLoading={loading}
            >
              Refresh
            </Button>
            <Button
              className='w-full min-w-0 flex-row whitespace-nowrap sm:w-auto sm:min-w-max'
              onClick={() => router.push('/checkout')}
            >
              <span className='inline-flex items-center gap-1.5'>
                <Icons.add className='size-4 shrink-0' />
                <span>{text('New sale', 'Mauzo mapya')}</span>
              </span>
            </Button>
          </div>
        </div>

        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-3 text-sm'
          >
            {error}
          </div>
        )}

        <div className='*:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card dark:*:data-[slot=card]:bg-card grid grid-cols-2 gap-3 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:shadow-xs sm:gap-4 lg:grid-cols-4'>
          {stats.map(({ label, value, note, detail, icon: Icon }) => (
            <Card className='@container/card' key={label}>
              <CardHeader>
                <CardDescription>{label}</CardDescription>
                <CardTitle className='text-2xl font-semibold tabular-nums @[250px]/card:text-3xl'>
                  {loading ? '—' : value}
                </CardTitle>
                <CardAction>
                  <Badge variant='outline'>
                    <Icon className='size-3.5' />
                    {detail}
                  </Badge>
                </CardAction>
              </CardHeader>
              <CardFooter className='flex-col items-start gap-1.5 text-sm'>
                <div className='line-clamp-1 flex gap-2 font-medium'>{note}</div>
                <div className='text-muted-foreground'>Active store performance</div>
              </CardFooter>
            </Card>
          ))}
        </div>

        <div className='grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-7'>
          <Card className='col-span-4'>
            <CardHeader>
              <CardTitle>{text('Sales and profit', 'Mauzo na faida')}</CardTitle>
              <CardDescription>
                {text(
                  'Daily totals for the last seven days',
                  'Jumla za kila siku kwa siku saba zilizopita'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {trend.length ? (
                <ChartContainer config={trendConfig} className='aspect-auto h-[320px] w-full'>
                  <AreaChart
                    accessibilityLayer
                    data={trend}
                    margin={{ top: 12, right: 12, left: 2, bottom: 4 }}
                  >
                    <defs>
                      <linearGradient id='sales-gradient' x1='0' y1='0' x2='0' y2='1'>
                        <stop offset='0%' stopColor='var(--color-sales)' stopOpacity={0.24} />
                        <stop offset='100%' stopColor='var(--color-sales)' stopOpacity={0.02} />
                      </linearGradient>
                      <linearGradient id='profit-gradient' x1='0' y1='0' x2='0' y2='1'>
                        <stop offset='0%' stopColor='var(--color-profit)' stopOpacity={0.18} />
                        <stop offset='100%' stopColor='var(--color-profit)' stopOpacity={0.02} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      vertical={false}
                      strokeDasharray='4 4'
                      className='stroke-border/60'
                    />
                    <XAxis
                      dataKey='label'
                      tickLine={false}
                      axisLine={false}
                      tickMargin={10}
                      tick={{ fontSize: 12 }}
                    />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      tickMargin={8}
                      width={58}
                      tick={{ fontSize: 11 }}
                      tickFormatter={(value) =>
                        money(Number(value)).replace('TZS', 'TSh').replace(/\.00$/, '')
                      }
                    />
                    <ChartTooltip
                      cursor={{ stroke: 'var(--border)', strokeDasharray: '4 4' }}
                      content={<ChartTooltipContent formatter={(value) => money(Number(value))} />}
                    />
                    <Legend
                      verticalAlign='top'
                      align='right'
                      height={32}
                      content={
                        <ChartLegendContent className='justify-end gap-4 pt-0 text-xs sm:text-sm' />
                      }
                    />
                    <Area
                      dataKey='sales'
                      name='Sales'
                      type='monotone'
                      fill='url(#sales-gradient)'
                      stroke='var(--color-sales)'
                      strokeWidth={3}
                      dot={{ r: 3, fill: 'var(--color-sales)', strokeWidth: 0 }}
                      activeDot={{ r: 5, strokeWidth: 2, stroke: 'var(--background)' }}
                    />
                    <Area
                      dataKey='profit'
                      name='Profit'
                      type='monotone'
                      fill='url(#profit-gradient)'
                      stroke='var(--color-profit)'
                      strokeWidth={3}
                      dot={{ r: 3, fill: 'var(--color-profit)', strokeWidth: 0 }}
                      activeDot={{ r: 5, strokeWidth: 2, stroke: 'var(--background)' }}
                    />
                  </AreaChart>
                </ChartContainer>
              ) : (
                <div className='text-muted-foreground flex h-[320px] items-center justify-center text-sm'>
                  {loading ? 'Loading sales trend…' : 'No sales history for the last seven days.'}
                </div>
              )}
            </CardContent>
          </Card>

          <Card className='col-span-4 md:col-span-3'>
            <CardHeader>
              <CardTitle>{text('Recent sales', 'Mauzo ya hivi karibuni')}</CardTitle>
              <CardDescription>
                {text('Latest receipts for this store', 'Risiti za mwisho za duka hili')}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className='space-y-5'>
                {transactions.length ? (
                  transactions.slice(0, 5).map((transaction) => {
                    const customer = transaction.customer_name || 'Walk-in customer';
                    const initials = customer
                      .split(/\s+/)
                      .slice(0, 2)
                      .map((part) => part[0])
                      .join('')
                      .toUpperCase();
                    return (
                      <div key={transaction.id} className='flex items-center gap-3'>
                        <Avatar className='size-9'>
                          <AvatarFallback>{initials || 'WC'}</AvatarFallback>
                        </Avatar>
                        <div className='min-w-0 space-y-1'>
                          <p className='truncate text-sm leading-none font-medium'>{customer}</p>
                          <p className='text-muted-foreground truncate text-xs'>
                            {transaction.txn_number} · {transaction.payment_method}
                          </p>
                        </div>
                        <div className='ml-auto text-right'>
                          <p className='text-sm font-medium'>{money(transaction.total)}</p>
                          <p className='text-muted-foreground text-xs'>
                            {new Date(transaction.created_at).toLocaleTimeString([], {
                              hour: '2-digit',
                              minute: '2-digit'
                            })}
                          </p>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <p className='text-muted-foreground py-8 text-center text-sm'>
                    {loading ? 'Loading sales…' : 'No sales recorded today.'}
                  </p>
                )}
              </div>
            </CardContent>
          </Card>

          <Card className='col-span-4'>
            <CardHeader>
              <CardTitle>{text('Top products', 'Bidhaa zinazoongoza')}</CardTitle>
              <CardDescription>
                {text('Sales by product today', 'Mauzo ya bidhaa leo')}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {topProducts.length ? (
                <ChartContainer
                  config={{ sales: { label: 'Sales', color: 'var(--chart-3)' } }}
                  className='aspect-auto h-[260px] w-full'
                >
                  <BarChart accessibilityLayer data={topProducts} margin={{ left: 8, right: 12 }}>
                    <CartesianGrid horizontal={false} />
                    <XAxis
                      dataKey='name'
                      tickLine={false}
                      axisLine={false}
                      tickMargin={8}
                      tickFormatter={(name) => (name.length > 10 ? `${name.slice(0, 10)}…` : name)}
                    />
                    <ChartTooltip
                      content={<ChartTooltipContent formatter={(value) => money(Number(value))} />}
                    />
                    <Bar dataKey='sales' fill='var(--color-sales)' radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ChartContainer>
              ) : (
                <div className='text-muted-foreground flex h-[260px] items-center justify-center text-sm'>
                  {loading ? 'Loading product sales…' : 'No product sales recorded today.'}
                </div>
              )}
            </CardContent>
          </Card>

          <Card className='col-span-4 flex min-h-0 flex-col md:col-span-3'>
            <CardHeader className='items-center pb-0'>
              <CardTitle>{text('Payment methods', 'Njia za malipo')}</CardTitle>
              <CardDescription>
                {text(
                  'Sales value by payment method today',
                  'Thamani ya mauzo kwa njia ya malipo leo'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='flex flex-1 items-center justify-center pb-2'>
              {paymentData.length ? (
                <ChartContainer
                  config={paymentConfig}
                  className='mx-auto aspect-square max-h-[260px] min-h-[220px] [&_.recharts-text]:fill-background'
                >
                  <PieChart>
                    <ChartTooltip
                      content={
                        <ChartTooltipContent
                          nameKey='key'
                          formatter={(value) => money(Number(value))}
                          hideLabel
                        />
                      }
                    />
                    <Pie
                      data={paymentData}
                      dataKey='amount'
                      nameKey='method'
                      innerRadius={58}
                      outerRadius={92}
                      paddingAngle={3}
                      cornerRadius={5}
                    >
                      <LabelList
                        dataKey='method'
                        stroke='none'
                        fontSize={10}
                        fontWeight={500}
                        fill='currentColor'
                        formatter={(value) => String(value ?? '')}
                      />
                    </Pie>
                  </PieChart>
                </ChartContainer>
              ) : (
                <div className='text-muted-foreground flex h-[220px] items-center justify-center text-sm'>
                  {loading ? 'Loading payment mix…' : 'No payments recorded today.'}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <div>
          <div className='mb-3 flex items-center justify-between gap-3'>
            <div>
              <h3 className='text-base font-semibold'>
                {text('Business pulse', 'Hali ya biashara')}
              </h3>
              <p className='text-muted-foreground text-sm'>
                {text(
                  "Quick signals to guide today's decisions.",
                  'Taarifa za haraka za kusaidia maamuzi ya leo.'
                )}
              </p>
            </div>
            <Badge variant='outline' className='hidden sm:inline-flex'>
              {text('Today · TZS', 'Leo · TZS')}
            </Badge>
          </div>
          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            <Card className='border-blue-500/20 bg-blue-500/[0.04]'>
              <CardHeader className='pb-2'>
                <CardDescription>
                  {text('Customers served', 'Wateja waliohudumiwa')}
                </CardDescription>
                <CardTitle className='text-2xl tabular-nums'>
                  {loading ? '—' : customerCount}
                </CardTitle>
              </CardHeader>
              <CardFooter className='text-muted-foreground flex items-center gap-2 text-xs'>
                <Icons.user className='size-3.5' /> Identified customers today
              </CardFooter>
            </Card>
            <Card className='border-violet-500/20 bg-violet-500/[0.04]'>
              <CardHeader className='pb-2'>
                <CardDescription>{text('Credit sales', 'Mauzo ya mkopo')}</CardDescription>
                <CardTitle className='truncate text-2xl tabular-nums'>
                  {loading ? '—' : money(creditSales)}
                </CardTitle>
              </CardHeader>
              <CardFooter className='text-muted-foreground flex items-center gap-2 text-xs'>
                <Icons.creditCard className='size-3.5' /> Review collections
              </CardFooter>
            </Card>
            <Card
              className={
                criticalStock
                  ? 'border-amber-500/30 bg-amber-500/[0.05]'
                  : 'border-emerald-500/20 bg-emerald-500/[0.04]'
              }
            >
              <CardHeader className='pb-2'>
                <CardDescription>{text('Inventory health', 'Hali ya stoo')}</CardDescription>
                <CardTitle className='text-2xl tabular-nums'>
                  {loading ? '—' : criticalStock ? `${criticalStock} critical` : 'Healthy'}
                </CardTitle>
              </CardHeader>
              <CardFooter className='text-muted-foreground flex items-center gap-2 text-xs'>
                <Icons.product className='size-3.5' />{' '}
                {loading ? 'Checking stock…' : `${lowStock.length} items need review`}
              </CardFooter>
            </Card>
            <Card className='border-emerald-500/20 bg-emerald-500/[0.04]'>
              <CardHeader className='pb-2'>
                <CardDescription>{text('Average ticket', 'Wastani wa mauzo')}</CardDescription>
                <CardTitle className='truncate text-2xl tabular-nums'>
                  {loading ? '—' : money(averageTicket)}
                </CardTitle>
              </CardHeader>
              <CardFooter className='text-muted-foreground flex items-center gap-2 text-xs'>
                <Icons.trendingUp className='size-3.5' /> Revenue per transaction
              </CardFooter>
            </Card>
          </div>
        </div>
        <Card className='border-sky-500/20 bg-sky-500/[0.04]'>
          <CardContent className='flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between'>
            <div className='flex min-w-0 items-center gap-3'>
              <div className='bg-sky-500/10 text-sky-600 dark:text-sky-300 grid size-10 shrink-0 place-items-center rounded-full'>
                <Icons.send className='size-5' />
              </div>
              <div className='min-w-0'>
                <p className='font-medium'>{text('SMS balance', 'Salio la SMS')}</p>
                <p className='text-muted-foreground truncate text-sm'>
                  {text(
                    'Send credit reminders and customer updates.',
                    'Tuma vikumbusho vya mikopo na taarifa kwa wateja.'
                  )}
                </p>
              </div>
            </div>
            <div className='flex items-center gap-3 sm:shrink-0'>
              <span className='text-lg font-semibold tabular-nums'>
                {loading
                  ? '—'
                  : smsBalance?.configured
                    ? `${amountOf(smsBalance.balance)} SMS`
                    : 'Not configured'}
              </span>
              <Button size='sm' variant='outline' onClick={() => router.push('/business-settings')}>
                {text('Manage', 'Dhibiti')}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  );
}
