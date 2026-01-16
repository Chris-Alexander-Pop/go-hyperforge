# System Design Library - The Master Roadmap

This is the master plan to build the ultimate Go microservices ecosystem. It is divided into specialized domains located in the `roadmap/` directory.

## 📚 The Roadmap Collection

### 1. [Infrastructure Fundamentals](roadmap/01_INFRASTRUCTURE_FUNDAMENTALS.md)
*   **Concepts**: Scalability, Availability, CAP Theorem.
*   **Cloud**: AWS, GCP, Azure.
*   **Messaging**: Kafka, RabbitMQ, NATS.

### 2. [The Database Universe](roadmap/02_DATABASE_UNIVERSE.md)
*   **Concepts**: Sharding, Replication, Caching Strategies.
*   **SQL**: Postgres, MySQL, SQLite.
*   **NoSQL**: Mongo, Cassandra, DynamoDB.
*   **Vector**: pgvector, Pinecone.

### 3. [AI & Machine Learning](roadmap/03_AI_AND_ML.md)
*   **LLMs**: Unified interfaces for OpenAI, Anthropic, Ollama.
*   **Agents**: RAG pipelines, Tool use.

### 4. [Concurrency & Resilience](roadmap/04_CONCURRENCY_AND_RESILIENCE.md)
*   **Reliability**: Rate Limiters, Circuit Breakers, Consistent Hashing.
*   **Patterns**: Worker Pools, Distributed Locks (Redlock).
*   **Optimization**: Bloom Filters.

### 5. [New Templates & Packages](roadmap/05_NEW_TEMPLATES_AND_PACKAGES.md)
*   **Reference Arch**: URL Shortener, Chat System, Notification Service.
*   **Templates**: Serverless, CLI tools.

### 6. [DevOps & Observability](roadmap/06_DEVOPS_AND_OBSERVABILITY.md)
*   **CI/CD**: GitHub Actions, Semantic Release.
*   **Observability**: Prometheus, Grafana, Jaeger.

### 7. [Security & Compliance](roadmap/07_SECURITY_AND_COMPLIANCE.md)
*   **Auth**: OIDC, MFA, RBAC.
*   **Compliance**: Audit Logs, PII Redaction.

### 8. [Client Libraries](roadmap/08_CLIENT_LIBRARIES.md)
*   **Web**: React Hooks.
*   **Mobile**: iOS/Android SDKs.

### 9. [Frontier Tech](roadmap/09_FRONTIER_TECH.md)
*   **Web3**: Blockchain interaction.
*   **GameDev**: UDP Servers.

### 10. [Enterprise Patterns](roadmap/10_ENTERPRISE_PATTERNS.md)
*   **DDD**: Aggregates, Specs.
*   **CQRS**: Command Bus.

### 11. [Testing Strategy](roadmap/11_TESTING_STRATEGY.md)
*   **Framework**: `pkg/test` (TestContainers, Mocks).
*   **Types**: Unit, Integration, E2E, Contract.

