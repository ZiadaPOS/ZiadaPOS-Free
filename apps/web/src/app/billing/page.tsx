'use client';

import { useEffect, useState } from 'react';
import PageContainer from '@/components/layout/page-container';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { money, posApi, unwrapList } from '@/lib/pos-api';

type Plan = {
  id: string;
  name: string;
  description?: string;
  price_per_month: number;
  included_stores?: number;
};

export default function BillingPage() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [subscription, setSubscription] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    Promise.all([
      posApi<unknown>('/subscriptions/plans'),
      posApi<Record<string, unknown>>('/subscriptions/my-subscription')
    ])
      .then(([plansResult, subscriptionResult]) => {
        setPlans(unwrapList<Plan>(plansResult.data));
        setSubscription(subscriptionResult.data);
      })
      .catch((reason) =>
        setError(reason instanceof Error ? reason.message : 'Could not load subscriptions.')
      );
  }, []);
  return (
    <PageContainer
      pageTitle='Subscriptions'
      pageDescription='Manage your plan, trial, and store limits.'
    >
      <div className='space-y-5'>
        {error && (
          <p role='alert' className='text-destructive text-sm'>
            {error}
          </p>
        )}
        {subscription && (
          <Card>
            <CardHeader>
              <CardTitle>Current subscription</CardTitle>
              <CardDescription>{String(subscription.status || 'Active')}</CardDescription>
            </CardHeader>
            <CardContent className='grid gap-3 sm:grid-cols-3'>
              <div>
                <p className='text-muted-foreground text-sm'>Plan</p>
                <p className='text-xl font-semibold'>
                  {String(subscription.plan_name || subscription.plan || 'Trial')}
                </p>
              </div>
              <div>
                <p className='text-muted-foreground text-sm'>Ends</p>
                <p className='font-medium'>
                  {String(subscription.ends_at || subscription.trial_ends_at || '—')}
                </p>
              </div>
              <div>
                <p className='text-muted-foreground text-sm'>Stores</p>
                <p className='font-medium'>{String(subscription.included_stores || '—')}</p>
              </div>
            </CardContent>
          </Card>
        )}
        <div className='grid gap-4 md:grid-cols-3'>
          {plans.map((plan) => (
            <Card key={plan.id}>
              <CardHeader>
                <CardTitle>{plan.name}</CardTitle>
                <CardDescription>
                  {plan.description || 'Business tools for your store.'}
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3'>
                <p className='text-3xl font-semibold'>
                  {money(plan.price_per_month)}
                  <span className='text-muted-foreground text-sm'> / month</span>
                </p>
                <p className='text-muted-foreground text-sm'>
                  {plan.included_stores || 0} included stores
                </p>
                <Button className='w-full' disabled>
                  Choose plan
                </Button>
              </CardContent>
            </Card>
          ))}
          {!plans.length && <p className='text-muted-foreground'>No plans are available yet.</p>}
        </div>
      </div>
    </PageContainer>
  );
}
