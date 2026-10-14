# SendAfrica and Ngamia integrations

## Owner setup

- Open **SMS messaging** to connect a SendAfrica API key and optionally enter the sender ID registered on that SendAfrica account. Sender IDs must be approved by the provider; eligibility and format depend on the destination country.
- Open **Settings → Ngamia AI** to connect an Ngamia key. Use **Fetch models / test connection** to load the live model catalog, then select a model and save it.
- Keys are entered into password fields, sent to the authenticated API server, encrypted before database persistence, and never included in API responses. The browser does not persist provider keys in `localStorage`.
- Only an organization owner/admin can update these settings. The authenticated organization ID scopes reads and writes. Customer SMS uses that organization’s own SendAfrica key and sender ID; it does not fall back to a different tenant’s or the platform’s SMS account.

## Server encryption key

Set `INTEGRATION_ENCRYPTION_KEY` in the API server’s environment. Generate one with:

```sh
openssl rand -base64 32
```

Keep the decoded key at exactly 32 random bytes, stable for the lifetime of the stored credentials, and backed up separately from the database. Do not commit it, put it in frontend variables, or pass it to the browser. If it is absent, new non-empty tenant API keys cannot be saved or used.

At startup, the API migrates legacy plaintext values in the organization provider-key columns to AES-256-GCM ciphertext before registering routes. If existing provider keys are present and the encryption key is missing or cannot decrypt existing ciphertext, startup fails closed; do not delete the old columns/data. Back up the database and preserve the current encryption key before deployment. Losing the key makes saved provider credentials unrecoverable; changing it requires an intentional re-encryption migration while the old key is still available.

The existing `.env` provider variables remain optional server-side defaults for platform-owned flows. They are not tenant settings and are never sent to the browser. Never put real values in `.env.example` or source control.

## SMS data flow

A user composes a message to one customer and must explicitly confirm the displayed recipient, exact text, and use of the organization’s provider balance. The server sends only `to`, `message`, and the configured sender ID (when set) to `https://api.sendafrica.online/v1/sms/` with the tenant key in `X-API-Key`. The app reports **provider accepted** plus the provider message ID/status; it does not claim delivery to the handset. Delivery webhooks are not configured, so handset delivery is not tracked here.

The API request logger records method, path, status, duration, and request ID only; it does not log request bodies or authorization headers. Provider error bodies are not relayed to clients.

## AI data flow and billing

Before an AI call, the user must explicitly check the data-sharing confirmation in the chat. The server sends the text typed by the user and recent conversation history plus a minimized shop summary to Ngamia’s OpenAI-compatible API. The generated summary contains daily/monthly sales aggregates, low-stock product names/SKUs, and aggregate credit counts/balances. It excludes customer names, phone numbers, addresses, and transaction identifiers. User-authored messages and conversation history are not automatically scrubbed; users should not include personal, payment, or other sensitive details.

Calls use the organization’s encrypted Ngamia key and selected live-catalog model when configured. Otherwise they may use the optional platform key and the existing organization AI-credit allowance. A tenant-owned key is billed against its Ngamia balance; model discovery itself does not submit chat content. Provider keys and prompts are sent only to the fixed provider endpoints configured by the server, never to a request-supplied URL.

## Official references

- [SendAfrica SMS API](https://docs.sendafrica.online/api/sms)
- [SendAfrica API docs](https://docs.sendafrica.online/)
- [Ngamia Chat Completions](https://docs.ngamia.cc/chat-completions)
- [Ngamia API docs](https://docs.ngamia.cc/)
