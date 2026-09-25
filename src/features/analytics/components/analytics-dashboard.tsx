'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from 'recharts';
import PageContainer from '@/components/layout/page-container';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { money, posApi } from '@/lib/pos-api';
import { useText } from '@/lib/i18n';

type Range = '7d' | '30d' | '90d' | 'ytd';
type AnyRecord = Record<string, any>;

const colors = ['#2563eb', '#16a34a', '#f59e0b', '#9333ea', '#ef4444'];

function arr(value: unknown, keys: string[] = []): AnyRecord[] {
  if (Array.isArray(value)) return value as AnyRecord[];
  if (!value || typeof value !== 'object') return [];
  const record = value as AnyRecord;
  for (const key of keys) if (Array.isArray(record[key])) return record[key];
  return [];
}

function num(value: unknown): number {
  return Number(value) || 0;
}

function percent(value: unknown): string {
  return `${num(value).toFixed(1)}%`;
}

function ChartFrame({ children, height = 280 }: { children: React.ReactNode; height?: number }) {
  return (
    <div className='h-[220px] w-full sm:h-[280px]' style={{ minHeight: height }}>
      {children}
    </div>
  );
}

function EmptyChart({ label }: { label: string }) {
  return (
    <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
      {label}
    </div>
  );
}

function Metric({
  label,
  value,
  detail,
  icon: Icon
}: {
  label: string;
  value: string;
  detail?: string;
  icon: React.ElementType;
}) {
  return (
    <Card className='min-w-0 bg-gradient-to-br from-primary/5 to-card shadow-xs'>
      <CardHeader className='gap-2'>
        <div className='flex items-center justify-between gap-2'>
          <CardDescription className='truncate'>{label}</CardDescription>
          <Icon className='text-muted-foreground size-4 shrink-0' />
        </div>
        <CardTitle className='truncate text-xl tabular-nums sm:text-2xl'>{value}</CardTitle>
        {detail && <CardDescription className='truncate text-xs'>{detail}</CardDescription>}
      </CardHeader>
    </Card>
  );
}

function SectionCard({
  title,
  description,
  children,
  className = ''
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

export default function AnalyticsDashboard() {
  const text = useText();
  const [range, setRange] = useState<Range>('30d');
  const [tab, setTab] = useState<'sales' | 'products' | 'customers' | 'cashflow'>('sales');
  const [data, setData] = useState<AnyRecord>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const query = `?range=${range}`;
      const result = await posApi<AnyRecord>(`/analytics/${tab}${query}`);
      setData(result.data ?? {});
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load analytics.');
      setData({});
    } finally {
      setLoading(false);
    }
  }, [range, tab]);

  useEffect(() => {
    void load();
  }, [load]);

  const kpis = data.kpis ?? {};
  const trend = arr(data.trend, ['daily_trend']);
  const products = arr(data.products, ['top_products']);
  const payments = arr(data.payment_mix, ['payment_inflow']);
  const categories = arr(data.category_breakdown);
  const hourly = arr(data.hourly_pattern);
  const dailyVisits = arr(data.daily_visits);
  const segments = arr(data.segments);
  const cashflow = arr(data.daily);
  const expenses = arr(data.expense_breakdown);
  const totals = data.totals ?? {};

  const metricCards = useMemo(() => {
    if (tab === 'products')
      return [
        [
          text('Revenue', 'Mapato'),
          money(totals.revenue),
          text('Selected period', 'Kipindi kilichochaguliwa'),
          Icons.trendingUp
        ],
        [
          text('Profit', 'Faida'),
          money(totals.profit),
          `${percent(totals.margin_pct)} ${text('margin', 'faida')}`,
          Icons.billing
        ],
        ['Units sold', String(num(totals.units_sold)), 'All tracked products', Icons.product],
        ['Products', String(num(totals.sku_count)), 'With sales in range', Icons.product]
      ];
    if (tab === 'customers')
      return [
        ['Visits', String(num(kpis.total_visits)), 'Total customer visits', Icons.user],
        ['New customers', String(num(kpis.new_customers)), 'First-time buyers', Icons.user],
        [
          'Returning',
          String(num(kpis.returning)),
          `${percent(kpis.retention_rate_pct)} retention`,
          Icons.teams
        ],
        ['Visit value', money(kpis.avg_visit_value), 'Average spend per visit', Icons.billing]
      ];
    if (tab === 'cashflow')
      return [
        ['Inflow', money(totals.inflow), 'Sales and credit inflow', Icons.trendingUp],
        ['COGS', money(totals.cogs), 'Cost of goods', Icons.product],
        ['Operating expenses', money(totals.opex), 'Recorded expenses', Icons.post],
        [
          'Net cashflow',
          money(totals.net),
          `${percent(totals.net_margin_pct)} net margin`,
          Icons.billing
        ]
      ];
    return [
      [
        'Revenue',
        money(kpis.revenue),
        `${percent(kpis.revenue_delta_pct)} vs previous`,
        Icons.trendingUp
      ],
      [
        text('Profit', 'Faida'),
        money(kpis.profit),
        `${percent(kpis.margin_pct)} ${text('margin', 'faida')}`,
        Icons.billing
      ],
      [
        'Transactions',
        String(num(kpis.transaction_count)),
        `${percent(kpis.transaction_delta_pct)} vs previous`,
        Icons.post
      ],
      [
        'Average ticket',
        money(kpis.avg_ticket),
        `${num(kpis.customer_count)} customers`,
        Icons.user
      ]
    ];
  }, [kpis, tab, totals]);

  return (
    <PageContainer
      pageTitle={text('Analytics', 'Uchambuzi')}
      pageDescription={text(
        'Understand what is driving sales, profit, customers, and cashflow.',
        'Elewa kinachoendesha mauzo, faida, wateja na mtiririko wa fedha.'
      )}
      pageHeaderAction={
        <Button variant='outline' onClick={() => void load()} isLoading={loading}>
          <Icons.refresh className='size-4' /> Refresh
        </Button>
      }
    >
      <div className='flex min-w-0 flex-1 flex-col gap-4 sm:gap-6'>
        <div className='flex flex-col gap-3 rounded-xl border bg-card p-3 sm:flex-row sm:items-center sm:justify-between sm:p-4'>
          <div className='flex min-w-0 items-center gap-2 overflow-x-auto pb-1'>
            {(['sales', 'products', 'customers', 'cashflow'] as const).map((item) => (
              <Button
                key={item}
                size='sm'
                variant={tab === item ? 'default' : 'ghost'}
                className='shrink-0 capitalize'
                onClick={() => setTab(item)}
              >
                {item === 'cashflow'
                  ? text('Cashflow & expenses', 'Mtiririko wa fedha na gharama')
                  : item === 'sales'
                    ? text('Sales', 'Mauzo')
                    : item === 'products'
                      ? text('Products', 'Bidhaa')
                      : text('Customers', 'Wateja')}
              </Button>
            ))}
          </div>
          <div className='flex items-center gap-2 overflow-x-auto'>
            {(['7d', '30d', '90d', 'ytd'] as const).map((item) => (
              <Button
                key={item}
                size='sm'
                variant={range === item ? 'secondary' : 'ghost'}
                className='shrink-0'
                onClick={() => setRange(item)}
              >
                {item.toUpperCase()}
              </Button>
            ))}
          </div>
        </div>

        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-lg border p-3 text-sm'
          >
            {error}
          </div>
        )}

        <div className='grid grid-cols-2 gap-3 lg:grid-cols-4 sm:gap-4'>
          {metricCards.map(([label, value, detail, Icon]) => (
            <Metric
              key={String(label)}
              label={String(label)}
              value={String(value)}
              detail={String(detail)}
              icon={Icon as React.ElementType}
            />
          ))}
        </div>

        {loading ? (
          <div className='text-muted-foreground flex min-h-64 items-center justify-center rounded-xl border border-dashed text-sm'>
            Loading server-side analytics…
          </div>
        ) : (
          <>
            {tab === 'sales' && (
              <div className='grid min-w-0 gap-4 lg:grid-cols-7'>
                <SectionCard
                  title='Revenue versus profit'
                  description='Daily performance for the selected period'
                  className='lg:col-span-4'
                >
                  <ChartFrame>
                    {trend.length ? (
                      <ResponsiveContainer width='100%' height='100%'>
                        <AreaChart data={trend}>
                          <CartesianGrid vertical={false} strokeDasharray='3 3' />
                          <XAxis dataKey='label' tickLine={false} axisLine={false} />
                          <YAxis
                            tickLine={false}
                            axisLine={false}
                            tickFormatter={(value) => `${Math.round(num(value) / 1000)}k`}
                          />
                          <Tooltip formatter={(value) => money(num(value))} />
                          <Area
                            dataKey='revenue'
                            name='Revenue'
                            type='monotone'
                            fill='#2563eb'
                            fillOpacity={0.14}
                            stroke='#2563eb'
                            strokeWidth={2}
                          />
                          <Area
                            dataKey='profit'
                            name='Profit'
                            type='monotone'
                            fill='#16a34a'
                            fillOpacity={0.1}
                            stroke='#16a34a'
                            strokeWidth={2}
                          />
                        </AreaChart>
                      </ResponsiveContainer>
                    ) : (
                      <EmptyChart label='No revenue history for this period.' />
                    )}
                  </ChartFrame>
                </SectionCard>
                <SectionCard
                  title='Payment mix'
                  description='Revenue by payment method'
                  className='lg:col-span-3'
                >
                  <div className='grid items-center gap-3 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2'>
                    {payments.length ? (
                      <>
                        <div className='h-[190px]'>
                          <ResponsiveContainer width='100%' height='100%'>
                            <PieChart>
                              <Pie
                                data={payments}
                                dataKey='amount'
                                nameKey='method'
                                innerRadius={52}
                                outerRadius={78}
                                paddingAngle={3}
                              >
                                {payments.map((entry, index) => (
                                  <Cell
                                    key={entry.method ?? index}
                                    fill={colors[index % colors.length]}
                                  />
                                ))}
                              </Pie>
                              <Tooltip formatter={(value) => money(num(value))} />
                            </PieChart>
                          </ResponsiveContainer>
                        </div>
                        <div className='space-y-2'>
                          {payments.map((payment, index) => (
                            <div
                              className='flex items-center justify-between gap-2 text-sm'
                              key={payment.method}
                            >
                              <span className='flex min-w-0 items-center gap-2'>
                                <i
                                  className='size-2.5 shrink-0 rounded-full'
                                  style={{ backgroundColor: colors[index % colors.length] }}
                                />{' '}
                                <span className='truncate'>{payment.method}</span>
                              </span>
                              <span className='font-medium tabular-nums'>
                                {percent(payment.pct)}
                              </span>
                            </div>
                          ))}
                        </div>
                      </>
                    ) : (
                      <EmptyChart label='No payment data for this period.' />
                    )}
                  </div>
                </SectionCard>
                <SectionCard
                  title='Category performance'
                  description='Revenue share and prior-period movement'
                  className='lg:col-span-4'
                >
                  <ChartFrame>
                    {categories.length ? (
                      <ResponsiveContainer width='100%' height='100%'>
                        <BarChart
                          data={categories}
                          layout='vertical'
                          margin={{ left: 12, right: 12 }}
                        >
                          <CartesianGrid horizontal={false} />
                          <XAxis type='number' hide />
                          <YAxis
                            type='category'
                            dataKey='name'
                            width={92}
                            tickLine={false}
                            axisLine={false}
                          />
                          <Tooltip formatter={(value) => money(num(value))} />
                          <Bar dataKey='revenue' fill='#2563eb' radius={[0, 5, 5, 0]} />
                        </BarChart>
                      </ResponsiveContainer>
                    ) : (
                      <EmptyChart label='No category data for this period.' />
                    )}
                  </ChartFrame>
                </SectionCard>
                <SectionCard
                  title='Busiest hours'
                  description='Average revenue by hour'
                  className='lg:col-span-3'
                >
                  <ChartFrame>
                    {hourly.length ? (
                      <ResponsiveContainer width='100%' height='100%'>
                        <BarChart data={hourly}>
                          <CartesianGrid vertical={false} />
                          <XAxis
                            dataKey='label'
                            tickLine={false}
                            axisLine={false}
                            interval='preserveStartEnd'
                          />
                          <YAxis hide />
                          <Tooltip formatter={(value) => money(num(value))} />
                          <Bar
                            dataKey='avg_revenue'
                            name='Average revenue'
                            fill='#f59e0b'
                            radius={[5, 5, 0, 0]}
                          />
                        </BarChart>
                      </ResponsiveContainer>
                    ) : (
                      <EmptyChart label='No hourly data for this period.' />
                    )}
                  </ChartFrame>
                </SectionCard>
              </div>
            )}

            {tab === 'products' && (
              <div className='grid min-w-0 gap-4 lg:grid-cols-2'>
                <SectionCard
                  title='Top products by revenue'
                  description='Best sellers in the selected period'
                >
                  <ChartFrame>
                    {products.length ? (
                      <ResponsiveContainer width='100%' height='100%'>
                        <BarChart
                          data={products.slice(0, 10)}
                          layout='vertical'
                          margin={{ left: 15, right: 15 }}
                        >
                          <CartesianGrid horizontal={false} />
                          <XAxis type='number' hide />
                          <YAxis
                            type='category'
                            dataKey='product_name'
                            width={110}
                            tickLine={false}
                            axisLine={false}
                            tickFormatter={(value) =>
                              String(value).length > 16 ? `${String(value).slice(0, 16)}…` : value
                            }
                          />
                          <Tooltip formatter={(value) => money(num(value))} />
                          <Bar dataKey='revenue' fill='#2563eb' radius={[0, 5, 5, 0]} />
                        </BarChart>
                      </ResponsiveContainer>
                    ) : (
                      <EmptyChart label='No product sales for this period.' />
                    )}
                  </ChartFrame>
                </SectionCard>
                <SectionCard title='Product performance' description='Revenue, profit, and margin'>
                  <div className='overflow-x-auto'>
                    <table className='w-full min-w-[520px] text-sm'>
                      <thead>
                        <tr className='border-b text-left text-muted-foreground'>
                          <th className='pb-2 font-medium'>Product</th>
                          <th className='pb-2 text-right font-medium'>Units</th>
                          <th className='pb-2 text-right font-medium'>Revenue</th>
                          <th className='pb-2 text-right font-medium'>Margin</th>
                        </tr>
                      </thead>
                      <tbody>
                        {products.slice(0, 12).map((product) => (
                          <tr
                            className='border-b last:border-0'
                            key={product.product_id ?? product.product_name}
                          >
                            <td className='max-w-[190px] truncate py-3 font-medium'>
                              {product.product_name}
                            </td>
                            <td className='py-3 text-right tabular-nums'>
                              {num(product.units_sold ?? product.qty_sold)}
                            </td>
                            <td className='py-3 text-right tabular-nums'>
                              {money(product.revenue)}
                            </td>
                            <td className='py-3 text-right tabular-nums'>
                              {percent(
                                product.margin_pct ??
                                  (num(product.profit) / num(product.revenue)) * 100
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </SectionCard>
              </div>
            )}

            {tab === 'customers' && (
              <div className='grid min-w-0 gap-4 lg:grid-cols-7'>
                <SectionCard
                  title='New versus returning visits'
                  description='Daily customer activity'
                  className='lg:col-span-4'
                >
                  <ChartFrame>
                    {dailyVisits.length ? (
                      <ResponsiveContainer width='100%' height='100%'>
                        <BarChart data={dailyVisits}>
                          <CartesianGrid vertical={false} />
                          <XAxis dataKey='label' tickLine={false} axisLine={false} />
                          <YAxis hide />
                          <Tooltip />
                          <Bar dataKey='new_customers' name='New' stackId='visits' fill='#2563eb' />
                          <Bar
                            dataKey='returning'
                            name='Returning'
                            stackId='visits'
                            fill='#16a34a'
                            radius={[5, 5, 0, 0]}
                          />
                        </BarChart>
                      </ResponsiveContainer>
                    ) : (
                      <EmptyChart label='No customer activity for this period.' />
                    )}
                  </ChartFrame>
                </SectionCard>
                <SectionCard
                  title='Customer segments'
                  description='Customers and their spend'
                  className='lg:col-span-3'
                >
                  <div className='space-y-3'>
                    {segments.length ? (
                      segments.map((segment, index) => (
                        <div key={segment.segment} className='space-y-1'>
                          <div className='flex justify-between gap-2 text-sm'>
                            <span className='truncate'>{segment.segment}</span>
                            <span className='tabular-nums'>{percent(segment.pct)}</span>
                          </div>
                          <div className='bg-muted h-2 rounded-full'>
                            <div
                              className='h-2 rounded-full'
                              style={{
                                width: `${Math.min(100, num(segment.pct))}%`,
                                backgroundColor: colors[index % colors.length]
                              }}
                            />
                          </div>
                          <div className='text-muted-foreground text-xs'>
                            {num(segment.count)} customers · {money(segment.spend)}
                          </div>
                        </div>
                      ))
                    ) : (
                      <EmptyChart label='No customer segments for this period.' />
                    )}
                  </div>
                </SectionCard>
              </div>
            )}

            {tab === 'cashflow' && (
              <div className='grid min-w-0 gap-4 lg:grid-cols-7'>
                <SectionCard
                  title='Cashflow movement'
                  description='Inflow, cost of goods, expenses, and net'
                  className='lg:col-span-4'
                >
                  <ChartFrame>
                    {cashflow.length ? (
                      <ResponsiveContainer width='100%' height='100%'>
                        <AreaChart data={cashflow}>
                          <CartesianGrid vertical={false} strokeDasharray='3 3' />
                          <XAxis dataKey='label' tickLine={false} axisLine={false} />
                          <YAxis hide />
                          <Tooltip formatter={(value) => money(num(value))} />
                          <Area
                            dataKey='inflow'
                            name='Inflow'
                            type='monotone'
                            fill='#2563eb'
                            fillOpacity={0.12}
                            stroke='#2563eb'
                          />
                          <Area
                            dataKey='net'
                            name='Net'
                            type='monotone'
                            fill='#16a34a'
                            fillOpacity={0.1}
                            stroke='#16a34a'
                          />
                          <Area
                            dataKey='opex'
                            name='Expenses'
                            type='monotone'
                            fill='#f59e0b'
                            fillOpacity={0.1}
                            stroke='#f59e0b'
                          />
                        </AreaChart>
                      </ResponsiveContainer>
                    ) : (
                      <EmptyChart label='No cashflow data for this period.' />
                    )}
                  </ChartFrame>
                </SectionCard>
                <SectionCard
                  title='Expense breakdown'
                  description='Where operating money is going'
                  className='lg:col-span-3'
                >
                  <div className='space-y-3'>
                    {expenses.length ? (
                      expenses.map((expense, index) => (
                        <div
                          key={expense.name}
                          className='flex items-center justify-between gap-3 rounded-lg border p-3'
                        >
                          <span className='flex min-w-0 items-center gap-2'>
                            <i
                              className='size-2.5 shrink-0 rounded-full'
                              style={{ backgroundColor: colors[index % colors.length] }}
                            />
                            <span className='truncate text-sm'>{expense.name}</span>
                          </span>
                          <span className='text-right text-sm font-medium tabular-nums'>
                            {money(expense.amount)}
                            <span className='text-muted-foreground ml-2 text-xs'>
                              {percent(expense.pct)}
                            </span>
                          </span>
                        </div>
                      ))
                    ) : (
                      <EmptyChart label='No expenses for this period.' />
                    )}
                  </div>
                </SectionCard>
              </div>
            )}
          </>
        )}
        <div className='text-muted-foreground flex items-center gap-2 text-xs'>
          <Icons.info className='size-3.5' /> Metrics are calculated server-side for the selected
          date range.
        </div>
      </div>
    </PageContainer>
  );
}
