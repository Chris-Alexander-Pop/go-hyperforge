# Hyperforge Service Conventions

Conventions for microservices under `services/`. Packages in `pkg/` remain the capability layer; services are thin HTTP processes that compose those packages.

## Directory layout

```
services/<name>/
  cmd/<name>/main.go     # process entrypoint
  server/                # HTTP API + typed env Config (importable)
  internal/store/        # persistence adapters (memory first)
  internal/<domain>/     # service logic when non-trivial
```

Shared helpers live in [`services/platform`](../services/platform) (bootstrap only — not a framework).

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

| Service   | Default port | Role                                      |
|-----------|--------------|-------------------------------------------|
| gateway   | `8080`       | Edge entry; JWT verify; reverse proxy     |
| auth      | `8081`       | Register / login; issues JWTs             |
| user      | `8082`       | User profiles; trusts `X-User-ID`         |

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
