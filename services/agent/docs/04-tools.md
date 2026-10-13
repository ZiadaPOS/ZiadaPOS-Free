# 04 — Tools

The agent registers **110 domain tools + the `ask` agent tool = 111 MCP tools**.

All tools are `async`, use the `ZiadaClient` (see [03 — Client & Auth](03-client-and-auth.md)),
return the API's `data` payload, and — except where noted — operate on the **active store** of the
logged-in service account.

> The `ask` agent knows all of these tools and calls them automatically. You can also call any of
> them directly from your MCP client. Prices are **TZS**, quantities are integers, dates are
> `YYYY-MM-DD`.

---

## Inventory & products — `tools/inventory.py` (13)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_categories` | — | Product categories (shared across the business). | `GET /inventory/categories` |
| `create_category` | `name`, `description=""` | Create a category. | `POST /inventory/categories` |
| `update_category` | `category_id`, `name?`, `description?`, `sort_order?`, `is_active?` | Partial update (only provided fields). | `PATCH /inventory/categories/{id}` |
| `delete_category` | `category_id` | Delete a category (products are kept, they just lose the link). ⚠️ destructive. | `DELETE /inventory/categories/{id}` |
| `list_products` | `search?`, `category_id?`, `status?`, `low_stock?`, `page=1`, `page_size=20` | List products with filters and pagination. | `GET /inventory/products` |
| `create_product` | `name`, `selling_price`, `cost_price=0`, `category_id?`, `sku?`, `barcode?`, `stock=0`, `reorder_point=5`, `image?`, `tax_rate=18`, `description?`, `is_service=False`, `is_active=True` | Create a product. `stock` is the opening quantity. | `POST /inventory/products` |
| `get_product` | `product_id` | Fetch one product by UUID. | `GET /inventory/products/{id}` |
| `get_product_by_barcode` | `barcode` | Find a product by its barcode (scanner / phone lookup). | `GET /inventory/products/barcode/{code}` |
| `update_product` | `product_id` + any product field | Partial update. **Stock is not editable here** — use `adjust_stock`. | `PATCH /inventory/products/{id}` |
| `delete_product` | `product_id` | Soft-delete (deactivate) a product. ⚠️ destructive. | `DELETE /inventory/products/{id}` |
| `list_low_stock` | — | Products at or below their reorder point. | `GET /inventory/low-stock` |
| `adjust_stock` | `product_id`, `qty_change`, `reason`, `note?` | Adjust stock (positive = add, negative = remove). Reasons: restock / damage / correction / return. | `POST /inventory/products/{id}/adjust` |
| `list_stock_adjustments` | `product_id`, `limit=50` | Adjustment history for one product. | `GET /inventory/products/{id}/adjustments` |

---

## Sales / POS transactions — `tools/transactions.py` (7)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `complete_sale` | `lines` (`[{product_id, qty}]`), `payment_method` (`Cash|M-Pesa|Tigo Pesa|Bank|Credit`), `payment_reference?`, `discount_pct=0`, `customer_name?`, `customer_phone?`, `customer_id?`, `till_number?`, `notes?` | Complete a POS sale; decrements stock; `Credit` opens/extends a customer tab. | `POST /pos/sale` |
| `list_transactions` | `status?` (`paid|credit|refunded|void`), `method?`, `search?`, `customer_id?`, `date_from?`, `date_to?`, `page`, `page_size` | List sales with filters. | `GET /pos/transactions` |
| `get_transaction` | `transaction_id` | One sale including its itemised lines. | `GET /pos/transactions/{id}` |
| `refund_transaction` | `transaction_id`, `reason?` | Refund a paid sale — restores stock, marks transaction refunded. ⚠️ destructive. | `POST /pos/transactions/{id}/refund` |
| `void_transaction` | `transaction_id`, `reason?` | Void a transaction. ⚠️ destructive. | `POST /pos/transactions/{id}/void` |
| `sales_summary` | `date_from?`, `date_to?` (default: today) | Revenue / transaction count / profit summary for the range. | `GET /pos/summary` |
| `hourly_sales` | `date?` | Hourly revenue breakdown for a day. | `GET /pos/summary/hourly` |

---

## Customers — `tools/customers.py` (5)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_customers` | `search?`, `segment?` (`New|Regular|VIP|Occasional`), `has_credit?`, `page`, `page_size` | List customers with filters. | `GET /customers` |
| `create_customer` | `name`, `phone?`, `email?`, `segment?`, `credit_limit?`, `notes?` | Register a customer (phone is needed for SMS reminders). | `POST /customers` |
| `get_customer` | `customer_id` | Full customer profile. | `GET /customers/{id}` |
| `update_customer` | `customer_id`, `name?`, `phone?`, `email?`, `segment?`, `credit_limit?` (pass `-1` to clear), `notes?`, `is_active?` | Partial update. | `PATCH /customers/{id}` |
| `credit_customers` | `overdue?` | Customers with outstanding credit tabs. | `GET /customers/credits` |

---

## Credit tabs & reminders — `tools/credits.py` (7)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_credit_tabs` | `status?` (`open|partially_paid|settled|written_off`), `customer_id?`, `overdue?`, `page`, `page_size` | List credit tabs. | `GET /credits` |
| `customer_credit_profile` | `customer_id` | Open tabs, balance and payment history for a customer. | `GET /credits/customers/{id}` |
| `record_credit_payment` | `customer_id`, `amount`, `method="Cash"` (`Cash|M-Pesa|Tigo Pesa|Bank`), `reference?`, `note?` | Record a payment against a customer's credit balance. | `POST /credits/customers/{id}/payments` |
| `send_credit_reminder` | `customer_id` | Send an SMS reminder about outstanding credit (needs phone + configured SMS). | `POST /credits/customers/{id}/remind` |
| `add_credit_message` | `customer_id`, `body`, `kind="whatsapp"` (`whatsapp|call|sms`) | Log a message sent to the customer about their credit. | `POST /credits/customers/{id}/messages` |
| `add_credit_note` | `customer_id`, `body` | Add an internal note to the credit record (shop-only). | `POST /credits/customers/{id}/notes` |
| `write_off_credit` | `tab_id`, `reason?` | Forgive a credit tab. ⚠️ irreversible — always confirm first. | `POST /credits/{tab_id}/write-off` |

---

## Staff — `tools/staff.py` (10)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_staff` | `search?`, `role?` (`staff|owner`), `status?` (`active|on_leave|inactive`), `shift?` (`morning|evening|full_day|weekend`), `include_stats=False`, `page`, `size` | List staff. | `GET /staff` |
| `create_staff` | `first_name`, `last_name?`, `phone?`, `email?`, `role="staff"`, `store_id?`, `shift="morning"`, `employment_status="active"`, `pin?`, `can_refund=False`, `can_discount=True`, `can_view_reports=False` | Hire staff; a temporary password is sent by SMS/email. | `POST /staff` |
| `get_staff` | `staff_id` | One staff member. | `GET /staff/{id}` |
| `update_staff` | `staff_id` + profile fields | Partial profile update. | `PATCH /staff/{id}` |
| `deactivate_staff` | `staff_id` | Deactivate (cannot log in anymore). ⚠️ destructive. | `DELETE /staff/{id}` |
| `staff_stats` | `staff_id` | Sales stats: transactions, revenue, average ticket. | `GET /staff/{id}/stats` |
| `staff_kpis` | — | Team performance KPIs per store. | `GET /staff/kpis` |
| `staff_activity` | `staff_id`, `date?` (default today) | Clocking / activity log for a day. | `GET /staff/{id}/activity` |
| `update_staff_shift` | `staff_id`, `shift` | Change a staff member's shift. | `PATCH /staff/{id}/shift` |
| `update_staff_permissions` | `staff_id`, `can_refund?`, `can_discount?`, `can_view_reports?` | Change POS permissions. | `PATCH /staff/{id}/permissions` |

---

## Suppliers, deliveries & payments — `tools/suppliers.py` (13)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_suppliers` | `search?`, `status?` (`active|inactive|on_hold`), `page`, `page_size` | List suppliers. | `GET /suppliers` |
| `create_supplier` | `name`, `category?`, `rep?`, `phone?`, `email?`, `address?`, `payment_terms=30`, `notes?` | Add a supplier (`payment_terms` in days). | `POST /suppliers` |
| `get_supplier` | `supplier_id` | Supplier details including outstanding balance. | `GET /suppliers/{id}` |
| `update_supplier` | `supplier_id` + fields | Partial update. | `PATCH /suppliers/{id}` |
| `deactivate_supplier` | `supplier_id` | Soft-delete a supplier. ⚠️ destructive. | `DELETE /suppliers/{id}` |
| `supplier_stats` | — | Supplier counts + total outstanding. | `GET /suppliers/stats` |
| `list_deliveries` | `supplier_id?`, `unpaid?`, `page`, `page_size` | List stock deliveries. | `GET /suppliers/deliveries` |
| `create_delivery` | `supplier_id`, `total_value`, `invoice_number?`, `delivery_date?`, `items_count=1`, `status="delivered"`, `is_paid=False`, `notes?` | Record a delivery. | `POST /suppliers/deliveries` |
| `get_delivery` | `delivery_id` | One delivery. | `GET /suppliers/deliveries/{id}` |
| `update_delivery` | `delivery_id` + fields | Partial update. | `PATCH /suppliers/deliveries/{id}` |
| `delete_delivery` | `delivery_id` | Delete a delivery record. ⚠️ destructive. | `DELETE /suppliers/deliveries/{id}` |
| `list_supplier_payments` | `supplier_id?`, `page`, `page_size` | Payments made to suppliers. | `GET /suppliers/payments` |
| `record_supplier_payment` | `supplier_id`, `amount`, `payment_method="cash"` (`cash|bank|mobile`), `reference?`, `delivery_id?`, `notes?` | Record a payment to a supplier (pass `delivery_id` to settle a specific delivery). | `POST /suppliers/{id}/pay` |

---

## Expenses — `tools/expenses.py` (6)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_expenses` | `search?`, `category?` (`Rent|Utilities|Salaries|Supplies|Transport|Marketing|Maintenance|Licences & permits|Insurance|Other`), `payment_method?`, `ordering?`, `page`, `page_size` | List expenses. | `GET /expenses` |
| `create_expense` | `title`, `amount`, `category="Other"`, `payment_method="Cash"` (`Cash|M-Pesa|Tigo Pesa|Airtel Money|Bank`), `payment_reference?`, `notes?`, `receipt_url?` | Record an expense. | `POST /expenses` |
| `get_expense` | `expense_id` | One expense. | `GET /expenses/{id}` |
| `update_expense` | `expense_id` + fields | Partial update. | `PATCH /expenses/{id}` |
| `delete_expense` | `expense_id` | Delete an expense. ⚠️ destructive. | `DELETE /expenses/{id}` |
| `expense_summary` | — | Expense totals by period and category. | `GET /expenses/summary` |

---

## Reports & scheduled reports — `tools/reports.py` (9)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `report_types` | — | Available report types (`sales|inventory|tax|credit`). | `GET /reports/types` |
| `generate_report` | `report_type`, `range?` (`7d|30d|90d|month|ytd`), `date_from?`, `date_to?`, `format="json"` (`json|csv`) | Generate a report; `csv` returns downloadable bytes. | `POST /reports/generate` |
| `list_report_exports` | `page`, `page_size` | Previously generated exports. | `GET /reports/exports` |
| `download_report_export` | `export_id` | Re-generate + download a past export. | `GET /reports/exports/{id}/download` |
| `list_scheduled_reports` | — | Scheduled (auto-sent) reports. | `GET /reports/scheduled` |
| `create_scheduled_report` | `report_type`, `name?`, `frequency="daily"` (`daily|weekly|monthly`), `date_range_preset="30d"`, `recipients?` | Schedule a recurring email report. | `POST /reports/scheduled` |
| `update_scheduled_report` | `scheduled_id`, `name?`, `frequency?`, `date_range_preset?`, `recipients?`, `is_enabled?` | Patch a scheduled report. | `PATCH /reports/scheduled/{id}` |
| `delete_scheduled_report` | `scheduled_id` | Delete a scheduled report. ⚠️ destructive. | `DELETE /reports/scheduled/{id}` |
| `send_scheduled_report_now` | `scheduled_id` | Send a scheduled report immediately. | `POST /reports/scheduled/{id}/send` |

---

## Analytics / BI — `tools/analytics.py` (10)

All analytics tools take a range as either `range?` (`7d|30d|90d|ytd`, default `30d`) or explicit
`date_from?` / `date_to?`.

| Tool | Extra parameters | What it does | API call |
| --- | --- | --- | --- |
| `analytics_overview` | `sections?` (comma list: `kpis,trend,mix,products`) | Combined dashboard: KPIs, revenue trend, payment mix, top products. | `GET /analytics/overview` |
| `analytics_summary` | — | KPI summary: revenue, profit, transactions, margin, averages. | `GET /analytics/summary` |
| `analytics_trend` | — | Revenue trend (daily buckets). | `GET /analytics/trend` |
| `analytics_payment_mix` | — | Sales by payment method (Cash/M-Pesa/Tigo Pesa/Bank/Credit). | `GET /analytics/payment-mix` |
| `analytics_top_products` | `limit=10` (max 50) | Best-selling products. | `GET /analytics/top-products` |
| `analytics_sales` | — | Full sales report: KPIs + trend + method/hour breakdown. | `GET /analytics/sales` |
| `analytics_products` | `category?`, `sort?` | Per-product performance (revenue, qty, margin). | `GET /analytics/products` |
| `analytics_customers` | — | Customer analytics: new vs repeat, top spenders, segments. | `GET /analytics/customers` |
| `analytics_cashflow` | — | Money in vs out (sales/payments vs expenses/supplier payments). | `GET /analytics/cashflow` |
| `analytics_dashboard` | — | Pre-aggregated dashboard data. | `GET /analytics/dashboard` |

---

## Notifications — `tools/notifications.py` (5)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_notifications` | `category?` (`stock|credit|sales|ai|system|supplier`), `unread?`, `page`, `page_size=30` | List in-app notifications. | `GET /notifications` |
| `unread_notification_count` | — | Number of unread notifications. | `GET /notifications/unread-count` |
| `mark_notification_read` | `notification_id` | Mark one notification as read. | `PATCH /notifications/{id}/read` |
| `mark_all_notifications_read` | — | Mark everything read. | `PATCH /notifications/read-all` |
| `delete_notification` | `notification_id` | Delete one notification. | `DELETE /notifications/{id}` |

---

## Stores — `tools/stores.py` (7)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_stores` | `status?` (`open|closed|paused`) | List the business's stores. | `GET /stores` |
| `get_store` | `store_id` | Store details. | `GET /stores/{id}` |
| `store_stats` | — | Org-wide store stats (sales per branch). | `GET /stores/stats` |
| `set_store_status` | `store_id`, `status` (`open|closed|paused`) | Open / close / pause a store. | `PATCH /stores/{id}/status` |
| `store_weekly_performance` | `store_id` | Weekly performance breakdown. | `GET /stores/{id}/week` |
| `store_staff` | `store_id` | Staff roster at a store. | `GET /stores/{id}/staff` |
| `switch_active_store` | `store_id` | Switch the agent's active store (re-exchanges JWT). | `POST /accounts/switch-store` |

---

## Accounts / auth / org / users — `tools/accounts.py` (8)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `get_me` | — | The owner's profile, active store and all stores. | `GET /accounts/me` |
| `update_me` | `first_name?`, `last_name?`, `email?`, `occupation?`, `password?` | Patch the profile (only provided fields). | `PATCH /accounts/me` |
| `get_organisation` | — | Business profile (name, TIN, location, plan). | `GET /accounts/organisation` |
| `update_organisation` | `name?`, `legal_name?`, `tin?`, `business_type?`, `phone?`, `email?`, `address?`, `region?`, `district?` | Patch the business profile. | `PATCH /accounts/organisation` |
| `list_users` | — | All organisation users (owners + staff). | `GET /accounts/users` |
| `create_user` | `first_name`, `phone`, `last_name?`, `email?`, `role="staff"`, `store_id?`, `can_refund=False`, `can_discount=True`, `can_view_reports=False` | Create an org user; a temporary password is generated and sent. | `POST /accounts/users` |
| `deactivate_user` | `user_id` | Deactivate a user. ⚠️ destructive. | `DELETE /accounts/users/{id}` |
| `ai_credits` | — | The org's AI credit balance for the month. | `GET /accounts/ai-credits` |

---

## SMS / email — `tools/communicate.py` (4)

These four call the **providers directly** (not the ZiadaPOS API) via `NotifyClient`
(see [07 — Communication](07-communication.md)). They spend provider credits.

| Tool | Parameters | What it does | Provider call |
| --- | --- | --- | --- |
| `send_sms` | `to` (`0628587749` or `+255628587749`), `message`, `from_name?` | Send a real SMS; returns message id / status / credits used. | `POST {SendAfrica}/v1/sms/` |
| `sms_balance` | — | SendAfrica SMS credit balance. | `GET {SendAfrica}/v1/credits/balance` |
| `send_email` | `to`, `subject`, `body`, `html?` | Send a real email; returns email id + status. | `POST {MailAfrica}/api/outbound/emails` |
| `email_balance` | — | MailAfrica wallet balance in TZS. | `GET {MailAfrica}/api/billing/balance` |

---

## Product images — `tools/images.py` (1)

| Tool | Parameters | What it does |
| --- | --- | --- |
| `suggest_product_image` | `query` (e.g. `"Pepsi 500ml bottle"`), `max_bytes` (default 2 MiB) | Searches Wikimedia Commons, downloads the best matching thumbnail, uploads it to the shop's MinIO storage and returns `{url, source, title, content_type, size}` (or `{error}`). |

Full pipeline in [06 — Product Images](06-product-images.md).

---

## Notebook — `tools/notebook.py` (5)

| Tool | Parameters | What it does | API call |
| --- | --- | --- | --- |
| `list_notes` | `tag?`, `search?`, `ordering?` (`created|updated|title`, prefix `-` for descending) | List the store's notes. | `GET /notebook` |
| `create_note` | `title`, `content=""`, `tags?` | Create a note / reminder. | `POST /notebook` |
| `get_note` | `note_id` | One note. | `GET /notebook/{id}` |
| `update_note` | `note_id`, `title?`, `content?`, `tags?` | Patch a note. | `PATCH /notebook/{id}` |
| `delete_note` | `note_id` | Delete a note. | `DELETE /notebook/{id}` |

---

## The agent — `agent.py` (1)

| Tool | Parameters | What it does |
| --- | --- | --- |
| `ask` | `question` | Runs the Ngamia function-calling loop over all the tools above and returns a natural-language answer. Max steps = `AGENT_MAX_STEPS`. |

See [05 — Agent & Persona](05-agent-and-persona.md).

---

## Destructive tools

The following tools are guarded by the persona: the agent **must confirm with the owner before
calling them** (they are irreversible or cost money):

`refund_transaction`, `void_transaction`, `write_off_credit`, `delete_category`,
`delete_product`, `deactivate_supplier`, `delete_delivery`, `delete_expense`,
`deactivate_staff`, `delete_scheduled_report`, `delete_notification`, `delete_note`,
`deactivate_user`, `send_sms`, `send_email`.
