# Service Deepening Standard

Goal: replace generic `platform/crudserver` scaffolding with **domain services** that compose `pkg/` interfaces (memory adapters by default; cloud adapters via env).

## Definition of “deepened”

A deepened service MUST:

1. **Domain HTTP API** — named resources/actions (`POST /v1/payments/charge`), not generic CRUD bags.
2. **Wire `pkg/`** — depend on package interfaces (`payment.Provider`, `llm.Client`, `email.Sender`, …).
3. **Default to memory adapters** — tests and local `go run` work without cloud credentials.
4. **Env-selectable backends** where adapters exist (`PAYMENT_PROVIDER=memory|stripe|paypal`).
5. **Keep `/healthz`**.
6. **Tests** — httptest covering happy path + at least one domain error.
7. **Preserve gateway routes** — path prefixes already registered on the gateway stay stable (or gateway is updated in the same PR).

Optional but preferred:

- `instrumented` / `resilient` wrappers from `pkg/`
- Idempotency keys where the package supports them
- Structured errors via `pkg/errors`

## Waves

| Wave | Focus | Status | Examples |
|------|--------|--------|----------|
| 1 | Highest pkg leverage | **Done** | payment, billing, tax, email, sms, notification, push, llmgateway, embeddings, workflow, blob, cache, search, fraud, kms |
| 2 | Commerce + order path | **Done** | order, cart, inventory, product, pricing, subscription, invoice |
| 3 | AI cluster remainder | **Done** | agentruntime, toolregistry, contextmanager, vectorsearch, promptengine, agentorchestrator, mlinference |
| 4 | Security / compliance | **Done** | encryption, secretmanager, kyc, compliance, gdpr, retention, accesslogs, identityprovider |
| 5 | Observability / data / rest | **Done** | metrics, logs, traces, alerting, discovery, flags, ratelimit, media, chat, webhooks, etl, catalogs, schemas, backup, archival, analytics, reporting, cms, jobs, feedback, models, finetunes, recommendations, appconfig, audit, permission, incidentmanager |

Specialty services already domain-shaped (not crudserver): **auth**, **user**, **gateway**.

No `services/*/server` packages remain on `platform/crudserver`.

## Wave 1 deepened endpoints (summary)

- **payment** — charge / refund / get via `pkg/commerce/payment`
- **billing** — plans / subscriptions / invoices via `pkg/commerce/billing`
- **taxcalculator** — calculate via `pkg/commerce/tax`
- **email / sms / pushnotification / notification** — send via `pkg/communication/*`
- **llmgateway** — chat via `pkg/ai/genai/llm`
- **embeddingsvc** — embed via `pkg/ai/nlp/embedding`
- **workflow** — definitions / start / executions via `pkg/workflow`
- **blobstorage** — upload / download / delete via `pkg/storage/blob`
- **cachinglayer** — set / get / delete via `pkg/cache`
- **searchsvc** — index / query via `pkg/data/search`
- **frauddetection** — score via `pkg/security/fraud`
- **keymanagement** — encrypt / decrypt via `pkg/security/crypto/kms`

## Layout for a deepened service

```
services/<name>/
  cmd/<name>/main.go
  server/
    server.go      # domain routes + wiring
    server_test.go
  internal/        # optional service-local logic
```

Keep `Config` on the server package with `env` tags. Construct memory (or selected) adapters in `New(cfg)`.

## Non-goals (per service)

- Full production multi-tenant SaaS
- Replacing `pkg/` with service-local SDKs
- Breaking existing gateway path prefixes without updating gateway
