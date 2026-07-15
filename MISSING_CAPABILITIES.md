# Missing Capabilities Backlog

> Consolidated Hyperforge package readiness gaps from a full `pkg/` review (2026-07-15).
> Use this file as the shared backlog before defining Hyperforge services.
> Status legend: ❌ missing · 🔄 partial/stub · ⚠️ docs overclaim · 🔗 should reuse another package

**Policy:** Cloud agents must spawn subagents only with model `cursor-grok-4.5-high` (see `.cursor/rules/cloud-agent-subagent-model.mdc`).

**Cross-cutting rules for all follow-up work:**
1. Prefer existing packages — never reinvent (`pkg/errors`, `pkg/logger`, `pkg/resilience`, `pkg/concurrency`, `pkg/algorithms/*`, `pkg/datastructures/*`, `pkg/events`, `pkg/messaging`, `pkg/validator`).
2. Important functions need thorough tests (aim for full coverage of public API paths, including failure modes and `-race` where concurrent).
3. Docs must match reality — demote false ✅ in `pkg/TODO.md` when implementing or when documenting gaps here.

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
| communication | 58 | Adapters present; resilience/template/tests thin |
| data | 56 | Search strong; bigdata/docs overclaim |
| compute | 52 | Interfaces + memory; cloud stubs |
| concurrency | 52 | SmartMutex strong; rest experimental |
| network | 50* | LB/DNS OK; CDN/APIGW/IP thin; no algo reuse |
| api | 48 | Broad surface; GraphQL stub; standards weak |
| test | 45 | Thin Suite; low adoption; dead containers |
| commerce | 42 | Payment partial; billing/tax/FX memory |
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
| streaming | 25* | PutRecord stub; duplicates Pub/Sub |
| security | 30* | Memory-only domain |
| servicemesh | 25* | Discovery OK; CB/RL reinvent resilience/algorithms |
| storage | 45* | Blob partial; file/block/archive memory-only |
| resilience | 40* | CB+retry only; bulkhead claimed missing |

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
- [ ] 🔗 Wire `HTTPStatus`/`GRPCStatus` into `pkg/api/rest` and `pkg/api/grpc`
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

### `pkg/resilience` (~40)
- [ ] ❌ Interfaces + `instrumented.go` + `errors.go` + env `Config`
- [ ] ❌ Real Timeout / Bulkhead / Hedge / Fallback (docs claim Bulkhead)
- [ ] ❌ Half-open `MaxRequests`; typed `(T, error)` Execute
- [ ] 🔗 Single CB source of truth vs `pkg/servicemesh/circuitbreaker`
- [ ] ❌ Map `CIRCUIT_OPEN` → UNAVAILABLE/503
- [ ] ❌ Tests for WithTimeout, ExponentialBackoff, RetryWithCircuitBreaker

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
- [ ] ❌ GCS/Azure must implement `blob.Store`; map S3 miss → NotFound
- [ ] ❌ `blob/errors.go`; `pkg/resilience` on cloud I/O
- [ ] ❌ Production adapters for file/block/archive/controller (or demote TODO)
- [ ] 🔗 `pkg/concurrency` in memory adapters; typed `pkg/events` payloads
- [ ] ❌ Root `storage.go`; fix archive doc (cold storage ≠ tar/zip)

### `pkg/data` (~56)
- [ ] ⚠️ Remove or implement claimed `etl` / `processing` top-level packages
- [ ] ❌ Typesense/OpenSearch; search autocomplete; Snowflake
- [ ] 🔗 Reuse `pkg/concurrency`, `pkg/database/sql`, `pkg/storage` in bigdata paths
- [ ] ❌ Bigdata `errors.go` + full instrumented logging; Spark Connect honesty

### `pkg/streaming` (~25)
- [ ] 🔗 Remove Pub/Sub duplication with `pkg/messaging` (keep Kinesis/EventHubs only or fold)
- [ ] ❌ Consume/batch APIs; `errors.go`; `pkg/resilience`; real tests
- [ ] ⚠️ Fix README Kafka claim (Kafka is messaging, not streaming)

### `pkg/analytics` (~32)
- [ ] ❌ Event ingest model + streaming/warehouse sinks (catalog analytics)
- [ ] ❌ Redis HLL adapter; Merge/windows; exact counters
- [ ] ❌ Fix PACKAGE_STANDARDS §6.11 dead `NewUniqueTracker` example

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
- [ ] 🔗 Clarify vs streaming for GCP Pub/Sub

### `pkg/communication` (~58)
- [ ] ❌ Root `communication.go`; `errors.go`; `resilient.go` using `pkg/resilience`
- [ ] ❌ Real html/text template adapters (not sprintf stub)
- [ ] ❌ Honor Attachments/MediaURL/Retry*; propagate ctx on HTTP SDKs
- [ ] ❌ Adapter unit tests; Mailgun/WebPush or remove from docs

### `pkg/api` (~48)
- [ ] ❌ Real GraphQL wiring; gRPC health/stream/auth + `GRPCStatus` mapping
- [ ] ❌ Echo↔stdlib middleware bridge; apply REST timeouts
- [ ] ❌ WebSocket rooms/auth/origin; RBAC middleware + concurrency
- [ ] ❌ OpenAPI helpers; `errors.go`; rate-limit key strategies beyond IP

---

## 4. Security & auth

### `pkg/auth` (~57)
- [ ] ❌ OAuth2 authorization server (README/catalog promise)
- [ ] ❌ Complete Cognito/Entra/GCP Verify/Login; OIDC code exchange
- [ ] ❌ SMS/email MFA; Apple social; SAML client
- [ ] ❌ `errors.go` + unify cloud vs root IdP adapters
- [ ] ❌ Use EncryptionKey fields; WebAuthn memory real path

### `pkg/security` (~30)
- [ ] ❌ Production adapters (Vault, cloud KMS, reCAPTCHA, WAF, scanners)
- [ ] 🔗 Bridge with `pkg/auth` (stop parallel IdP models); hash via crypto
- [ ] ❌ `pkg/errors` in crypto/pqc; `pkg/validator` on Config
- [ ] ❌ Real PQC or mark experimental; Dilithium claim

### `pkg/audit` (~34)
- [ ] ❌ Durable adapters (kafka/postgres); query/export/retention/GDPR APIs
- [ ] ❌ Tamper-evident store; `Auditor` returns error; field-name redaction wired
- [ ] ❌ Real asserting tests

---

## 5. Infrastructure

### `pkg/network` (~50)
- [ ] 🔗 Wire `pkg/algorithms/loadbalancing` into LB selection (or document CP-only)
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

### `pkg/commerce` (~42)
- [ ] ❌ Root `commerce.go`; shared `Money` (no float64)
- [ ] ❌ Payment webhooks, auth/capture, idempotency; Braintree or drop claim
- [ ] ❌ Billing plans/proration/dunning; TaxJar/Avalara; live FX
- [ ] 🔗 `pkg/resilience` on Stripe/PayPal; `pkg/concurrency` in memory
- [ ] ❌ Domain events; stripe/paypal unit tests

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

### `pkg/iot` (~28)
- [ ] ❌ Interfaces + memory adapters + instrumented + tests (0% today)
- [ ] 🔗 `pkg/resilience` for OTA; `pkg/storage/blob` for firmware; `pkg/concurrency` for MQTT
- [ ] ❌ CoAP; device registry/certs; fix MQTT timeout + OTA semver
- [ ] ❌ Demote TODO ✅

### `pkg/web3` (~22)
- [ ] ❌ Interfaces + adapters/memory + instrumented + tests
- [ ] ❌ WalletConnect / DID resolve or drop claims; race-safe nonces
- [ ] ❌ SDK isolation under adapters

---

## 7. AI / algorithms / datastructures

### `pkg/ai` (~36)
- [ ] ❌ LLM streaming, multimodal, gateway, prompt engine, evals
- [ ] ❌ instrumented/errors/memory for all capabilities; Context on memory APIs
- [ ] 🔗 RAG ↔ `pkg/database/vector` + `pkg/database/rerank`
- [ ] ❌ OCR/vision/speech cloud adapters beyond stubs
- [ ] ⚠️ Fix TODO dual `ai/llm` vs `genai/llm` ledger

### `pkg/algorithms` (~38)
- [ ] ❌ Implement standards-cited `binarysearch`, `bfs`, `dfs`
- [ ] ❌ Finish Raft/Paxos/Chord/SWIM/Louvain/DistLimiter (or mark educational)
- [ ] ❌ True sliding window; health-aware / sticky LB; Maglev/P2C
- [ ] 🔗 Shared graph/heap types with `pkg/datastructures`

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
4. **Catalog depth:** commerce webhooks/FX/tax; auth OAuth2; storage blob Store parity; AI gateway/streaming
5. **Docs honesty:** `pkg/TODO.md` status pass; `pkg/README.md` maturity notes; package `doc.go` overclaims

---

## Review artifacts

Reviews were produced by parallel `cursor-grok-4.5-high` explore subagents, one per top-level `pkg/*` package, against `pkg/PACKAGE_STANDARDS.md`, `pkg/README.md`, `pkg/TODO.md`, and `services/SERVICE_CATALOG.md`.
