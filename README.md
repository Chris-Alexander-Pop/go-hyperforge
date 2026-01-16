# System Design Library 🏗️

**A comprehensive, modular, and opinionated library for building production-ready distributed systems in Go.**

This project serves as a foundational ecosystem for modern microservice development, implementing standardized adapters for infrastructure, resilience patterns, and security best practices.

## 📂 Project Structure

*   [`pkg/`](pkg/): **The Foundation Library**. Modular, reusable packages (Config, Logger, Database, Events).
*   [`templates/`](templates/): **Service Starters**. Scaffolding for REST, gRPC, and Worker services.
*   [`services/`](services/): **Reference Architectures**. Real implementations of common systems (e.g., Gateway, Auth).
*   [`roadmap/`](roadmap/): **Documentation**. Detailed breakdowns of supported features and future implementations.

## 🗺️ Roadmap

The [**MASTER ROADMAP**](MASTER_ROADMAP.md) provides a comprehensive index of all supported and planned features across 12 domains:

1.  [Infrastructure Fundamentals](roadmap/01_INFRASTRUCTURE_FUNDAMENTALS.md) (Scalability, Availability, Messaging)
2.  [The Database Universe](roadmap/02_DATABASE_UNIVERSE.md) (SQL, NoSQL, Vector)
3.  [AI & Machine Learning](roadmap/03_AI_AND_ML.md) (LLMs, Agents)
4.  [Concurrency & Resilience](roadmap/04_CONCURRENCY_AND_RESILIENCE.md) (Rate Limiting, Circuit Breakers)
5.  [New Templates & Packages](roadmap/05_NEW_TEMPLATES_AND_PACKAGES.md) (Serverless, CLI)
6.  [DevOps & Observability](roadmap/06_DEVOPS_AND_OBSERVABILITY.md) (CI/CD, Prometheus)
7.  [Security & Compliance](roadmap/07_SECURITY_AND_COMPLIANCE.md) (OIDC, RBAC)
8.  [Client Libraries](roadmap/08_CLIENT_LIBRARIES.md) (Web, Mobile)
9.  [Frontier Tech](roadmap/09_FRONTIER_TECH.md) (Web3, GameDev)
10. [Enterprise Patterns](roadmap/10_ENTERPRISE_PATTERNS.md) (DDD, CQRS)
11. [Testing Strategy](roadmap/11_TESTING_STRATEGY.md) (Unit, Integration, Contract)

## 🛠️ Quick Start

The project leverages **Docker Compose** for local infrastructure orchestration.

```bash
# 1. Clone the repository
git clone https://github.com/chris-alexander-pop/system-design-library.git
cd system-design-library

# 2. Start Infrastructure (Postgres, Redis, NATS)
make up

# 3. Run Tests (Requires Docker for TestContainers)
make test-cover
```

## 🧪 Testing

Testing is implemented using a unified `pkg/test` framework that wraps `testify` and `testcontainers-go`.

*   **Unit Tests**: Validate logic in isolation.
*   **Integration Tests**: Spin up ephemeral Docker containers (Postgres, Redis) to verify interactions.

```go
// Example Usage
func TestUserRepo(t *testing.T) {
    db := test.StartPostgres(t) // Configures ephemeral PG container
    repo := user.NewRepo(db)
    // ... assertions
}
```

## License

MIT
