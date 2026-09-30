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
  const isFormData = typeof FormData !== 'undefined' && init.body instanceof FormData;
  const response = await fetch(`/api/pos${path.startsWith('/') ? path : `/${path}`}`, {
    ...init,
    headers: { ...(isFormData ? {} : { 'Content-Type': 'application/json' }), ...init.headers }
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
  unit?: string;
  sku: string;
  barcode: string;
  selling_price: number;
  wholesale_price?: number | null;
  wholesale_minimum?: number;
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
  next_expiry_date?: string;
  sellable_stock?: number;
  expired_stock?: number;
  untracked_stock?: number;
  requires_prescription?: boolean;
  medicine_profile_configured?: boolean;
  medicine_schedule?: string;
  generic_name?: string;
  brand_name?: string;
};
export type PosMedicineBatch = {
  id: string;
  product_id: string;
  source_batch_id?: string;
  product_name?: string;
  batch_number: string;
  quantity: number;
  expiry_date: string;
  cost_price: number;
  supplier_name?: string;
  stock_status: 'available' | 'quarantined' | string;
};
export type PosMedicineStockMovement = {
  id: string;
  movement_type: string;
  quantity: number;
  reference_id?: string;
  reason?: string;
  performed_by: string;
  created_at: string;
};
export type PosMedicineProfile = {
  product_id: string;
  generic_name: string;
  brand_name: string;
  unit: string;
  pack_size: string;
  requires_prescription: boolean;
  medicine_schedule: string;
  registration_number: string;
};
export type PosPrescriptionDetails = {
  prescriber_name: string;
  prescriber_address: string;
  prescriber_registration_number: string;
  prescriber_authenticity_confirmed: boolean;
  prescribing_facility_name: string;
  prescribing_facility_address: string;
  prescription_reference: string;
  patient_name: string;
  patient_address: string;
  medicine_strength: string;
  dosage_form: string;
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
export type PosCategory = {
  id: string;
  name: string;
  description?: string;
  sort_order?: number;
  product_count?: number;
  is_default?: boolean;
};
export type PosUnit = {
  id: string;
  name: string;
  description?: string;
  is_active?: boolean;
  is_default?: boolean;
  usage_count?: number;
};
export type PosExpenseCategory = {
  id: string;
  name: string;
  is_default?: boolean;
  expense_count?: number;
};
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
      'units',
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
