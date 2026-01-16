# Testing Strategy Roadmap

## 1. Unit Testing (`pkg/`)
We use standard Go testing + `testify` for assertions.
- [x] **Core Logic**: Config, JWT, Validation, Hashing, Errors (Implemented).
- [ ] **Mocks**: Use `mockery` or `gomock` to generate interfaces for DB/Events to test business logic isolation.
- [ ] **Table Driven Tests**: Standard pattern for testing multiple inputs.

## 2. Integration Testing
- [ ] **Docker Containers**: Use `testcontainers-go` to spin up real Postgres/Redis for tests.
    ```go
    // Example concept
    postgresC, _ := testcontainers.SetupPostgres(ctx)
    defer postgresC.Terminate(ctx)
    db := database.New(postgresC.ConnString)
    ```
- [ ] **Service-to-Service**: Use `httptest` to mock external API calls or spin up dependent services.

## 3. End-to-End (E2E) Testing
- [ ] **Black Box**: Run `docker compose up` and run a separate Go test suite that hits the public API Gateway.
- [ ] **Load Testing**: `k6` or `vegeta` scripts to verify performance under load.

## 4. Contract Testing
- [ ] **Pact**: Ensure API consumers (Frontend/Mobile) and Provider (Backend) agree on the JSON schema.

## 5. Continuous Testing
- [ ] **CI Pipeline**: Run `make test-cover` on every Pull Request.
- [ ] **Linting**: Enforce `golangci-lint` to catch bugs before they hit production.
