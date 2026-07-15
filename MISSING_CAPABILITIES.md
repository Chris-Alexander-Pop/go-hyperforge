# Missing Capabilities Backlog

> Consolidated Hyperforge package readiness gaps from a full `pkg/` review (2026-07-15).
> Use this file as the shared backlog before defining Hyperforge services.
> Status legend: ❌ missing · 🔄 partial/stub · ⚠️ docs overclaim · 🔗 should reuse another package

**Policy:** Cloud agents must spawn subagents only with model `cursor-grok-4.5-high` (see `.cursor/rules/cloud-agent-subagent-model.mdc`).
**Attribution:** All commits must be authored as Chris Pop `<chrisalexanderpop@gmail.com>` (see `.cursor/rules/git-commit-attribution.mdc`).

**Cross-cutting rules for all follow-up work:**
1. Prefer existing packages — never reinvent (`pkg/errors`, `pkg/logger`, `pkg/resilience`, `pkg/concurrency`, `pkg/algorithms/*`, `pkg/datastructures/*`, `pkg/events`, `pkg/messaging`, `pkg/validator`).
2. Important functions need thorough tests (aim for full coverage of public API paths, including failure modes and `-race` where concurrent).
3. Docs must match reality — demote false ✅ in `pkg/TODO.md` when implementing or when documenting gaps here.

### Progress since review (branch `branch/package-readiness-review-35ed`)

Landed foundation/reuse/domain hardening (scores above are the *pre-fix* snapshot):

- ✅ `errors`, `logger`, `cache`, `events`, `config`, `validator`, `resilience`
- ✅ `servicemesh` facades → resilience/algorithms; `network/loadbalancer` → algorithms
- ✅ `enterprise`, `metering`, `analytics`, `audit`, `iot`, `web3`, `communication`, `streaming`
- ✅ `database` resilience/sharding helpers; `workflow` distlock + events + cron
- ✅ `algorithms`: binarysearch, bfs/dfs, DistLimiter store-backed, sliding-window counter, educational stub docs, heap reuse in dijkstra/astar
- ✅ `storage` root drivers, blob errors/resilience, GCS/Azure `blob.Store`, S3 miss→NotFound, SmartRWMutex memory adapters
- ✅ `security`: root/errors, crypto harden + memory KeyProvider, secrets Rotate/events, reCAPTCHA adapter, honest docs + auth bridge
- 🔄 `auth`: OAuth2 AS interfaces + memory; Cognito/Entra/GCP Verify/Login; OIDC exchange; EncryptionKey; root errors.go
- ✅ `commerce`: root Money, payment webhooks/auth-capture/idempotency/events/resilience, billing plans+past_due, tax multi-jurisdiction, FX formatting
- 🔄 Remaining large gaps still listed below (TaxJar/Avalara/live FX, AI gateway, cloud IaaS adapters, security Vault/cloud KMS/WAF, etc.)

---

## Completeness scores (review snapshot)

| Package | Score | Notes |
|---------|------:|-------|
| messaging | 71 | Strong adapters; factory/tests/options gaps |
| database | 62 | Broad skeleton; sharding/resilience/tests thin |
| auth | 57 | Session/MFA/JWT solid; OAuth2 AS + cloud stubs |
| cache | 60 | Core OK; TTL=0 / miss→CB footguns |
| logger | 58 | Widely used; Init/Async/trace bugs |
| errors | 58 | Foundation usable; codes/Is/Wrap incomplete |
| datastructures | 58 | Broad catalog; many stubs / low reuse |
| communication | 58 | Ready: root drivers/errors/resilience, html/text templates, adapter tests |
| data | 56 | Search strong; bigdata/docs overclaim |
| compute | 52 | Interfaces + memory; cloud stubs |
| concurrency | 52 | SmartMutex strong; rest experimental |
| network | 50* | LB/DNS OK; CDN/APIGW/IP thin; no algo reuse |
| api | 48 | Broad surface; GraphQL stub; standards weak |
| test | 45 | Thin Suite; low adoption; dead containers |
| commerce | 42→68 | Money + payment depth; billing plans; tax multi-jurisdiction; FX still memory |
| events | 42 | Skeleton bus; unsafe async |
| workflow | 38 | Scaffold; no events/messaging/distlock |
| algorithms | 38 | Many educational stubs |
| cloud | 38 | Memory-only IaaS scaffold |
| telemetry | 36 | OTLP init stub only |
| ai | 36 | Broad stubs; gateway/streaming missing |
| analytics | 32 | HLL uniqueness only |
| validator | 32 | Thin; config bypasses it |
| audit | 34 | Stdout + redact; no store/query |
| config | 28 | Unused env loader; reinvented validation |
| iot | 28 | Concrete SDKs; 0 tests |
| enterprise | 24 | Design stub; 0 tests |
| web3 | 22 | Client scaffolds; no interfaces/tests |
| metering | 20* | Memory only; 0 tests; no consumers |
| streaming | — | PutRecord + memory/Kinesis/EventHubs; Pub/Sub → messaging |
| security | 30* → improved | Root/errors, crypto harden, secrets Rotate, reCAPTCHA; Vault/cloud KMS still open |
| servicemesh | 25* | Discovery OK; CB/RL reinvent resilience/algorithms |
| storage | 45* | Blob Store parity + resilience landed; file/block/archive still memory-only |
| resilience | 75* | CB+retry+timeout+bulkhead; typed Execute / Hedge / Fallback still open |

\*Approximate where review used checklist form without a single headline score.

---

## Cross-cutting (all packages)

- [ ] 🔗 Use `pkg/errors` everywhere (no `fmt.Errorf` / stdlib `errors.New` for domain errors)
- [ ] 🔗 Use `pkg/concurrency.SmartMutex` / `SmartRWMutex` instead of `sync.Mutex` / `RWMutex`
- [ ] 🔗 Use `pkg/resilience` for all external I/O (CB + retry); delete reinvented wrappers
- [ ] 🔗 Use `pkg/validator` for Config validation; fix `pkg/config` to call it
- [ ] 🔗 Use `pkg/algorithms/*` and `pkg/datastructures/*` instead of local copies (Dijkstra PQ, LB selection, etc.)
- [ ] 🔗 Emit domain events via `pkg/events` where standards §9 apply
- [ ] ❌ Package `errors.go` + `instrumented.go` + `adapters/memory/` where PACKAGE_STANDARDS require them
- [ ] ❌ Interface tests / `pkg/test` suites for every adapter surface
- [ ] ⚠️ Align module branding (`go-hyperforge` vs `system-design-library` in go.mod/imports)
- [ ] ⚠️ Demote false ✅ in `pkg/TODO.md` to 🔄/❌ to match this backlog

---

## 1. Core foundation

### `pkg/errors` (~58)
- [ ] ❌ Codes: `DEADLINE_EXCEEDED`, `UNAVAILABLE`, `RESOURCE_EXHAUSTED`, `CANCELED`, `ABORTED`, `FAILED_PRECONDITION`
- [ ] ❌ `IsCode(err, code)` / `Code(err)` helpers
- [ ] ❌ `Wrap` preserving `*AppError` (or `WrapCode`)
- [ ] ❌ HTTP/gRPC mapping for custom/domain codes; reverse `FromHTTP` / `FromGRPC`
- [x] 🔗 Wire `HTTPStatus`/`GRPCStatus` into `pkg/api/rest` and `pkg/api/grpc`
- [ ] ❌ Full test matrix for all helpers + wrapped errors

### `pkg/logger` (~58)
- [ ] ❌ Fix `Init` double-wrap of handler stack
- [ ] ❌ Trace correlation with default `Async=true` (attrs before queue / copy span IDs)
- [ ] ❌ `Shutdown(ctx)` flush for AsyncHandler
- [ ] ❌ Redact `WithAttrs` / bound attrs
- [ ] ❌ Bootstrap: apps must call `Init`; examples in templates/services
- [ ] ❌ Tests for Init layering, Trace+Async, WithAttrs leak

### `pkg/config` (~28)
- [ ] 🔗 Route validation through `pkg/validator` (not raw playground)
- [ ] ❌ Typed `AppError`s (`InvalidArgument` / `Internal`) instead of unstructured `Wrap`
- [ ] ❌ `LoadFrom(path)` / options; multi-format; secrets integration
- [ ] ❌ In-repo adoption (`config.Load` unused outside itself)
- [ ] ❌ Failure-path tests

### `pkg/validator` (~32)
- [ ] ❌ Interfaces + `errors.go` + `instrumented.go`
- [ ] ❌ Map failures to `errors.InvalidArgument`
- [ ] ❌ Context-first APIs; implement or remove dead `AllowedTags`
- [ ] ❌ Tests for slug/phone/SQL/command/SanitizeMap

### `pkg/telemetry` (~36)
- [ ] ❌ Adapter-isolated exporters; noop/stdout for tests
- [ ] ❌ Configurable sampler + TLS (not AlwaysSample + Insecure)
- [ ] ❌ Metrics pipeline; `Init(ctx, cfg)`; shared span helpers
- [ ] ❌ Deterministic tests (no hang on collector)

### `pkg/test` (~45)
- [ ] ❌ Self-tests + examples; split/remove unused testcontainers helpers
- [ ] ❌ Drive adoption in cache/messaging/events/resilience/logger/api

### `pkg/resilience` (~75)
- [x] ✅ Breaker/Retrier interfaces + `instrumented.go` + `errors.go` (UNAVAILABLE/RESOURCE_EXHAUSTED)
- [x] ✅ Real Timeout (`WithTimeout`) + semaphore Bulkhead via `pkg/concurrency`
- [ ] ❌ Hedge / Fallback; typed `(T, error)` Execute; env `Config`
- [x] ✅ Half-open `MaxRequests` (`ErrTooManyRequests`)
- [x] 🔗 Single CB source of truth vs `pkg/servicemesh/circuitbreaker` (thin facade)
- [x] ✅ Map circuit-open → UNAVAILABLE/503; bulkhead/half-open cap → RESOURCE_EXHAUSTED/429
- [x] ✅ Tests for WithTimeout, ExponentialBackoff, RetryWithCircuitBreaker, Bulkhead, MaxRequests

### `pkg/concurrency` (~52)
- [ ] 🔗 Wrap/re-export `x/sync/semaphore` + `errgroup` instead of competing copies
- [ ] ❌ Distlock: use `LockConfig`, retry, Redlock-or-honest-docs, `pkg/errors`
- [ ] 🔗 Wire `algorithms/concurrency/adaptive` into pools
- [ ] ❌ Tests for semaphore/pool/pipeline/runner/redis lock
- [ ] ❌ `singleflight`-style coalesce helper

### `pkg/events` (~42)
- [ ] ❌ `Config`, `errors.go`, Unsubscribe, graceful Close
- [ ] ❌ Bounded async via `pkg/concurrency.WorkerPool`; propagate ctx; surface handler errors
- [ ] ❌ Outbox / messaging bridge helpers (standards §9.5)
- [ ] ❌ Full fan-out / Close / race / instrumented tests

---

## 2. Data & storage

### `pkg/cache` (~60)
- [ ] ❌ Fix memory TTL=0 (“no expiration” currently expires immediately)
- [ ] ❌ ResilientCache / Instrumented: do not treat NotFound as failure
- [ ] ❌ `errors.go`, `manager.go`, Config parity (pool/TLS/timeouts)
- [ ] ❌ Exists/MGet/MSet/Expire/invalidation/warming; Redis Cluster
- [ ] ❌ Redis conformance tests (miniredis already in go.mod)

### `pkg/database` (~62)
- [ ] ❌ Multi-shard manager wiring `pkg/algorithms/consistenthash` into `GetShard`
- [ ] 🔗 Replace `ops.WithRetry` with `pkg/resilience`
- [ ] ❌ Adapters: Cassandra KV, Neo4j/Neptune graph, Weaviate/Milvus vector
- [ ] ❌ ClickHouse must implement `sql.SQL`; vector filters/hybrid search
- [ ] ❌ Interface conformance tests across stores

### `pkg/storage` (~45)
- [x] ✅ GCS/Azure implement `blob.Store`; map S3 miss → NotFound
- [x] ✅ `blob/errors.go`; `pkg/resilience` on cloud I/O (`resilient.go`)
- [x] ✅ Docs demoted: file/block/archive/controller memory-only (cloud adapters not claimed)
- [x] ✅ `pkg/concurrency` in memory adapters; typed `pkg/events` payloads (`BlobEventPayload`)
- [x] ✅ Root `storage.go`; archive doc clarified (cold storage ≠ tar/zip)
- [ ] ❌ Production adapters for file/block/archive/controller (still future work)

### `pkg/data` (~56)
- [ ] ⚠️ Remove or implement claimed `etl` / `processing` top-level packages
- [ ] ❌ Typesense/OpenSearch; search autocomplete; Snowflake
- [ ] 🔗 Reuse `pkg/concurrency`, `pkg/database/sql`, `pkg/storage` in bigdata paths
- [ ] ❌ Bigdata `errors.go` + full instrumented logging; Spark Connect honesty

### `pkg/streaming` (~25)
- [x] ✅ Remove Pub/Sub duplication with `pkg/messaging` (Kinesis/EventHubs + memory only)
- [x] ✅ `errors.go`; `resilient.go` via `pkg/resilience`; root memory tests; BufferSize honored
- [x] ✅ Fix README: Kafka and Pub/Sub live under `messaging`, not `streaming`
- [ ] ❌ Consume/batch APIs (out of current PutRecord-only scope)

### `pkg/analytics` (~32)
- [ ] ❌ Event ingest model + streaming/warehouse sinks (catalog analytics)
- [x] ✅ Redis HLL adapter (PFADD/PFCOUNT/PFMERGE); Merge on Tracker; precision 4–16
- [ ] ❌ Windows / exact counters (out of uniqueness-only scope today)
- [x] ✅ Fix PACKAGE_STANDARDS §6.11 example (`memory.New` + Close/Merge)

### `pkg/metering` (~20)
- [ ] ❌ Tests; `InstrumentedRater`; postgres/prometheus adapters or honest Config
- [ ] 🔗 Wire to `pkg/commerce/billing` + `pkg/events`
- [ ] ❌ Period aggregation / rate-card mutation APIs

---

## 3. Communication & API

### `pkg/messaging` (~71)
- [ ] ❌ `manager.go` factory; wire Publish/Consume options or remove them
- [ ] ❌ Honor TLS/claim/prefetch config fields; memory `ErrQueueFull`
- [ ] ❌ ResilientConsumer; adapter unit/integration tests
- [x] ✅ Clarify vs streaming for GCP Pub/Sub (streaming docs point to messaging)

### `pkg/communication` (~58)
- [x] ✅ Root `communication.go`; `errors.go`; `resilient.go` using `pkg/resilience`
- [x] ✅ Real html/text template adapters (not sprintf stub)
- [x] ✅ Honor Attachments/MediaURL/Retry*; propagate ctx on HTTP SDKs
- [x] ✅ Adapter unit tests; Mailgun/WebPush softened in docs (not implemented)

### `pkg/api` (~48 → improved)
- [x] ✅ GraphQL schema injection via `api.Config.GraphQLSchema` (no no-op stub); honest docs
- [x] ✅ gRPC health (`grpc.health.v1`), stream recovery, unary `GRPCStatus` ErrorInterceptor
- [x] ✅ REST `ReadTimeout`/`WriteTimeout` applied; full `HTTPStatus` error map
- [x] ✅ WebSocket origin allowlist, Hub `Shutdown`, broadcast no longer mutates under RLock
- [x] ✅ RBAC `SmartRWMutex` + `middleware.RequirePermission`; rate-limit `KeyByUser`/`KeyByAPIKey`
- [x] ✅ `pkg/api/errors.go`; softened overclaiming `doc.go`s; tests for RBAC/WS/HTTPStatus
- [ ] ❌ OpenAPI helpers; Echo↔stdlib middleware bridge utilities
- [ ] ❌ WebSocket rooms / upgrade-time auth; gRPC auth + stream error-mapping interceptors

---

## 4. Security & auth

### `pkg/auth` (~57)
- [x] ✅ OAuth2 authorization server interfaces + memory adapter (auth code / client credentials / refresh; not full OpenID Provider)
- [x] ✅ Cognito/Entra Verify via OIDC JWKS; GCP Login via Identity Toolkit REST; OIDC code exchange + memory exchanger
- [ ] ❌ SMS/email MFA; Apple social; SAML client
- [x] ✅ Root `errors.go` sentinels; cloud vs root IdP adapters remain dual surfaces (documented)
- [x] ✅ EncryptionKey wired for session/MFA memory+redis; WebAuthn memory still stub (library adapter is real path)

### `pkg/security` (~30 → improved)
- [x] ✅ Root `security.go` + domain `errors.go` (fraud/captcha/waf/scanning/secrets/kms/crypto) via `pkg/errors`
- [x] ✅ Crypto: `pkg/errors`, `crypto/subtle` compare, `InstrumentedEncryptor`, MemoryKeyProvider → `crypto/adapters/memory`
- [x] ✅ Secrets: `Rotate` + Config `Validate` (`pkg/validator`) + optional `EventedSecretManager` audit events
- [x] ✅ Captcha: `adapters/recaptcha` siteverify HTTP adapter + honest memory/docs
- [x] ✅ Softened Vault/cloud KMS/Dilithium/reCAPTCHA overclaims; bridge note vs `pkg/auth` IdP
- [ ] ❌ Remaining production adapters (Vault, cloud KMS/WAF, scanners, GuardDuty)
- [ ] ❌ Real/vetted PQC (CIRCL/liboqs); Dilithium/ML-DSA still absent
- [ ] 🔗 Broader hash/password reuse via crypto across auth (partial)

### `pkg/audit` (~34)
- [ ] ❌ Durable adapters (kafka/postgres); query/export/retention/GDPR APIs
- [ ] ❌ Tamper-evident store; `Auditor` returns error; field-name redaction wired
- [ ] ❌ Real asserting tests

---

## 5. Infrastructure

### `pkg/network` (~50)
- [x] 🔗 Wire `pkg/algorithms/loadbalancing` into LB selection (memory `SelectTarget`)
- [ ] ❌ `instrumented.go` + `errors.go` for cdn/apigateway/ip + root TCP/UDP
- [ ] ❌ Cloud adapters matching TODO (Route53, CloudFront, etc.) or demote ✅
- [ ] 🔗 `pkg/concurrency` in all memory adapters

### `pkg/compute` (~52)
- [ ] ❌ VM adapters EC2/GCE/Azure; Docker; Azure Functions
- [ ] ❌ Fix k8s ID/name bug; real Exec/Stats; `pkg/resilience`
- [ ] 🔗 `pkg/concurrency` in memory; use package sentinels
- [ ] ❌ Root `compute.go`; clarify vs `pkg/cloud`

### `pkg/cloud` (~38)
- [ ] ❌ Libvirt/Firecracker/IPMI/PXE/etcd adapters (or demote TODO ✅)
- [ ] ❌ Control-plane instance APIs; real scheduler strategies
- [ ] ❌ Shared vocabulary with `pkg/compute`
- [ ] ❌ Tests beyond one hypervisor case

### `pkg/servicemesh` (~25)
- [ ] 🔗 **Delete or thin-wrap** circuitbreaker → `pkg/resilience`
- [ ] 🔗 **Delete or thin-wrap** ratelimit → `pkg/algorithms/ratelimit` (+ `pkg/api/ratelimit`)
- [ ] ❌ Keep/expand discovery with Consul/etcd/K8s adapters
- [ ] ❌ Mesh: mTLS, retry reuse, honest docs

### `pkg/storage` — see Data & storage

---

## 6. Domain & enterprise

### `pkg/commerce` (~42 → improved)
- [x] ✅ Root `commerce.go`; shared `Money` (int64 minor units, no float64)
- [x] ✅ Payment webhooks (Stripe HMAC + PayPal verifier), Authorizer auth/capture/void, Charge idempotency; Braintree claim dropped
- [x] ✅ Billing Plan catalog + Upgrade stub + `StatusPastDue` via MarkPastDue; memory plan catalog
- [x] 🔗 `pkg/resilience` on Stripe/PayPal (+ ResilientProvider); `SmartRWMutex` in memory adapters
- [x] ✅ Domain events (`NewEventedProvider`); webhook + money + memory billing unit tests
- [ ] 🔄 Real proration/dunning automation; TaxJar/Avalara adapters; live FX `LiveRateProvider` impl

### `pkg/enterprise` (~24)
- [ ] ❌ Full standards layout (instrumented, adapters/memory, Config, tests)
- [ ] 🔗 Bridge eventsource → `pkg/events` / `pkg/messaging` / `pkg/database`
- [ ] ❌ Projection runner; durable store; fix LoadFrom/version bugs
- [ ] ❌ Demote TODO ✅ → 🔄

### `pkg/workflow` (~38)
- [ ] ❌ Real state-machine execution; honor timeout/idempotency
- [ ] 🔗 Scheduler + `pkg/concurrency/distlock`; saga + `pkg/events`/`messaging`
- [ ] ❌ Durable saga; real cron; cloud adapter completeness
- [ ] ❌ Saga/scheduler instrumented + interfaces

### `pkg/iot` (~28 → improved)
- [x] ✅ Root Client/Updater interfaces + memory adapters + instrumented + tests
- [x] 🔗 `pkg/resilience` for OTA downloads; `pkg/concurrency` for MQTT/memory
- [x] ✅ MQTT WaitTimeout bug fixed; OTA semver via `golang.org/x/mod/semver`
- [ ] ❌ CoAP; device registry/certs; `pkg/storage/blob` firmware backing
- [ ] ❌ AWS IoT / Greengrass behind root Client interface
- [x] ✅ Demoted TODO overclaims

### `pkg/web3` (~22)
- [x] ✅ Interfaces + adapters/memory + instrumented + tests
- [x] ✅ Softened WalletConnect / DID claims; race-safe SIWE nonces
- [ ] 🔄 SDK isolation under adapters (ethereum/ipfs still concrete scaffolds)

---

## 7. AI / algorithms / datastructures

### `pkg/ai` (~36)
- [ ] ❌ LLM streaming, multimodal, gateway, prompt engine, evals
- [ ] ❌ instrumented/errors/memory for all capabilities; Context on memory APIs
- [ ] 🔗 RAG ↔ `pkg/database/vector` + `pkg/database/rerank`
- [ ] ❌ OCR/vision/speech cloud adapters beyond stubs
- [ ] ⚠️ Fix TODO dual `ai/llm` vs `genai/llm` ledger

### `pkg/algorithms` (~38 → improved)
- [x] ✅ Implement standards-cited `search/binarysearch`, `graph/bfs`, `graph/dfs` (+ tests)
- [x] ✅ Soften Raft/Paxos/Chord/SWIM/Louvain docs as educational stubs; DistLimiter uses cache store
- [x] ✅ Sliding window counter (weighted prev+curr windows); Local remains exact log
- [x] 🔗 Dijkstra/A* reuse `pkg/datastructures/heap`; shared `algorithms/graph` types
- [ ] ❌ Health-aware / sticky LB; Maglev/P2C
- [ ] ❌ Finish Raft/Paxos/Chord/SWIM/Louvain beyond educational stubs

### `pkg/datastructures` (~58)
- [ ] ❌ Tests for ARC/CRDT/roaring/cuckoo/scalable/DAG
- [ ] ❌ Honest docs (drop Consistent Hashing/G-Set/Red-Black until real)
- [ ] 🔗 Drive reuse into algorithms/cache/workflow (stop local PQs)
- [ ] ❌ Harden or quarantine placeholders (tdigest, histogram, disruptor, hllpp)

---

## Suggested implementation order (for agents)

1. **Foundation correctness:** logger Init/trace, errors codes/Wrap/IsCode, config→validator, cache TTL + miss semantics
2. **Reuse cleanup:** servicemesh wraps resilience/algorithms; network uses loadbalancing algos; database uses resilience; streaming vs messaging boundary
3. **Standards skeleton:** events Config/errors/lifecycle; enterprise/iot/web3/metering tests + memory adapters
4. **Catalog depth:** commerce TaxJar/Avalara/live FX; auth OAuth2 polish; storage file/block/archive cloud adapters; AI gateway/streaming
5. **Docs honesty:** `pkg/TODO.md` status pass; `pkg/README.md` maturity notes; package `doc.go` overclaims

---

## Review artifacts

Reviews were produced by parallel `cursor-grok-4.5-high` explore subagents, one per top-level `pkg/*` package, against `pkg/PACKAGE_STANDARDS.md`, `pkg/README.md`, `pkg/TODO.md`, and `services/SERVICE_CATALOG.md`.
