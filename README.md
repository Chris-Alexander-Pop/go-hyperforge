# ⚡ go-hyperforge

**Production-ready Go libraries and microservices for building apps fast. 130+ services, zero boilerplate.**

---

## 📂 Project Structure

- [`pkg/`](pkg/) — **Reusable Libraries**: Config, Logger, Database, Events, Resilience, and 20+ more
- [`services/`](services/) — **Reference Microservices**: 130 production-ready service implementations
- [`templates/`](templates/) — **Service Starters**: Scaffolding for REST, gRPC, and Worker services
- [`roadmap/`](roadmap/) — **Documentation**: Feature breakdowns and future plans

---

## 🚀 Quick Start

```bash
# Clone
git clone https://github.com/chris-alexander-pop/go-hyperforge.git
cd go-hyperforge

# Start infrastructure (Postgres, Redis, NATS)
make up

# Run tests
make test
```

---

## 📦 What's Included

### Libraries (`pkg/`)
| Package | Purpose |
|---------|---------|
| `auth` | JWT/OAuth2 authentication |
| `database` | Multi-database adapters (Postgres, MySQL, MongoDB, Redis) |
| `messaging` | NATS, Kafka, RabbitMQ abstractions |
| `events` | Event bus and pub/sub |
| `resilience` | Circuit breakers, retries, rate limiting |
| `telemetry` | OpenTelemetry tracing |
| `cache` | Redis, in-memory caching |
| `secrets` | Vault, AWS Secrets Manager |
| ... | [See all 26 packages](pkg/) |

### Services (`services/`)
130 production-ready microservices across:
- **Identity**: auth, user, permission, identity-provider
- **Communication**: notification, email, sms, push, chat
- **Infrastructure**: gateway, discovery, config, load-balancer
- **E-Commerce**: product, cart, order, payment, inventory
- **AI/Agents**: agent-runtime, llm-gateway, vector-search, embedding
- **And 100+ more**... [See full catalog](services/SERVICE_CATALOG.md)

---

## 🧪 Testing

```bash
make test        # Unit tests
make test-cover  # With coverage
make check       # Lint + vet + test
```

---

## 📄 License

MIT
