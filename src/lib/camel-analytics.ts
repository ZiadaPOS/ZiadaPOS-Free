type CamelAnalyticsClient = {
  identify: (accountId: string) => void;
  clearIdentity: () => void;
};

declare global {
  interface Window {
    camel?: CamelAnalyticsClient;
  }
}

let currentIdentity: string | null | undefined;

function applyIdentity(identity: string | null) {
  if (identity) window.camel?.identify(identity);
  else window.camel?.clearIdentity();
}

export function identifyCamelUser(accountId: string | number) {
  const identity = String(accountId).trim();
  if (!identity) return;

  currentIdentity = identity;
  applyIdentity(identity);
}

export function clearCamelIdentity() {
  currentIdentity = null;
  applyIdentity(null);
}

export function syncCamelIdentity() {
  if (currentIdentity !== undefined) {
    applyIdentity(currentIdentity);
    return;
  }

  try {
    const user = JSON.parse(localStorage.getItem('pos_user') || 'null') as { id?: unknown } | null;
    if (user?.id !== undefined && user.id !== null && String(user.id).trim()) {
      currentIdentity = String(user.id);
      applyIdentity(currentIdentity);
    }
  } catch {
    // Ignore malformed local session data; authentication remains authoritative.
  }
}
