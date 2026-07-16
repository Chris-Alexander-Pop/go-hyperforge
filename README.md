# ⚡ go-hyperforge

[![CI](https://github.com/chris-alexander-pop/go-hyperforge/actions/workflows/ci.yml/badge.svg)](https://github.com/chris-alexander-pop/go-hyperforge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/chris-alexander-pop/go-hyperforge)](https://goreportcard.com/report/github.com/chris-alexander-pop/go-hyperforge)
[![Go Reference](https://pkg.go.dev/badge/github.com/chris-alexander-pop/go-hyperforge.svg)](https://pkg.go.dev/github.com/chris-alexander-pop/go-hyperforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Production-ready Go libraries for building distributed systems. 34 packages, 67 microservices, zero boilerplate.**

---

## 📂 Project Structure

```
├── pkg/           # Reusable libraries (34 packages)
├── services/      # Microservices (67 runnable: identity + communication + commerce + platform)
├── templates/     # Service starters
└── docs/          # Documentation
```

---

## 🚀 Quick Start

```bash
# Install
go get github.com/chris-alexander-pop/go-hyperforge/pkg/...

# Or clone for development
git clone https://github.com/chris-alexander-pop/go-hyperforge.git
cd go-hyperforge
make setup  # Install tools + hooks
make up     # Start infrastructure
make check  # Run all quality gates
```

### Run the identity cluster locally

```bash
# Terminal 1 — user profiles (:8082)
go run ./services/user/cmd/user

# Terminal 2 — auth (:8081), posts profiles to user
go run ./services/auth/cmd/auth

# Terminal 3 — gateway (:8080)
go run ./services/gateway/cmd/gateway

# Register / login / me
curl -s -X POST localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"s3cret-pass","name":"Alice"}'

TOKEN=$(curl -s -X POST localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"s3cret-pass"}' | jq -r .access_token)

curl -s localhost:8080/v1/users/me -H "Authorization: Bearer $TOKEN"
```

Shared JWT settings default to `JWT_SECRET=dev-hyperforge-jwt-secret-change-me` on auth and gateway. See [Service Conventions](docs/services.md).

---

## 📦 Package Overview

| Domain | Packages |
|--------|----------|
| **Core** | [errors](pkg/errors), [logger](pkg/logger), [config](pkg/config), [validator](pkg/validator), [resilience](pkg/resilience), [events](pkg/events) |
| **Data** | [cache](pkg/cache), [database](pkg/database), [storage](pkg/storage), [data](pkg/data), [streaming](pkg/streaming) |
| **Comms** | [messaging](pkg/messaging), [communication](pkg/communication), [api](pkg/api) |
| **Security** | [auth](pkg/auth), [security](pkg/security) (iam, crypto, secrets) |
| **Infra** | [network](pkg/network), [compute](pkg/compute), [cloud](pkg/cloud) |
| **AI/ML** | [ai](pkg/ai) (genai, ml, nlp, perception) |

[See all packages →](pkg/README.md)

---

## 💡 Usage Examples

### Caching with Redis

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/redis"

cache := redis.New(redis.Config{Addr: "localhost:6379"})
cache.Set(ctx, "user:123", user, time.Hour)
```

### Circuit Breaker

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"

cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
    Name:             "api-call",
    FailureThreshold: 5,
})

err := cb.Execute(ctx, func(ctx context.Context) error {
    return callExternalAPI()
})
```

### Event Publishing

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/kafka"

bus := kafka.New(kafka.Config{Brokers: []string{"localhost:9092"}})
bus.Publish(ctx, &messaging.Message{Topic: "orders", Payload: data})
```

---

## 🧪 Testing

```bash
make test        # Unit tests
make test-cover  # With coverage report
make check       # Full quality gates (fmt, vet, lint, test)
```

### Benchmarks

```bash
go test -bench=. -benchmem ./pkg/cache/...
go test -bench=. -benchmem ./pkg/messaging/...
go test -bench=. -benchmem ./pkg/resilience/...
```

---

## 📖 Documentation

- [Package Standards](pkg/PACKAGE_STANDARDS.md) - Design patterns and conventions
- [Package Index](pkg/README.md) - All packages with descriptions
- [Service Conventions](docs/services.md) - Microservice layout and identity cluster
- [Service Catalog](services/SERVICE_CATALOG.md) - Planned microservices
- [Contributing](CONTRIBUTING.md) - Development workflow
- [Changelog](CHANGELOG.md) - Version history
- [Security](SECURITY.md) - Vulnerability reporting

---

## 🤝 Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) before submitting PRs.

```bash
make setup  # Install development tools
make check  # Run before pushing
```

---

## 📄 License

MIT - see [LICENSE](LICENSE)
