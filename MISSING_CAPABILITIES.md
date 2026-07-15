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

### Still open (truly remaining)

- Cross-cutting: adopt `pkg/errors` / `SmartMutex` / `resilience` / `validator` / `algorithms` / `events` everywhere; PACKAGE_STANDARDS skeletons & interface tests; demote false ✅ in `pkg/TODO.md`
- `pkg/logger`: bootstrap examples in templates/services (Init/Shutdown already fixed)
- `pkg/config`: broader in-repo `Load` adoption
- `pkg/test`: drive Suite adoption across cache/messaging/events/…
- `pkg/concurrency`: re-export `x/sync/semaphore` + `errgroup` (optional)
- `pkg/database`: Cassandra KV, Neptune graph, Milvus vector; broader conformance
- `pkg/storage`: block/archive/controller cloud adapters (file **local** shipped)
- `pkg/data`: real Typesense/OpenSearch HTTP clients (memory stubs exist); Snowflake
- `pkg/analytics`: exact counters / warehouse adapters
- `pkg/metering`: period aggregation / rate-card mutation APIs; postgres adapter
- `pkg/api`: Echo↔stdlib middleware bridge; WS rooms / upgrade auth; gRPC auth + stream error mapping
- `pkg/security`: AWS WAF, GCP/Azure KMS, cloud secret managers, scanners, GuardDuty; real CIRCL/liboqs PQC (experimental stub remains)
- `pkg/cloud`: PXE imaging + postgres controlplane driver
- `pkg/servicemesh`: etcd/K8s discovery (mTLS helpers + honest retry docs landed)
- `pkg/enterprise`: fuller standards layout polish (ProjectionRunner + durable checkpoint + messaging outbox landed)
- `pkg/workflow`: real cron; cloud adapter completeness; saga/scheduler instrumented
- `pkg/iot`: device certs; Greengrass behind root Client
- `pkg/web3`: fuller SDK isolation under adapters
- `pkg/ai`: fuller prompt engine; remaining ML depth beyond inference/feature instrumented
- `pkg/algorithms`: Raft/Paxos/Chord/SWIM/Louvain beyond educational stubs; sticky LB
- `pkg/datastructures`: drive reuse into algorithms/cache/workflow

### Progress since review (branch `branch/package-readiness-review-35ed`)

Landed foundation/reuse/domain hardening (scores above are the *pre-fix* snapshot):

- ✅ `errors`, `logger`, `cache`, `events`, `config`, `validator`, `resilience`
- ✅ `servicemesh` facades → resilience/algorithms; `network/loadbalancer` → algorithms
- ✅ `enterprise`, `metering`, `analytics`, `audit`, `iot`, `web3`, `communication`, `streaming`
- ✅ `database` resilience/sharding helpers; `workflow` distlock + events + cron
- ✅ `algorithms`: binarysearch, bfs/dfs, DistLimiter store-backed, sliding-window counter, educational stub docs, heap reuse in dijkstra/astar
- ✅ `storage` root drivers, blob errors/resilience, GCS/Azure `blob.Store`, S3 miss→NotFound, SmartRWMutex memory adapters
- ✅ `security`: root/errors, crypto harden + memory KeyProvider, secrets Rotate/events, reCAPTCHA adapter, honest docs + auth bridge
- ✅ `auth`: OAuth2 AS + IdP verify/login; SMS/email MFA via communication; Apple social; WebAuthn memory test path; EncryptionKey; root errors
- ✅ `commerce`: root Money, payment webhooks/auth-capture/idempotency/events/resilience, billing plans+proration+dunning, TaxJar/Avalara, live FX
- ✅ `messaging`: NewFromConfig(memory), Publish/Consume options helpers, ErrQueueFull, ResilientConsumer, dedup TOCTOU, wrapper tests
- ✅ `compute`: EC2/GCE/Docker adapters, k8s SPDY Exec + Stats Unimplemented, Azure Functions/VM scaffolds
- ✅ `cloud`: remote libvirt, Firecracker, Redfish/IPMI, controlplane instance create/bind APIs
- ✅ `telemetry`: `Init(ctx,cfg)`, SampleRate/Insecure, noop/stdout providers, MeterProvider (OTLP/noop/stdout), RecordError/SetStatus
- ✅ `resilience`: Hedge/Fallback/ExecuteT + env-tagged Config; CB+retry+timeout+bulkhead
- ✅ `cache`: Exists/MGet/MSet/Expire/GetTTL, NewFromConfig, miniredis conformance, InvalidatePrefix
- ✅ `streaming`: PutRecords + optional Consume (memory consumer)
- ✅ `analytics`: Event Sink + memory sink + WindowedUniqueness
- ✅ `ai` (critical): LLM `StreamChat` + memory streaming, `errors.go`/instrumented, context-first conversation memory, embedding/image memory adapters; softened dual `ai/llm` vs `genai/llm` ledger; Chat (not Generate) docs
- ✅ `test`: Suite self-tests + examples; StartPostgres/StartRedis Short-skip + t.Cleanup
- ✅ `auth` SAML: SP client interface + memory ACS/AuthnRequest stub (XML crypto Unimplemented)
- ✅ `ai` gateway + prompt: multi-provider `genai/gateway` fallback router; versioned `genai/prompt` template stub
- ✅ `ai` multimodal + evals + RAG: `llm.ContentPart`/Parts; memory+OpenAI paths; `genai/evals` EvalRunner/golden/LLM-judge; RAG↔vector+rerank; Textract OCR adapter
- ✅ `database` Neo4j HTTP graph + Weaviate vector adapters; `SearchWithOpts` metadata filter; ClickHouse implements `sql.SQL`
- ✅ `security`: Vault KV v2, AWS KMS Encrypt/Decrypt, Cloudflare WAF IP access rules
- ✅ `audit`: SQL/Postgres durable store, messaging fanout, hash-chain, retention/GDPR
- ✅ Module branding: `go.mod` + all imports → `github.com/chris-alexander-pop/go-hyperforge` (was `system-design-library`)
- ✅ `servicemesh/discovery/adapters/consul` HTTP agent/health API + httptest tests
- ✅ `iot`: CoAP stub protocol (`protocols/coap`) + `device/registry` interface/memory
- ✅ `algorithms/loadbalancing`: Maglev + P2C
- ✅ `workflow` memory engine: real Task/Wait state-machine execution + idempotency key
- ✅ `workflow` durable saga: StateStore (memory + file/json) + Resume/ResumeAll after crash
- ✅ `cloud/controlplane` etcd HTTP adapter for host inventory persistence
- ✅ `database/vector` HybridSearch (keyword metadata + vector score)
- ✅ `metering` Prometheus exporter adapter + CalculateCostMoney → commerce.Money
- ✅ `iot` awsiot behind root Client adapter; blob-backed OTA via pkg/storage/blob
- ✅ `algorithms/loadbalancing/healthaware` skips unhealthy nodes
- ✅ Deep wave: `concurrency` singleflight + adaptive WorkerPool; `events` Outbox→messaging; `cache` Redis Cluster Config; search Typesense/OpenSearch memory stubs; `storage/file` local FS; `api/openapi` stub; errors ABORTED/FAILED_PRECONDITION + FromHTTP/FromGRPC

---

## Completeness scores (review snapshot)

| Package | Score | Notes |
|---------|------:|-------|
| messaging | 71→82 | Factory/options/ErrQueueFull/ResilientConsumer/tests landed |
| database | 62→improved | Neo4j+Weaviate adapters; vector filters; ClickHouse sql.SQL; Cassandra still open |
| auth | 57→improved | Session/MFA/JWT; OAuth2 AS; SMS/email MFA; Apple social; SAML skeleton |
| cache | 60→improved | Exists/MGet/MSet/Expire/TTL; NewFromConfig; miniredis; Cluster Config |
| logger | 58→improved | Init/Async/Shutdown/redact fixed |
| errors | 58→improved | Codes/IsCode/Wrap/FromHTTP/FromGRPC |
| datastructures | 58 | Broad catalog; many stubs / low reuse |
| communication | 58 | Ready: root drivers/errors/resilience, html/text templates, adapter tests |
| data | 62 | Search+Suggest; Typesense/OpenSearch memory stubs; bigdata errors/instrumented |
| compute | 52→78 | EC2/GCE/Docker + k8s Exec; Azure VM/Functions scaffolds |
| concurrency | 58→improved | singleflight + adaptive WorkerPool option |
| network | 50* | LB/DNS/CDN/APIGW/IP instrumented; cloud adapters reserved |
| api | 48→improved | OpenAPI helper stub; GraphQL; standards weak in places |
| test | 45→improved | Suite self-tests/examples; containers Short-skip + Cleanup |
| commerce | 42→78 | Money + payment depth; billing proration+dunning; TaxJar/Avalara; live FX |
| events | 42→improved | Config/errors/lifecycle + Outbox messaging bridge |
| workflow | 38→improved | Task/Wait SM + distlock/events |
| algorithms | 38→improved | Maglev/P2C; many educational stubs remain |
| cloud | 38→72 | Libvirt/Firecracker/Redfish/IPMI + instance bind APIs |
| telemetry | 36→improved | OTLP/noop/stdout traces+metrics MeterProvider |
| ai | 36→improved | StreamChat/gateway/prompt; multimodal Parts; evals; RAG↔vector/rerank; Textract |
| analytics | 32→improved | HLL + event Sink + windowed uniqueness |
| validator | 32→improved | Interface/errors/instrumented; config routes through it |
| audit | 34→improved | SQL/Postgres + messaging fanout; hash-chain; GDPR/retention |
| security | 30* → improved | Vault KV v2, AWS KMS, Cloudflare WAF; scanners still open; PQC experimental |
| servicemesh | 25*→improved | Discovery OK + Consul; CB/RL facades |
| storage | 45*→improved | Blob Store parity; file local FS adapter |
| resilience | 75→improved | Hedge/Fallback/ExecuteT + env Config; CB+retry+timeout+bulkhead |

\*Approximate where review used checklist form without a single headline score.

---

## Cross-cutting (all packages)

- [ ] 🔗 Use `pkg/errors` everywhere (no `fmt.Errorf` / stdlib `errors.New` for domain errors)
- [ ] 🔗 Use `pkg/concurrency.SmartMutex` / `SmartRWMutex` instead of `sync.Mutex` / `RWMutex`
- [ ] 🔗 Use `pkg/resilience` for all external I/O (CB + retry); delete reinvented wrappers
- [x] 🔗 Use `pkg/validator` for Config validation; fix `pkg/config` to call it
- [ ] 🔗 Use `pkg/algorithms/*` and `pkg/datastructures/*` instead of local copies (Dijkstra PQ, LB selection, etc.)
- [ ] 🔗 Emit domain events via `pkg/events` where standards §9 apply
- [ ] ❌ Package `errors.go` + `instrumented.go` + `adapters/memory/` where PACKAGE_STANDARDS require them
- [ ] ❌ Interface tests / `pkg/test` suites for every adapter surface
- [x] ✅ Align module branding (`go.mod` + imports → `github.com/chris-alexander-pop/go-hyperforge`; rename done)
- [ ] ⚠️ Demote false ✅ in `pkg/TODO.md` to 🔄/❌ to match this backlog

---

## 1. Core foundation

### `pkg/errors` (~58 → improved)
- [x] ✅ Codes: `DEADLINE_EXCEEDED`, `UNAVAILABLE`, `RESOURCE_EXHAUSTED`, `CANCELED`, `ABORTED`, `FAILED_PRECONDITION`
- [x] ✅ `IsCode(err, code)` / `Code(err)` helpers
- [x] ✅ `Wrap` preserving `*AppError` (or `WrapCode`)
- [x] ✅ HTTP/gRPC mapping for custom/domain codes; reverse `FromHTTP` / `FromGRPC`
- [x] 🔗 Wire `HTTPStatus`/`GRPCStatus` into `pkg/api/rest` and `pkg/api/grpc`
- [x] ✅ Full test matrix for helpers + wrapped errors (including FromHTTP/FromGRPC)

### `pkg/logger` (~58 → improved)
- [x] ✅ Fix `Init` double-wrap of handler stack
- [x] ✅ Trace correlation with default `Async=true` (attrs before queue / copy span IDs)
- [x] ✅ `Shutdown(ctx)` flush for AsyncHandler
- [x] ✅ Redact `WithAttrs` / bound attrs
- [ ] ❌ Bootstrap: apps must call `Init`; examples in templates/services
- [x] ✅ Tests for Init layering, Trace+Async, WithAttrs leak

### `pkg/config` (~28 → improved)
- [x] 🔗 Route validation through `pkg/validator` (not raw playground)
- [x] ✅ Typed `AppError`s (`InvalidArgument` / `Internal`) instead of unstructured `Wrap`
- [x] ✅ `LoadFrom(path)` / options; multi-format; secrets integration
- [ ] ❌ In-repo adoption (`config.Load` unused outside itself)
- [x] ✅ Failure-path tests

### `pkg/validator` (~32 → improved)
- [x] ✅ Interfaces + `errors.go` + `instrumented.go`
- [x] ✅ Map failures to `errors.InvalidArgument`
- [x] ✅ Context-first APIs; `AllowedTags` retained for sanitizer config
- [x] ✅ Tests for slug/phone/SQL/command/SanitizeMap

### `pkg/telemetry` (~36 → improved)
- [x] ✅ Adapter-isolated exporters; noop/stdout for tests (`Provider` + `adapters/noop`, `adapters/stdout`)
- [x] ✅ Configurable sampler (`SampleRate`) + TLS (`Insecure` opt-in; not hard-coded AlwaysSample + Insecure)
- [x] ✅ `Init(ctx, cfg)`; shared `RecordError` / `SetStatus` helpers
- [x] ✅ Metrics `MeterProvider` alongside traces (OTLP / noop / stdout); `Meter(name)`; `DisableMetrics`
- [x] ✅ Deterministic tests (noop/stdout; no hang on collector)

### `pkg/test` (~45 → improved)
- [x] ✅ Self-tests + `example_test.go`; StartPostgres/StartRedis skip on `-short` + `t.Cleanup` (idempotent terminate)
- [ ] ❌ Drive adoption in cache/messaging/events/resilience/logger/api

### `pkg/resilience` (~75 → improved)
- [x] ✅ Breaker/Retrier interfaces + `instrumented.go` + `errors.go` (UNAVAILABLE/RESOURCE_EXHAUSTED)
- [x] ✅ Real Timeout (`WithTimeout`) + semaphore Bulkhead via `pkg/concurrency`
- [x] ✅ Hedge / Fallback; typed `ExecuteT` / `RetryT` / `HedgeT` / `FallbackT`; env-tagged `Config`
- [x] ✅ Half-open `MaxRequests` (`ErrTooManyRequests`)
- [x] 🔗 Single CB source of truth vs `pkg/servicemesh/circuitbreaker` (thin facade)
- [x] ✅ Map circuit-open → UNAVAILABLE/503; bulkhead/half-open cap → RESOURCE_EXHAUSTED/429
- [x] ✅ Tests for WithTimeout, ExponentialBackoff, RetryWithCircuitBreaker, Bulkhead, MaxRequests, Hedge, Fallback, ExecuteT

### `pkg/concurrency` (~52 → improved)
- [ ] 🔗 Wrap/re-export `x/sync/semaphore` + `errgroup` instead of competing copies
- [x] ✅ Distlock: `AcquireWithRetry` uses `LockConfig`; Redis adapter uses `pkg/errors`; docs honest (single-instance SET NX, not Redlock)
- [x] 🔗 Wire `algorithms/concurrency/adaptive` into pools (`WithAdaptiveLimiter`)
- [x] ✅ Tests for semaphore cancel paths + distlock retry/cancel (pool/pipeline/runner/redis lock still thin)
- [x] ✅ `singleflight`-style coalesce helper (`concurrency.Group`)

### `pkg/events` (~42 → improved)
- [x] ✅ `Config`, `errors.go`, Unsubscribe, graceful Close
- [x] ✅ Bounded async via `pkg/concurrency.WorkerPool`; propagate ctx; surface handler errors
- [x] ✅ Outbox / messaging bridge helpers (standards §9.5)
- [x] ✅ Fan-out / Close / race / instrumented tests (outbox + memory bus)

---

## 2. Data & storage

### `pkg/cache` (~60 → improved)
- [x] ✅ Fix memory TTL=0 (“no expiration” persists)
- [x] ✅ ResilientCache / Instrumented: do not treat NotFound as failure
- [x] ✅ `errors.go`, `manager.go` (`NewFromConfig` + RegisterDriver), Config pool/TLS/timeouts
- [x] ✅ Exists/MGet/MSet/Expire/GetTTL; `InvalidatePrefix`; Bloom Warm remains
- [x] ✅ Redis Cluster (`Config.Cluster` / `Addrs` + `NewCluster`)
- [x] ✅ Redis conformance tests (miniredis)

### `pkg/database` (~62 → improved)
- [x] ✅ Multi-shard manager wiring `pkg/algorithms/consistenthash` into `GetShard` (`sql.NewSharded` + `sharding.ConsistentHash`)
- [x] 🔗 Replace `ops.WithRetry` with `pkg/resilience`
- [x] ✅ Adapters: Neo4j HTTP graph (`graph/adapters/neo4j`); Weaviate vector (`vector/adapters/weaviate`)
- [ ] ❌ Adapters still open: Cassandra KV, Neptune graph, Milvus vector
- [x] ✅ ClickHouse implements `sql.SQL`; vector `SearchWithOpts` metadata filter (memory/pinecone/weaviate)
- [x] ✅ Hybrid search (`HybridSearch` keyword metadata + vector score)
- [ ] ❌ Broader interface conformance tests across stores

### `pkg/storage` (~45 → improved)
- [x] ✅ GCS/Azure implement `blob.Store`; map S3 miss → NotFound
- [x] ✅ `blob/errors.go`; `pkg/resilience` on cloud I/O (`resilient.go`)
- [x] ✅ Docs demoted: block/archive/controller memory-only (cloud adapters not claimed)
- [x] ✅ `pkg/concurrency` in memory adapters; typed `pkg/events` payloads (`BlobEventPayload`)
- [x] ✅ Root `storage.go`; archive doc clarified (cold storage ≠ tar/zip)
- [x] ✅ Local/NFS-shaped `file` adapter (`file/adapters/local` real FS)
- [ ] ❌ Production adapters for block/archive/controller (still future work)

### `pkg/data` (~56 → improved)
- [x] ✅ Docs: top-level `etl` / `processing` marked planned-only (`data/doc.go`, `pkg/README`)
- [x] ✅ Search `Suggest` autocomplete on interface + memory; Typesense/OpenSearch memory stubs
- [x] ✅ Reuse `pkg/concurrency` (SmartRWMutex/SmartMutex) in search memory, mapreduce, DAG
- [x] ✅ Bigdata `errors.go` + instrumented logging; Spark docs honest (local spark-submit, not Connect)
- [ ] ❌ Real Typesense/OpenSearch HTTP clients; Snowflake adapters (still future work)

### `pkg/streaming` (~25 → improved)
- [x] ✅ Remove Pub/Sub duplication with `pkg/messaging` (Kinesis/EventHubs + memory only)
- [x] ✅ `errors.go`; `resilient.go` via `pkg/resilience`; root memory tests; BufferSize honored
- [x] ✅ Fix README: Kafka and Pub/Sub live under `messaging`, not `streaming`
- [x] ✅ `PutRecords` batch API; optional `Consumer` + memory consumer

### `pkg/analytics` (~32 → improved)
- [x] ✅ Event ingest model (`Sink` / `Event`) + memory sink
- [x] ✅ Redis HLL adapter (PFADD/PFCOUNT/PFMERGE); Merge on Tracker; precision 4–16
- [x] ✅ Windowed uniqueness helper (`WindowKey` / `WindowedUniqueness`)
- [ ] ❌ Exact counters / warehouse adapters (still future work)
- [x] ✅ Fix PACKAGE_STANDARDS §6.11 example (`memory.New` + Close/Merge)

### `pkg/metering` (~20 → improved)
- [x] ✅ Tests; `InstrumentedRater`; memory + Prometheus exporter adapters
- [x] 🔗 Wire to `pkg/events` (`EventedMeter`) + `pkg/commerce.Money` via `CalculateCostMoney`
- [ ] ❌ Period aggregation / rate-card mutation APIs; postgres adapter still open

---

## 3. Communication & API

### `pkg/messaging` (~71 → improved)
- [x] ✅ `manager.go` `NewFromConfig` (memory via RegisterDriver; other drivers documented / adapter `New`)
- [x] ✅ Wire PublishOption/ConsumeOption via `Publish`/`Consume` helpers + headers/context (no interface break)
- [x] ✅ Memory honors `BufferSize`; returns `ErrQueueFull` instead of silent drop
- [x] ✅ `ResilientConsumer` (retry/CB on handler failures); `ResilientBroker.Consumer` wraps it
- [x] ✅ Tests: instrumented/resilient/dedup, memory `ErrQueueFull`, dedup TOCTOU claim fix
- [x] ✅ Softened kafka `doc.go` TODO; clarify TLS/prefetch are adapter-Config fields
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
- [x] ✅ OpenAPI helpers stub (`pkg/api/openapi`)
- [ ] ❌ Echo↔stdlib middleware bridge utilities
- [ ] ❌ WebSocket rooms / upgrade-time auth; gRPC auth + stream error-mapping interceptors

---

## 4. Security & auth

### `pkg/auth` (~57 → improved)
- [x] ✅ OAuth2 authorization server interfaces + memory adapter (auth code / client credentials / refresh; not full OpenID Provider)
- [x] ✅ Cognito/Entra Verify via OIDC JWKS; GCP Login via Identity Toolkit REST; OIDC code exchange + memory exchanger
- [x] ✅ SMS/email MFA ChannelProvider (`mfa/adapters/sms|email|channel`) via `pkg/communication` Sender; Twilio/SendGrid path documented
- [x] ✅ Apple social provider (`endpoints.Apple` + id_token claims); client-secret JWT minting remains caller-owned
- [x] ✅ SAML SP client skeleton (`pkg/auth/saml`) + memory ACS/AuthnRequest adapter; `ValidateXMLSignature` → Unimplemented (full SSO crypto reserved)
- [x] ✅ Root `errors.go` sentinels; cloud vs root IdP adapters remain dual surfaces (documented)
- [x] ✅ EncryptionKey wired for session/MFA memory+redis; WebAuthn memory is a usable challenge-tracking test double (library adapter remains production path)

### `pkg/security` (~30 → improved)
- [x] ✅ Root `security.go` + domain `errors.go` (fraud/captcha/waf/scanning/secrets/kms/crypto) via `pkg/errors`
- [x] ✅ Crypto: `pkg/errors`, `crypto/subtle` compare, `InstrumentedEncryptor`, MemoryKeyProvider → `crypto/adapters/memory`
- [x] ✅ Secrets: `Rotate` + Config `Validate` (`pkg/validator`) + optional `EventedSecretManager` audit events
- [x] ✅ Captcha: `adapters/recaptcha` siteverify HTTP adapter + honest memory/docs
- [x] ✅ Softened docs vs reality; bridge note vs `pkg/auth` IdP
- [x] ✅ Vault KV v2 HTTP adapter (`secrets/adapters/vault`) + httptest tests
- [x] ✅ AWS KMS Encrypt/Decrypt (`crypto/kms/adapters/awskms`) + injectable API tests
- [x] ✅ Cloudflare WAF IP access rules (`waf/adapters/cloudflare`) + httptest tests
- [ ] ❌ Remaining: AWS WAF, GCP/Azure KMS, cloud secret managers, scanners, GuardDuty
- [x] 🔄 PQC marked **experimental** (demo Kyber + X25519); CIRCL/liboqs not in go.mod; Dilithium/ML-DSA absent
- [ ] 🔗 Broader hash/password reuse via crypto across auth (partial)

### `pkg/audit` (~34 → improved)
- [x] ✅ Durable `adapters/sql` + `adapters/postgres` (database/sql Append/Query)
- [x] ✅ Messaging fanout bridge (`adapters/messaging` via `pkg/messaging`)
- [x] ✅ Tamper-evident hash-chain (`Hash`/`PrevHash`, memory + SQL option, `VerifyChain`)
- [x] ✅ Retention `Purge` + GDPR `ExportByActor` / `EraseByActor` (`LifecycleStore`)
- [x] ✅ Asserting tests (memory lifecycle/chain, SQL sqlite, messaging fanout)
- [x] ✅ Field-name redaction + Auditor error returns (prior wave)

---

## 5. Infrastructure

### `pkg/network` (~50 → improved)
- [x] 🔗 Wire `pkg/algorithms/loadbalancing` into LB selection (memory `SelectTarget`)
- [x] ✅ `instrumented.go` + `errors.go` for cdn/apigateway/ip + root TCP/UDP
- [x] ✅ Softened cloud claims (Route53/CloudFront/etc. reserved; TODO demoted to 🔄)
- [x] 🔗 `pkg/concurrency.SmartRWMutex` in all memory adapters (cdn/apigateway/ip)
- [x] ✅ Memory adapter tests for cdn/apigateway/ip

### `pkg/compute` (~52 → 78)
- [x] ✅ VM adapters EC2 + GCE; Azure VM scaffold (Unimplemented); Docker Engine adapter
- [x] ✅ Fix k8s ID/name bug (Create returns pod name usable with Get); UID legacy fallback
- [x] ✅ k8s Exec via SPDY remotecommand; Stats returns clear Unimplemented (needs metrics-server)
- [x] ✅ Azure Functions scaffold (HTTP Invoke + ARM CRUD Unimplemented)
- [x] ✅ Optional `container.ResilientRuntime` via `pkg/resilience`
- [x] 🔗 `pkg/concurrency.SmartRWMutex` in memory adapters; package sentinels
- [x] ✅ Root `compute.go`; docs clarify vs `pkg/cloud`

### `pkg/cloud` (~38 → 72)
- [x] ✅ Remote libvirt JSON/HTTP (pure Go, no CGO); Firecracker unix/HTTP API; Redfish + IPMI BMC power
- [x] ✅ Control-plane instance APIs (create/bind/unbind/list + capacity reservation)
- [x] ✅ etcd HTTP controlplane adapter for host inventory persistence (`adapters/etcd`)
- [ ] ❌ PXE imaging + postgres controlplane driver still open
- [x] ✅ Real scheduler strategies: binpack / spread / random (memory adapter)
- [x] ✅ Shared vocabulary note vs `pkg/compute` in docs
- [x] ✅ Tests for controlplane / provisioning / scheduler memory adapters + new adapters

### `pkg/servicemesh` (~25 → improved)
- [x] 🔗 **Thin-wrap** circuitbreaker → `pkg/resilience`
- [x] 🔗 **Thin-wrap** ratelimit → `pkg/algorithms/ratelimit` (+ `pkg/api/ratelimit`)
- [x] ✅ Consul HTTP discovery adapter (`adapters/consul`) + httptest tests; etcd/K8s still open
- [ ] ❌ Mesh: mTLS, retry reuse, honest docs; etcd/K8s discovery

### `pkg/storage` — see Data & storage

---

## 6. Domain & enterprise

### `pkg/commerce` (~42 → improved)
- [x] ✅ Root `commerce.go`; shared `Money` (int64 minor units, no float64)
- [x] ✅ Payment webhooks (Stripe HMAC + PayPal verifier), Authorizer auth/capture/void, Charge idempotency; Braintree claim dropped
- [x] ✅ Billing Plan catalog + Upgrade with proration + `StatusPastDue` via MarkPastDue; ProcessDunning (invoice→past_due); memory plan catalog
- [x] 🔗 `pkg/resilience` on Stripe/PayPal (+ ResilientProvider); `SmartRWMutex` in memory adapters
- [x] ✅ Domain events (`NewEventedProvider`); webhook + money + memory billing unit tests
- [x] ✅ TaxJar + Avalara HTTP adapters (`tax/adapters/taxjar`, `tax/adapters/avalara`) with httptest tests
- [x] ✅ Live FX `LiveRateProvider`/`Converter` via `currency/adapters/openexchangerates` (OER + Frankfurter; optional `pkg/cache`)

### `pkg/enterprise` (~24 → improved)
- [x] ✅ Standards skeleton: instrumented, adapters/memory, errors, eventsource tests
- [x] 🔗 Bridge eventsource → `pkg/events` (`evented.go`); messaging noted in docs
- [ ] ❌ Projection runner; durable store
- [ ] ❌ Demote TODO ✅ → 🔄 where still overclaiming

### `pkg/workflow` (~38 → improved)
- [x] ✅ Memory engine Task/Wait state-machine execution + IdempotencyKey; timeout still honored on empty/legacy path
- [x] 🔗 Scheduler + `pkg/concurrency/distlock`; saga + `pkg/events`/`messaging`
- [x] ✅ Durable saga (`StateStore` memory + file/json; `DurableExecutor.Resume` / `ResumeAll`)
- [ ] ❌ Real cron; cloud adapter completeness
- [ ] ❌ Saga/scheduler instrumented + interfaces

### `pkg/iot` (~28 → improved)
- [x] ✅ Root Client/Updater interfaces + memory adapters + instrumented + tests
- [x] 🔗 `pkg/resilience` for OTA downloads; `pkg/concurrency` for MQTT/memory
- [x] ✅ MQTT WaitTimeout bug fixed; OTA semver via `golang.org/x/mod/semver`
- [x] ✅ CoAP stub (`protocols/coap`) + device registry interface/memory
- [x] ✅ AWS IoT behind root Client interface (`adapters/awsiot.NewAdapter`); blob-backed OTA (`device/ota.BlobUpdater`)
- [x] ✅ Demoted TODO overclaims

### `pkg/web3` (~22)
- [x] ✅ Interfaces + adapters/memory + instrumented + tests
- [x] ✅ Softened WalletConnect / DID claims; race-safe SIWE nonces
- [ ] 🔄 SDK isolation under adapters (ethereum/ipfs still concrete scaffolds)

---

## 7. AI / algorithms / datastructures

### `pkg/ai` (~36 → improved)
- [x] ✅ LLM `StreamChat` on `genai/llm.Client` + memory adapter streaming (`StreamFromChat` fallback for cloud adapters)
- [x] ✅ `instrumented.go` + `errors.go` for genai/llm; context-first conversation `memory` APIs
- [x] ✅ Memory adapters for embedding + image generation
- [x] ✅ Softened dual `ai/llm` vs `genai/llm` ledger in `pkg/TODO.md`; fixed Generate vs Chat docs
- [x] ✅ `genai/gateway` multi-provider `llm.Client` router with ordered fallback + memory tests
- [x] ✅ `genai/prompt` versioned template store stub (`{{key}}` render) + memory adapter
- [x] ✅ Multimodal `Message.Parts` / `ContentPart`; conversation memory `AddUserParts`; OpenAI + memory adapter paths; tests
- [x] ✅ `genai/evals`: `EvalRunner`, golden set, exact-match + LLM-as-judge (memory-backed tests)
- [x] ✅ RAG ↔ `pkg/database/vector` + `pkg/database/rerank` (`WithReranker`, `RetrieveResults` + metadata filter)
- [x] ✅ OCR Textract cloud adapter (+ `ocr/errors.go`); vision Rekognition already present
- [ ] ❌ Fuller prompt engine; speech cloud adapters beyond stubs
- [ ] ❌ instrumented/errors/memory for *all* remaining AI capabilities (ml depth)

### `pkg/algorithms` (~38 → improved)
- [x] ✅ Implement standards-cited `search/binarysearch`, `graph/bfs`, `graph/dfs` (+ tests)
- [x] ✅ Soften Raft/Paxos/Chord/SWIM/Louvain docs as educational stubs; DistLimiter uses cache store
- [x] ✅ Sliding window counter (weighted prev+curr windows); Local remains exact log
- [x] 🔗 Dijkstra/A* reuse `pkg/datastructures/heap`; shared `algorithms/graph` types
- [x] ✅ Maglev + P2C loadbalancing; health-aware balancer (`healthaware`)
- [ ] ❌ Sticky LB still open
- [ ] ❌ Finish Raft/Paxos/Chord/SWIM/Louvain beyond educational stubs

### `pkg/datastructures` (~58)
- [x] ✅ Tests for ARC/CRDT/roaring/cuckoo/scalable/graph/DAG; G-Set CRDT implemented
- [x] ✅ Honest docs (drop Consistent Hashing/Red-Black; G-Set real; root doc softened)
- [ ] 🔗 Drive reuse into algorithms/cache/workflow (stop local PQs)
- [x] ✅ Quarantine placeholders as experimental (tdigest, histogram, disruptor, hllpp, roaring)

---

## Suggested implementation order (for agents)

1. **Foundation correctness:** logger Init/trace, errors codes/Wrap/IsCode, config→validator, cache TTL + miss semantics
2. **Reuse cleanup:** servicemesh wraps resilience/algorithms; network uses loadbalancing algos; database uses resilience; streaming vs messaging boundary
3. **Standards skeleton:** events Config/errors/lifecycle; enterprise/iot/web3/metering tests + memory adapters
4. **Catalog depth:** auth OAuth2 polish; storage file/block/archive cloud adapters; AI gateway/streaming
5. **Docs honesty:** `pkg/TODO.md` status pass; `pkg/README.md` maturity notes; package `doc.go` overclaims

---

## Review artifacts

Reviews were produced by parallel `cursor-grok-4.5-high` explore subagents, one per top-level `pkg/*` package, against `pkg/PACKAGE_STANDARDS.md`, `pkg/README.md`, `pkg/TODO.md`, and `services/SERVICE_CATALOG.md`.
