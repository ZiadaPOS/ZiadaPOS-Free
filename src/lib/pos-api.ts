export type ApiEnvelope<T> = {
  success: boolean;
  message: string;
  data?: T;
  errors?: unknown;
  meta?: {
    total_count: number;
    current_page: number;
    page_size: number;
    total_pages: number;
    has_next: boolean;
    has_prev: boolean;
  };
};

export type ApiResult<T> = ApiEnvelope<T> & { data: T };

export async function posApi<T>(path: string, init: RequestInit = {}): Promise<ApiResult<T>> {
  const response = await fetch(`/api/pos${path.startsWith('/') ? path : `/${path}`}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init.headers }
  });
  if (response.status === 204) return { success: true, message: 'OK', data: undefined as T };
  const body = (await response.json().catch(() => ({}))) as ApiEnvelope<T>;
  if (!response.ok || !body.success)
    throw new Error(body.message || `Request failed (${response.status})`);
  return body as ApiResult<T>;
}

export type PosUser = {
  id: string;
  full_name: string;
  first_name: string;
  last_name: string;
  phone: string;
  email?: string;
  role: 'admin' | 'owner' | 'staff';
  organisation_id?: string;
  store_id?: string;
};
export type PosProduct = {
  id: string;
  name: string;
  sku: string;
  barcode: string;
  selling_price: number;
  cost_price: number;
  stock: number;
  reorder_point: number;
  low_stock?: boolean;
  category_id?: string;
  category?: { id: string; name: string };
  image?: string;
  description?: string;
  is_service: boolean;
  is_active: boolean;
  tax_rate: number;
};
export type PosCustomer = {
  id: string;
  name: string;
  phone?: string;
  email?: string;
  segment: string;
  total_spent: number;
  open_credit: number;
  credit_limit?: number;
  transaction_count?: number;
};
export type PosTransaction = {
  id: string;
  txn_number: string;
  payment_method: string;
  status: string;
  subtotal: number;
  discount_amount: number;
  tax_amount: number;
  total: number;
  profit: number;
  customer_id?: string;
  customer_name: string;
  customer_phone?: string;
  created_at: string;
  lines?: {
    id: string;
    product_id?: string;
    product_name: string;
    qty: number;
    unit_price: number;
    total: number;
  }[];
};
export type PosCategory = { id: string; name: string; description?: string; sort_order?: number };
export type PosCreditTab = {
  id: string;
  customer_id: string;
  amount: number;
  amount_paid: number;
  balance: number;
  status: string;
  due_date?: string;
  is_overdue: boolean;
  created_at: string;
};

export function unwrapList<T>(data: unknown): T[] {
  if (Array.isArray(data)) return data as T[];
  if (data && typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    for (const key of [
      'items',
      'products',
      'customers',
      'transactions',
      'categories',
      'suppliers',
      'expenses',
      'staff',
      'stores',
      'tabs',
      'deliveries',
      'payments',
      'notes',
      'notifications',
      'exports',
      'reports'
    ]) {
      if (Array.isArray(obj[key])) return obj[key] as T[];
    }
  }
  return [];
}

export const money = (amount: number | string | null | undefined) =>
  new Intl.NumberFormat('en-TZ', {
    style: 'currency',
    currency: 'TZS',
    maximumFractionDigits: 0
  }).format(Number(amount) || 0);
