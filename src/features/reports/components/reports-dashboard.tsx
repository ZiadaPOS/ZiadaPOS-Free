'use client';

import { useCallback, useEffect, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { money, posApi } from '@/lib/pos-api';

type ReportType = {
  id: string;
  name: string;
  desc: string;
  color?: string;
  supports_date_range?: boolean;
};
type ExportRow = {
  id: string;
  name?: string;
  report_type: string;
  period_label?: string;
  format: string;
  file_size_bytes?: number;
  created_at?: string;
};

export default function ReportsDashboard() {
  const [types, setTypes] = useState<ReportType[]>([]);
  const [exports, setExports] = useState<ExportRow[]>([]);
  const [reportType, setReportType] = useState('sales');
  const [range, setRange] = useState('30d');
  const [preview, setPreview] = useState<Record<string, any> | null>(null);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [typeResult, exportResult] = await Promise.all([
        posApi<{ report_types: ReportType[] }>('/reports/types'),
        posApi<ExportRow[]>('/reports/exports?limit=20')
      ]);
      setTypes(typeResult.data.report_types ?? []);
      setExports(Array.isArray(exportResult.data) ? exportResult.data : []);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not load reports.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function generate(format: 'json' | 'csv') {
    setGenerating(true);
    setError('');
    try {
      const result = await fetch('/api/pos/reports/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ report_type: reportType, range, format })
      });
      if (!result.ok) throw new Error('Could not generate report.');
      if (format === 'csv') {
        const blob = await result.blob();
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `${reportType}-${range}.csv`;
        anchor.click();
        URL.revokeObjectURL(url);
      } else setPreview(await result.json());
      void load();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not generate report.');
    } finally {
      setGenerating(false);
    }
  }

  async function download(row: ExportRow) {
    const response = await fetch(`/api/pos/reports/exports/${row.id}/download`);
    if (!response.ok) return;
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `${row.report_type}-${row.format}`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  const report = preview?.report as Record<string, any> | undefined;
  const kpis = report?.kpis ?? {};
  const selected = types.find((type) => type.id === reportType);

  return (
    <PageContainer
      pageTitle='Reports'
      pageDescription='Build, preview, export, or schedule business reports.'
      pageHeaderAction={
        <Button variant='outline' onClick={() => void load()} isLoading={loading}>
          <Icons.refresh className='size-4' /> Refresh
        </Button>
      }
    >
      <div className='flex min-w-0 flex-col gap-4 sm:gap-6'>
        {error && (
          <div
            role='alert'
            className='border-destructive/40 bg-destructive/5 text-destructive rounded-lg border p-3 text-sm'
          >
            {error}
          </div>
        )}
        <div className='grid gap-4 lg:grid-cols-[280px_1fr]'>
          <Card className='h-fit'>
            <CardHeader>
              <CardTitle>Choose report</CardTitle>
              <CardDescription>Select a business question to answer.</CardDescription>
            </CardHeader>
            <CardContent className='grid gap-2'>
              {(types.length
                ? types
                : [
                    {
                      id: 'sales',
                      name: 'Sales Summary',
                      desc: 'Daily sales, products, payments, and trends.'
                    }
                  ]
              ).map((type) => (
                <button
                  key={type.id}
                  type='button'
                  onClick={() => {
                    setReportType(type.id);
                    setPreview(null);
                  }}
                  className={`rounded-lg border p-3 text-left transition-colors ${reportType === type.id ? 'border-primary bg-primary/5' : 'hover:bg-muted/60'}`}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{type.name}</span>
                    {reportType === type.id && <Icons.check className='text-primary size-4' />}
                  </div>
                  <p className='text-muted-foreground mt-1 text-xs'>{type.desc}</p>
                </button>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>{selected?.name ?? 'Sales Summary'}</CardTitle>
              <CardDescription>
                {selected?.desc ?? 'Preview a server-generated report.'}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='flex flex-wrap gap-2'>
                {['7d', '30d', '90d', 'ytd'].map((item) => (
                  <Button
                    key={item}
                    size='sm'
                    variant={range === item ? 'default' : 'outline'}
                    onClick={() => setRange(item)}
                  >
                    {item.toUpperCase()}
                  </Button>
                ))}
              </div>
              <div className='flex flex-wrap gap-2'>
                <Button onClick={() => void generate('json')} isLoading={generating}>
                  <Icons.eye className='size-4' /> Preview report
                </Button>
                <Button
                  variant='outline'
                  onClick={() => void generate('csv')}
                  isLoading={generating}
                >
                  <Icons.download className='size-4' /> Download CSV
                </Button>
              </div>
              {preview && (
                <div className='space-y-4 rounded-xl border bg-muted/20 p-3 sm:p-5'>
                  <div className='flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between'>
                    <div>
                      <Badge variant='secondary'>Preview</Badge>
                      <h3 className='mt-2 text-lg font-semibold'>
                        {preview.report_name ?? selected?.name}
                      </h3>
                      <p className='text-muted-foreground text-sm'>
                        {preview.date_from} – {preview.date_to}
                      </p>
                    </div>
                    <Button size='sm' variant='outline' onClick={() => void generate('csv')}>
                      <Icons.download className='size-4' /> CSV
                    </Button>
                  </div>
                  <div className='grid grid-cols-2 gap-3 lg:grid-cols-4'>
                    {[
                      ['Revenue', kpis.revenue],
                      ['Profit', kpis.profit],
                      ['Transactions', kpis.transaction_count],
                      ['Avg. ticket', kpis.avg_ticket]
                    ].map(([label, value]) => (
                      <div className='rounded-lg border bg-card p-3' key={String(label)}>
                        <p className='text-muted-foreground text-xs'>{label}</p>
                        <p className='mt-1 truncate text-lg font-semibold'>
                          {typeof value === 'number'
                            ? String(label) === 'Transactions'
                              ? value
                              : money(value)
                            : '—'}
                        </p>
                      </div>
                    ))}
                  </div>
                  <div className='rounded-lg border bg-card p-4 text-sm'>
                    <p className='font-medium'>Business summary</p>
                    <p className='text-muted-foreground mt-2'>
                      This preview is generated from the server-side report data for the selected
                      period. The next enhancement can add AI-written findings and recommended
                      actions.
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
        <Card>
          <CardHeader>
            <CardTitle>Export history</CardTitle>
            <CardDescription>Previously generated reports for this store.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className='overflow-x-auto'>
              <table className='w-full min-w-[640px] text-sm'>
                <thead>
                  <tr className='border-b text-left text-muted-foreground'>
                    <th className='pb-2 font-medium'>Report</th>
                    <th className='pb-2 font-medium'>Period</th>
                    <th className='pb-2 font-medium'>Format</th>
                    <th className='pb-2 font-medium'>Created</th>
                    <th className='pb-2 text-right font-medium'>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {exports.map((row) => (
                    <tr className='border-b last:border-0' key={row.id}>
                      <td className='py-3 font-medium'>{row.name ?? row.report_type}</td>
                      <td className='py-3 text-muted-foreground'>{row.period_label ?? '—'}</td>
                      <td className='py-3 uppercase'>{row.format}</td>
                      <td className='py-3 text-muted-foreground'>
                        {row.created_at ? new Date(row.created_at).toLocaleDateString() : '—'}
                      </td>
                      <td className='py-3 text-right'>
                        <Button size='sm' variant='ghost' onClick={() => void download(row)}>
                          <Icons.download className='size-4' /> Download
                        </Button>
                      </td>
                    </tr>
                  ))}
                  {!exports.length && (
                    <tr>
                      <td colSpan={5} className='text-muted-foreground py-10 text-center'>
                        No reports have been exported yet.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  );
}
