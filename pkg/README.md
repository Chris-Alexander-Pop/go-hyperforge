# Package Directory

> System Design Library - Go packages for building production systems

## Readiness

Capability packages are feature-complete enough to define Hyperforge services (interfaces,
memory adapters, cloud drivers, resilient/evented wrappers, and conformance suites).
Remaining depth varies by adapter — see [`TODO.md`](./TODO.md) for ✅ / 🔄 / ❌ status.

---

## Quick Navigation

| Domain | Packages | Description |
|--------|----------|-------------|
| **Core** | [errors], [logger], [config], [validator], [concurrency], [resilience], [events], [telemetry], [test] | Foundational utilities |
| **Data** | [cache], [database], [storage], [data], [streaming] | Data storage and processing |
| **Communication** | [messaging], [communication], [api] | Messaging and API tools |
| **Security** | [auth], [security] | Authentication, IAM, crypto, secrets |
| **Infrastructure** | [network], [compute], [cloud], [servicemesh] | Infrastructure abstractions |
| **Domain** | [commerce], [workflow], [analytics], [audit], [metering], [enterprise] | Business domain patterns |
| **AI/ML** | [ai], [iot], [web3] | Emerging tech integrations |
| **Algorithms** | [datastructures], [algorithms] | Data structures and algorithms |

---

## Package Details

### Core Infrastructure

| Package | Description |
|---------|-------------|
| `errors` | Structured error handling with HTTP/gRPC status mapping |
| `logger` | slog-based logging with OpenTelemetry trace correlation |
| `config` | Environment-based configuration loading |
| `validator` | Input validation and sanitization |
| `concurrency` | Mutexes, semaphores, worker pools, distributed locks |
| `resilience` | Circuit breakers, retry, timeout, bulkhead |
| `events` | Event bus for domain events |
| `telemetry` | OpenTelemetry setup utilities |
| `test` | Test suite helpers |

### Data & Storage

| Package | Subpackages |
|---------|-------------|
| `cache` | memory, redis |
| `database` | sql, document, vector, kv, graph, timeseries |
| `storage` | blob, file, block, archive, controller |
| `data` | search, bigdata (etl/processing planned) |
| `streaming` | Kinesis, EventHubs (Pub/Sub and Kafka live under `messaging`) |

### Security

| Package | Subpackages |
|---------|-------------|
| `auth` | jwt, oauth2 (AS interfaces + memory), oidc (verify + code exchange), mfa, session, webauthn, social, cloud |
| `security` | iam, crypto, secrets, fraud, captcha, waf, scanning |

### Infrastructure

| Package | Subpackages |
|---------|-------------|
| `network` | loadbalancer, dns, dhcp, cdn, apigateway, ip, firewall, sdn |
| `compute` | vm, container, serverless |
| `cloud` | controlplane, hypervisor, provisioning, scheduler |

### AI & ML

| Package | Subpackages |
|---------|-------------|
| `ai` | genai/llm, genai/image, genai/agents, ml, nlp, perception |

---

## Standards

All packages follow [PACKAGE_STANDARDS.md](./PACKAGE_STANDARDS.md):

- Interface-first design
- Decorator pattern for observability (`instrumented.go`)
- Memory adapters for testing (`adapters/memory/`)
- Context-first methods
- Unified configuration with `env` tags

[errors]: ./errors
[logger]: ./logger
[config]: ./config
[validator]: ./validator
[concurrency]: ./concurrency
[resilience]: ./resilience
[events]: ./events
[telemetry]: ./telemetry
[test]: ./test
[cache]: ./cache
[database]: ./database
[storage]: ./storage
[data]: ./data
[streaming]: ./streaming
[messaging]: ./messaging
[communication]: ./communication
[api]: ./api
[auth]: ./auth
[security]: ./security
[network]: ./network
[compute]: ./compute
[cloud]: ./cloud
[servicemesh]: ./servicemesh
[commerce]: ./commerce
[workflow]: ./workflow
[analytics]: ./analytics
[audit]: ./audit
[metering]: ./metering
[enterprise]: ./enterprise
[ai]: ./ai
[iot]: ./iot
[web3]: ./web3
[datastructures]: ./datastructures
[algorithms]: ./algorithms
