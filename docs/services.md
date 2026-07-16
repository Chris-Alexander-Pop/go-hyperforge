# Hyperforge Service Conventions

Conventions for microservices under `services/`. Packages in `pkg/` remain the capability layer; services are thin HTTP processes that compose those packages.

## Directory layout

```
services/<name>/
  cmd/<name>/main.go     # process entrypoint
  server/                # HTTP API + typed env Config (importable)
  internal/...           # domain logic / stores when not using platform/crudserver
```

Shared helpers live in [`services/platform`](../services/platform): bootstrap, `memstore`, and `crudserver`.

## Bootstrap

1. Load config with `pkg/config.Load`.
2. Initialize the process logger with `pkg/logger.Init`.
3. Build the HTTP server with `pkg/api/rest` (Echo).
4. Register `GET /healthz` on every service.
5. Call `pkg/logger.Shutdown` on process exit.

Use `services/platform.Bootstrap` for steps 1–2, or copy the pattern from `templates/service/starter`.

## HTTP

- REST/JSON via Echo (`pkg/api/rest`).
- Map domain errors through Echo’s error handler (`pkg/errors` → HTTP status).
- Public APIs are versioned under `/v1/...`.
- Health: `GET /healthz` → `{"status":"ok"}`.

## Identity cluster (v1)

| Service      | Default port | Role                                      |
|--------------|--------------|-------------------------------------------|
| gateway      | `8080`       | Edge entry; JWT verify; reverse proxy     |
| auth         | `8081`       | Register / login; issues JWTs             |
| user         | `8082`       | User profiles; trusts `X-User-ID`         |
| permission   | `8083`       | Permission records                        |
| notification | `8084`       | Notification records                      |
| email        | `8085`       | Email send records                        |
| sms          | `8086`       | SMS send records                          |
| product      | `8087`       | Product catalog (public via gateway)      |
| cart         | `8088`       | Shopping carts                            |
| order        | `8089`       | Orders                                    |
| payment      | `8090`       | Payments                                  |
| inventory    | `8091`       | Inventory                                 |
| appconfig    | `8092`       | App / feature config                      |
| audit        | `8093`       | Audit events                              |
| workflow     | `8094`       | Workflow instances                        |
| llmgateway   | `8095`       | LLM request gateway                       |
| agentruntime | `8096`       | Agent runtime                             |
| toolregistry | `8097`       | Tool registry                             |
| contextmanager | `8098`     | Conversation / agent contexts             |
| embeddingsvc | `8099`       | Embeddings                                |
| vectorsearch | `8100`       | Vector search                             |
| promptengine | `8101`       | Prompt templates                          |
| metricscollector | `8102`   | Metrics ingestion                         |
| logaggregator | `8103`      | Log aggregation                           |
| tracecollector | `8104`     | Trace collection                          |
| alerting     | `8105`       | Alerting                                  |
| discovery    | `8106`       | Service discovery registry                |
| featureflag  | `8107`       | Feature flags                             |
| secretmanager | `8108`      | Secrets                                   |
| searchsvc    | `8109`       | Search (public via gateway)               |
| mediasvc     | `8110`       | Media assets                              |
| ratelimitersvc | `8111`     | Rate-limit policies                       |
| pricing | `8112` | Pricing |
| analytics | `8113` | Analytics |
| reporting | `8114` | Reporting |
| mlinference | `8115` | ML inference |
| recommendation | `8116` | Recommendations (public) |
| cms | `8117` | CMS pages (public) |
| scheduledjobs | `8118` | Scheduled jobs |
| agentorchestrator | `8119` | Agent orchestrator |
| finetuning | `8120` | Fine-tuning jobs |
| modelregistry | `8121` | Model registry |
| billing | `8122` | Billing |
| invoice | `8123` | Invoices |
| taxcalculator | `8124` | Tax calculator |
| subscription | `8125` | Subscriptions |
| feedback | `8126` | Feedback |
| identityprovider | `8127` | Identity provider records |
| pushnotification | `8128` | Push notifications |
| chat | `8129` | Chat |
| webhookmanager | `8130` | Webhooks |
| frauddetection | `8131` | Fraud detection |
| kycverification | `8132` | KYC |
| encryption | `8133` | Encryption |
| keymanagement | `8134` | Key management |
| compliance | `8135` | Compliance |
| dataretention | `8136` | Data retention |
| gdprprocessor | `8137` | GDPR processor |
| accesslogs | `8138` | Access logs |
| etlpipeline | `8139` | ETL pipelines |
| datacatalog | `8140` | Data catalog |
| schemaregistry | `8141` | Schema registry |
| backupsvc | `8142` | Backups |
| archival | `8143` | Archival |
| cachinglayer | `8144` | Caching layer |
| blobstorage | `8145` | Blob storage |
| incidentmanager | `8146` | Incident manager |

CRUD-shaped services use `services/platform/crudserver` + `memstore` (in-memory). Domain-deep adapters come later.

Shared secrets: `JWT_SECRET` and `JWT_ISSUER` must match on **auth** and **gateway**.

After JWT verification, gateway strips `Authorization` and injects:

- `X-User-ID` — JWT subject
- `X-User-Roles` — comma-separated roles (optional)

Downstream services (user) trust these headers from the gateway only. Do not expose user directly without the gateway in production topologies.

## Config

Use `env` struct tags compatible with `pkg/config` / cleanenv. Prefer defaults that work with `make up` (Postgres/Redis/NATS) even when v1 uses memory stores.

## Testing

- Prefer `net/http/httptest` handler tests.
- Keep memory adapters for unit tests; integration against compose infra is optional.

## Out of scope for v1 conventions

- gRPC / GraphQL transports
- Durable Postgres schemas (interfaces should allow swapping later)
- Service mesh / mTLS
