export async function captureException(error: unknown) {
  if (process.env.NEXT_PUBLIC_SENTRY_DISABLED) return

  const Sentry = await import('@sentry/nextjs')
  Sentry.captureException(error)
}
