'use client';

import Link from 'next/link';
import { FormEvent, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Icons } from '@/components/icons';
import { tanzaniaPhoneLocalNumber, tanzaniaPhoneNationalDigits } from '@/lib/tanzania-phone';

const steps = [
  [
    ['first_name', 'First name'],
    ['last_name', 'Last name'],
    ['phone', 'Phone number'],
    ['email', 'Email address'],
    ['password', 'Password'],
    ['confirm_password', 'Confirm password']
  ],
  [
    ['shop_name', 'Business name'],
    ['business_type', 'Business type']
  ],
  [
    ['region', 'Region'],
    ['district', 'District'],
    ['ward', 'Ward']
  ]
];

type SelectOption = { slug: string; name: string; description?: string };
type RegionOption = SelectOption & { districts: string[] };
type MetadataResponse<T> = { success: boolean; data: T };
type RegistrationProgress = {
  step: number;
  draft: Record<string, string>;
  businessType: string;
  phoneDigits: string;
  selectedRegion: string;
  verifying: boolean;
  registeredPhone: string;
};
const REGISTRATION_PROGRESS_KEY = 'pos_registration_progress';

export default function SignUpPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [registeredPhone, setRegisteredPhone] = useState('');
  const [verificationDigits, setVerificationDigits] = useState<string[]>(Array(6).fill(''));
  const verificationInputs = useRef<(HTMLInputElement | null)[]>([]);
  const verificationCode = verificationDigits.join('');
  const [resending, setResending] = useState(false);
  const [resendCountdown, setResendCountdown] = useState(15);
  const [draft, setDraft] = useState<Record<string, FormDataEntryValue>>({});
  const [businessType, setBusinessType] = useState('');
  const [businessTypeOpen, setBusinessTypeOpen] = useState(false);
  const [regionOpen, setRegionOpen] = useState(false);
  const [districtOpen, setDistrictOpen] = useState(false);
  const [phoneDigits, setPhoneDigits] = useState('');
  const [selectedRegion, setSelectedRegion] = useState('');
  const [businessTypes, setBusinessTypes] = useState<SelectOption[]>([]);
  const [outletTypes, setOutletTypes] = useState<SelectOption[]>([]);
  const [regions, setRegions] = useState<RegionOption[]>([]);
  const [metadataLoading, setMetadataLoading] = useState(true);
  const [metadataError, setMetadataError] = useState('');
  const [metadataRetry, setMetadataRetry] = useState(0);
  const [progressReady, setProgressReady] = useState(false);
  useEffect(() => {
    if (verifying) verificationInputs.current[0]?.focus();
  }, [verifying]);
  useEffect(() => {
    if (!verifying || resendCountdown <= 0) return;
    const timer = window.setTimeout(() => setResendCountdown((remaining) => remaining - 1), 1000);
    return () => window.clearTimeout(timer);
  }, [resendCountdown, verifying]);
  useEffect(() => {
    try {
      const saved = localStorage.getItem(REGISTRATION_PROGRESS_KEY);
      if (saved) {
        const progress = JSON.parse(saved) as RegistrationProgress;
        if (progress && typeof progress === 'object') {
          if (Number.isInteger(progress.step)) setStep(Math.min(2, Math.max(0, progress.step)));
          if (progress.draft && typeof progress.draft === 'object') {
            const safeDraft = Object.fromEntries(
              Object.entries(progress.draft).filter(
                ([key, value]) =>
                  key !== 'password' &&
                  key !== 'confirm_password' &&
                  key !== 'occupation' &&
                  key !== 'address' &&
                  typeof value === 'string'
              )
            );
            setDraft(safeDraft);
          }
          if (typeof progress.businessType === 'string') setBusinessType(progress.businessType);
          if (typeof progress.phoneDigits === 'string') setPhoneDigits(progress.phoneDigits);
          if (typeof progress.selectedRegion === 'string')
            setSelectedRegion(progress.selectedRegion);
          if (progress.verifying && typeof progress.registeredPhone === 'string') {
            setVerifying(true);
            setRegisteredPhone(progress.registeredPhone);
          }
        }
      }
    } catch {
      try {
        localStorage.removeItem(REGISTRATION_PROGRESS_KEY);
      } catch {
        // Storage can be blocked by the browser; registration remains usable.
      }
    } finally {
      setProgressReady(true);
    }
  }, []);
  useEffect(() => {
    if (!progressReady) return;
    const safeDraft = Object.fromEntries(
      Object.entries(draft).filter(
        ([key, value]) =>
          key !== 'password' &&
          key !== 'confirm_password' &&
          key !== 'occupation' &&
          key !== 'address' &&
          typeof value === 'string'
      )
    );
    const progress: RegistrationProgress = {
      step,
      draft: safeDraft as Record<string, string>,
      businessType,
      phoneDigits,
      selectedRegion,
      verifying,
      registeredPhone
    };
    try {
      localStorage.setItem(REGISTRATION_PROGRESS_KEY, JSON.stringify(progress));
    } catch {
      // Continue registration if browser storage is unavailable or full.
    }
  }, [
    businessType,
    draft,
    phoneDigits,
    progressReady,
    registeredPhone,
    selectedRegion,
    step,
    verifying
  ]);
  useEffect(() => {
    const controller = new AbortController();
    async function loadMetadata() {
      setMetadataLoading(true);
      setMetadataError('');
      try {
        const endpoints = [
          '/api/pos/meta/business-types',
          '/api/pos/meta/regions',
          '/api/pos/meta/outlet-types'
        ];
        const responses = await Promise.all(
          endpoints.map((endpoint) =>
            fetch(endpoint, { signal: controller.signal, cache: 'no-store' })
          )
        );
        if (responses.some((response) => !response.ok))
          throw new Error('Could not load registration options from the API.');
        const [businessBody, regionBody, outletBody] = (await Promise.all(
          responses.map((response) => response.json())
        )) as [
          MetadataResponse<SelectOption[]>,
          MetadataResponse<RegionOption[]>,
          MetadataResponse<SelectOption[]>
        ];
        if (
          !businessBody.success ||
          !regionBody.success ||
          !outletBody.success ||
          !Array.isArray(businessBody.data) ||
          !Array.isArray(regionBody.data) ||
          !Array.isArray(outletBody.data) ||
          !businessBody.data.length ||
          !regionBody.data.length ||
          !outletBody.data.length
        ) {
          throw new Error('The API returned incomplete registration options.');
        }
        setBusinessTypes(businessBody.data);
        setBusinessType((current) => current || businessBody.data[0].slug);
        setRegions(regionBody.data);
        setOutletTypes(outletBody.data);
      } catch (reason) {
        if (!controller.signal.aborted) {
          setMetadataError(
            reason instanceof Error ? reason.message : 'Could not load registration options.'
          );
        }
      } finally {
        if (!controller.signal.aborted) setMetadataLoading(false);
      }
    }
    void loadMetadata();
    return () => controller.abort();
  }, [metadataRetry]);
  function next(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    const fields = Object.fromEntries(new FormData(event.currentTarget).entries());
    setDraft((current) => ({
      ...current,
      ...fields
    }));
    if (typeof fields.business_type === 'string') setBusinessType(fields.business_type);
    if (typeof fields.region === 'string') setSelectedRegion(fields.region);
    setStep((value) => value + 1);
  }
  function back() {
    setError('');
    setStep((current) => (current === 2 ? 1 : 0));
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');
    const form = new FormData(event.currentTarget);
    const values = { ...draft, ...Object.fromEntries(form.entries()) };
    const value = (key: string) => String(values[key] || '');
    const phone = tanzaniaPhoneLocalNumber(value('phone'));
    if (!value('region')) {
      setPending(false);
      setError('Please choose a region to continue.');
      return;
    }
    if (!value('password') || !value('confirm_password')) {
      setPending(false);
      setStep(0);
      setError(
        'For your security, please re-enter your password and confirmation to finish registration.'
      );
      return;
    }
    try {
      const response = await fetch('/api/pos/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          first_name: value('first_name'),
          last_name: value('last_name'),
          phone,
          email: value('email'),
          password: value('password'),
          confirm_password: value('confirm_password'),
          shop: {
            name: value('shop_name'),
            business_type: value('business_type'),
            outlet_type: value('outlet_type'),
            region: value('region'),
            district: value('district'),
            ward: value('ward')
          }
        })
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.message || 'Could not create account.');
      setRegisteredPhone(phone);
      setResendCountdown(15);
      setVerifying(true);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not create account.');
    } finally {
      setPending(false);
    }
  }
  async function verifyRegistration(code: string) {
    if (!/^\d{6}$/.test(code) || pending) return;
    setPending(true);
    setError('');
    try {
      const response = await fetch('/api/pos-session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'verify-registration',
          phone: registeredPhone,
          code
        })
      });
      const result = await response.json();
      if (!response.ok || !result.success)
        throw new Error(result.message || 'Could not verify phone.');
      localStorage.removeItem(REGISTRATION_PROGRESS_KEY);
      const verifiedBusinessType = result.data.business_type;
      if (verifiedBusinessType) localStorage.setItem('pos_business_type', verifiedBusinessType);
      if (result.data.outlet_type) localStorage.setItem('pos_outlet_type', result.data.outlet_type);
      const destinations: Record<string, string> = {
        pharmacy: '/dashboard/pharmacy/expiry',
        retail: '/dashboard/overview',
        wholesale: '/dashboard/overview',
        retail_wholesale: '/dashboard/overview'
      };
      router.replace(destinations[verifiedBusinessType] || '/dashboard/overview');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Could not verify phone.');
    } finally {
      setPending(false);
    }
  }
  function updateVerificationCode(index: number, rawValue: string) {
    const digits = rawValue.replace(/\D/g, '');
    const next = [...verificationDigits];
    if (digits.length > 1) {
      digits
        .slice(0, 6 - index)
        .split('')
        .forEach((digit, offset) => {
          next[index + offset] = digit;
        });
    } else {
      next[index] = digits;
    }
    setVerificationDigits(next);
    if (next.every(Boolean)) {
      void verifyRegistration(next.join(''));
    } else if (digits.length > 1) {
      verificationInputs.current[Math.min(index + digits.length, 5)]?.focus();
    } else if (digits && index < 5) {
      verificationInputs.current[index + 1]?.focus();
    }
  }
  const visibleFields: string[][] =
    step === 1
      ? [...steps[1], ...(businessType === 'pharmacy' ? [['outlet_type', 'Outlet type']] : [])]
      : steps[step];
  const districtOptions = regions.find((region) => region.slug === selectedRegion)?.districts || [];
  const selectedBusinessType = businessTypes.find((option) => option.slug === businessType);
  return (
    <main className='bg-muted/30 flex min-h-svh items-center justify-center p-4 font-sans'>
      <Card className='w-full max-w-xl'>
        <CardHeader className='space-y-4'>
          <div className='flex items-center justify-center gap-3'>
            <img src='/ziada-final.jpeg' alt='' className='size-10 rounded-full object-cover' />
            <span className='text-primary font-bold tracking-wide'>ZIADA POS</span>
          </div>
          <div className='text-center'>
            <CardTitle className='text-2xl'>Create your business account</CardTitle>
            <CardDescription>Set up your owner account and store in a few steps.</CardDescription>
          </div>
          <div className='grid grid-cols-3 gap-2'>
            {['Account', 'Business', 'Location'].map((label, index) => (
              <div
                key={label}
                className={`border-t-2 pt-2 text-xs ${index <= step ? 'border-primary text-primary' : 'border-muted text-muted-foreground'}`}
              >
                {index + 1}. {label}
              </div>
            ))}
          </div>
        </CardHeader>
        <CardContent>
          {verifying && (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void verifyRegistration(verificationCode);
              }}
              className='space-y-5'
            >
              <div className='space-y-2'>
                <Label htmlFor='verification_digit_0' className='block text-center'>
                  Verification code
                </Label>
                <p className='text-muted-foreground text-center text-sm'>
                  Enter the six digit code sent to {registeredPhone}.
                </p>
                <div
                  className='flex justify-center gap-2'
                  role='group'
                  aria-label='Six digit verification code'
                >
                  {verificationDigits.map((digit, index) => (
                    <Input
                      key={index}
                      ref={(element) => {
                        verificationInputs.current[index] = element;
                      }}
                      id={`verification_digit_${index}`}
                      type='text'
                      inputMode='numeric'
                      autoComplete={index === 0 ? 'one-time-code' : 'off'}
                      aria-label={`Digit ${index + 1} of 6`}
                      value={digit}
                      onChange={(event) => updateVerificationCode(index, event.target.value)}
                      onPaste={(event) => {
                        const pasted = event.clipboardData.getData('text').replace(/\D/g, '');
                        if (!pasted) return;
                        event.preventDefault();
                        updateVerificationCode(index, pasted);
                      }}
                      onKeyDown={(event) => {
                        if (event.key === 'Backspace' && !digit && index > 0) {
                          event.preventDefault();
                          const next = [...verificationDigits];
                          next[index - 1] = '';
                          setVerificationDigits(next);
                          verificationInputs.current[index - 1]?.focus();
                        } else if (event.key === 'ArrowLeft' && index > 0) {
                          verificationInputs.current[index - 1]?.focus();
                        } else if (event.key === 'ArrowRight' && index < 5) {
                          verificationInputs.current[index + 1]?.focus();
                        }
                      }}
                      className='size-12 px-0 text-center text-lg sm:size-14'
                      disabled={pending}
                    />
                  ))}
                </div>
              </div>
              {error && (
                <p role='alert' className='text-destructive text-sm'>
                  {error}
                </p>
              )}
              <Button
                className='h-11 w-full'
                type='submit'
                isLoading={pending}
                disabled={verificationCode.length !== 6 || pending}
              >
                Verify and continue
              </Button>
              <div className='flex items-center justify-between gap-3 text-sm'>
                <Button
                  type='button'
                  variant='outline'
                  disabled={pending}
                  onClick={() => {
                    setError('');
                    setVerifying(false);
                    setStep(2);
                  }}
                >
                  Back
                </Button>
                <Button
                  type='button'
                  variant='ghost'
                  disabled={resending || resendCountdown > 0}
                  onClick={async () => {
                    setResending(true);
                    setError('');
                    try {
                      const response = await fetch('/api/pos/auth/otp/send', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ phone: registeredPhone, purpose: 'registration' })
                      });
                      const result = await response.json();
                      if (!response.ok) throw new Error(result.message || 'Could not resend code.');
                      setResendCountdown(15);
                    } catch (reason) {
                      setError(reason instanceof Error ? reason.message : 'Could not resend code.');
                    } finally {
                      setResending(false);
                    }
                  }}
                >
                  {resending
                    ? 'Sending…'
                    : resendCountdown > 0
                      ? `Resend code in ${resendCountdown}s`
                      : 'Resend code'}
                </Button>
              </div>
            </form>
          )}
          {!verifying && (
            <form
              onChange={(event) => {
                const target = event.target;
                if (target instanceof HTMLSelectElement)
                  setDraft((current) => ({ ...current, [target.name]: target.value }));
              }}
              onSubmit={step < 2 ? next : submit}
              className='space-y-5'
            >
              <div className='grid gap-4'>
                {visibleFields.map(([name, label]) => (
                  <div key={name} className='space-y-2'>
                    <Label htmlFor={name}>{label}</Label>
                    <div className='relative'>
                      {name === 'business_type' ? (
                        <>
                          <Popover open={businessTypeOpen} onOpenChange={setBusinessTypeOpen}>
                            <PopoverTrigger
                              render={
                                <Button
                                  id={name}
                                  type='button'
                                  variant='outline'
                                  role='combobox'
                                  aria-expanded={businessTypeOpen}
                                  disabled={metadataLoading || Boolean(metadataError)}
                                  className='relative h-11 w-full justify-start pr-10 font-normal [&>span]:w-full'
                                >
                                  <span className='block w-full truncate text-left'>
                                    {selectedBusinessType?.name || 'Choose a business type'}
                                  </span>
                                  <Icons.chevronsUpDown className='text-muted-foreground pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2' />
                                </Button>
                              }
                            />
                            <PopoverContent className='w-(--anchor-width) p-0'>
                              <Command>
                                <CommandInput placeholder='Search business types…' />
                                <CommandList>
                                  <CommandEmpty>No business types found.</CommandEmpty>
                                  <CommandGroup>
                                    {businessTypes.map((option) => (
                                      <CommandItem
                                        key={option.slug}
                                        value={option.name}
                                        keywords={[option.slug]}
                                        onSelect={() => {
                                          setBusinessType(option.slug);
                                          setDraft((current) => ({
                                            ...current,
                                            business_type: option.slug
                                          }));
                                          setBusinessTypeOpen(false);
                                        }}
                                      >
                                        <Icons.check
                                          className={`mr-2 size-4 ${businessType === option.slug ? 'opacity-100' : 'opacity-0'}`}
                                        />
                                        <span>{option.name}</span>
                                      </CommandItem>
                                    ))}
                                  </CommandGroup>
                                </CommandList>
                              </Command>
                            </PopoverContent>
                          </Popover>
                          <input type='hidden' name='business_type' value={businessType} />
                          {selectedBusinessType?.description && (
                            <p className='text-muted-foreground mt-2 text-xs'>
                              {selectedBusinessType.description}
                            </p>
                          )}
                        </>
                      ) : name === 'outlet_type' ? (
                        <div className='relative'>
                          <select
                            id={name}
                            name={name}
                            value={String(draft[name] || outletTypes[0]?.slug || '')}
                            required
                            disabled={metadataLoading || Boolean(metadataError)}
                            className='bg-background h-11 w-full appearance-none rounded-md border py-1 pr-10 pl-3 text-sm'
                          >
                            {outletTypes.map((outletType) => (
                              <option key={outletType.slug} value={outletType.slug}>
                                {outletType.name}
                              </option>
                            ))}
                          </select>
                          <Icons.chevronDown className='text-muted-foreground pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2' />
                        </div>
                      ) : name === 'region' ? (
                        <>
                          <Popover open={regionOpen} onOpenChange={setRegionOpen}>
                            <PopoverTrigger
                              render={
                                <Button
                                  id={name}
                                  type='button'
                                  variant='outline'
                                  role='combobox'
                                  aria-expanded={regionOpen}
                                  disabled={metadataLoading || Boolean(metadataError)}
                                  className='relative h-11 w-full justify-start pr-10 font-normal [&>span]:w-full'
                                >
                                  <span
                                    className={`block w-full truncate text-left ${draft.region ? '' : 'text-muted-foreground'}`}
                                  >
                                    {regions.find((region) => region.slug === draft.region)?.name ||
                                      'Select region'}
                                  </span>
                                  <Icons.chevronsUpDown className='text-muted-foreground pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2' />
                                </Button>
                              }
                            />
                            <PopoverContent className='w-(--anchor-width) p-0'>
                              <Command>
                                <CommandInput placeholder='Search regions…' />
                                <CommandList>
                                  <CommandEmpty>No regions found.</CommandEmpty>
                                  <CommandGroup>
                                    {regions.map((region) => (
                                      <CommandItem
                                        key={region.slug}
                                        value={region.name}
                                        keywords={[region.slug]}
                                        onSelect={() => {
                                          setDraft((current) => ({
                                            ...current,
                                            region: region.slug,
                                            district: ''
                                          }));
                                          setSelectedRegion(region.slug);
                                          setRegionOpen(false);
                                        }}
                                      >
                                        <Icons.check
                                          className={`mr-2 size-4 ${draft.region === region.slug ? 'opacity-100' : 'opacity-0'}`}
                                        />
                                        <span>{region.name}</span>
                                      </CommandItem>
                                    ))}
                                  </CommandGroup>
                                </CommandList>
                              </Command>
                            </PopoverContent>
                          </Popover>
                          <input type='hidden' name='region' value={String(draft.region || '')} />
                        </>
                      ) : name === 'district' ? (
                        <>
                          <Popover open={districtOpen} onOpenChange={setDistrictOpen}>
                            <PopoverTrigger
                              render={
                                <Button
                                  id={name}
                                  type='button'
                                  variant='outline'
                                  role='combobox'
                                  aria-expanded={districtOpen}
                                  disabled={
                                    metadataLoading || Boolean(metadataError) || !selectedRegion
                                  }
                                  className='relative h-11 w-full justify-start pr-10 font-normal [&>span]:w-full'
                                >
                                  <span
                                    className={`block w-full truncate text-left ${draft.district ? '' : 'text-muted-foreground'}`}
                                  >
                                    {String(draft.district || 'Select district (optional)')}
                                  </span>
                                  <Icons.chevronsUpDown className='text-muted-foreground pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2' />
                                </Button>
                              }
                            />
                            <PopoverContent className='w-(--anchor-width) p-0'>
                              <Command>
                                <CommandInput placeholder='Search districts…' />
                                <CommandList>
                                  <CommandEmpty>No districts found.</CommandEmpty>
                                  <CommandGroup>
                                    {districtOptions.map((district) => (
                                      <CommandItem
                                        key={district}
                                        value={district}
                                        onSelect={() => {
                                          setDraft((current) => ({ ...current, district }));
                                          setDistrictOpen(false);
                                        }}
                                      >
                                        <Icons.check
                                          className={`mr-2 size-4 ${draft.district === district ? 'opacity-100' : 'opacity-0'}`}
                                        />
                                        <span>{district}</span>
                                      </CommandItem>
                                    ))}
                                    {draft.district && (
                                      <CommandItem
                                        value='clear district'
                                        onSelect={() => {
                                          setDraft((current) => ({ ...current, district: '' }));
                                          setDistrictOpen(false);
                                        }}
                                      >
                                        Clear district
                                      </CommandItem>
                                    )}
                                  </CommandGroup>
                                </CommandList>
                              </Command>
                            </PopoverContent>
                          </Popover>
                          <input
                            type='hidden'
                            name='district'
                            value={String(draft.district || '')}
                          />
                        </>
                      ) : (
                        <Input
                          id={name}
                          name={name}
                          className={`h-11 ${name === 'phone' ? 'pl-16' : ''}`}
                          value={name === 'phone' ? phoneDigits : String(draft[name] || '')}
                          onChange={(event) => {
                            if (name === 'phone')
                              setPhoneDigits(tanzaniaPhoneNationalDigits(event.target.value));
                            else
                              setDraft((current) => ({ ...current, [name]: event.target.value }));
                          }}
                          type={
                            (name === 'password' && showPassword) ||
                            (name === 'confirm_password' && showConfirmPassword)
                              ? 'text'
                              : name.includes('password')
                                ? 'password'
                                : name === 'email'
                                  ? 'email'
                                  : name === 'phone'
                                    ? 'tel'
                                    : 'text'
                          }
                          required={!['last_name', 'email', 'district', 'ward'].includes(name)}
                          inputMode={name === 'phone' ? 'tel' : undefined}
                          maxLength={name === 'phone' ? 9 : undefined}
                          pattern={name === 'phone' ? '[0-9]{9}' : undefined}
                          placeholder={
                            name === 'phone'
                              ? '712345678'
                              : name === 'email'
                                ? '(optional) example@email.com'
                                : name === 'ward'
                                  ? 'Enter ward'
                                  : undefined
                          }
                          minLength={name.includes('password') ? 6 : undefined}
                        />
                      )}
                      {name === 'phone' && (
                        <span className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 flex h-5 -translate-y-1/2 items-center border-r border-border pr-2 text-sm'>
                          +255
                        </span>
                      )}
                      {['password', 'confirm_password'].includes(name) && (
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon'
                          className='absolute top-1/2 right-1 -translate-y-1/2'
                          onClick={() =>
                            name === 'password'
                              ? setShowPassword((value) => !value)
                              : setShowConfirmPassword((value) => !value)
                          }
                          aria-label={`${(name === 'password' ? showPassword : showConfirmPassword) ? 'Hide' : 'Show'} ${name === 'password' ? 'password' : 'confirm password'}`}
                        >
                          {(name === 'password' ? showPassword : showConfirmPassword) ? (
                            <Icons.eyeOff className='size-4' />
                          ) : (
                            <Icons.eye className='size-4' />
                          )}
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
              {error && (
                <p role='alert' className='text-destructive text-sm'>
                  {error}
                </p>
              )}
              {metadataLoading && (
                <p className='text-muted-foreground text-sm' role='status'>
                  Loading registration options…
                </p>
              )}
              {metadataError && (
                <div className='flex flex-wrap items-center justify-between gap-3' role='alert'>
                  <p className='text-destructive text-sm'>{metadataError}</p>
                  <Button
                    type='button'
                    variant='outline'
                    onClick={() => setMetadataRetry((n) => n + 1)}
                  >
                    Retry
                  </Button>
                </div>
              )}
              {step === 0 ? (
                <div className='flex flex-col gap-3'>
                  <Button
                    className='h-11 w-full'
                    type='submit'
                    isLoading={pending}
                    disabled={metadataLoading || Boolean(metadataError)}
                  >
                    Continue
                  </Button>
                  <Link
                    href='/auth/sign-in'
                    className='text-muted-foreground self-center text-sm hover:underline'
                  >
                    Already have an account? Sign in
                  </Link>
                </div>
              ) : (
                <div className='flex flex-col-reverse gap-2 sm:flex-row sm:justify-between'>
                  <Button type='button' variant='outline' onClick={back}>
                    Back
                  </Button>
                  <Button
                    className='h-11 sm:min-w-40'
                    type='submit'
                    isLoading={pending}
                    disabled={metadataLoading || Boolean(metadataError)}
                  >
                    {step < 2 ? 'Continue' : 'Create account'}
                  </Button>
                </div>
              )}
            </form>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
