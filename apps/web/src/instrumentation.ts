import type * as Sentry from '@sentry/nextjs'

const sentryOptions: Sentry.NodeOptions | Sentry.EdgeOptions = {
  // Sentry DSN
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,

  // Enable Spotlight in development
  spotlight: process.env.NODE_ENV === 'development',

  // Adds request headers and IP for users, for more info visit
  sendDefaultPii: true,

  // Adjust this value in production, or use tracesSampler for greater control
  tracesSampleRate: 1,

  // Setting this option to true will print useful information to the console while you're setting up Sentry.
  debug: false
}

type CaptureRequestError = typeof import('@sentry/nextjs').captureRequestError

export async function register() {
  if (process.env.NEXT_PUBLIC_SENTRY_DISABLED) return

  const Sentry = await import('@sentry/nextjs')
  if (process.env.NEXT_RUNTIME === 'nodejs' || process.env.NEXT_RUNTIME === 'edge') {
    Sentry.init(sentryOptions)
  }
}

export async function onRequestError(...args: Parameters<CaptureRequestError>) {
  if (process.env.NEXT_PUBLIC_SENTRY_DISABLED) return

  const Sentry = await import('@sentry/nextjs')
  return Sentry.captureRequestError(...args)
}
