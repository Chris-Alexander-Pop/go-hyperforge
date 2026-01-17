# Package Design Philosophy & Developer Guide

This document defines the architectural standards, file structure, and development workflow for the `system-design-library`. It serves as the "Rulebook" for any developer or LLM agent contributing to this codebase.

## 1. Core Principles (The "Why")

### Interface-First Design
**Define behavior, not implementation.**
- Every package must define its core capabilities as Go interfaces in the root file (`pkg/{name}/{name}.go`).
- Consumers depend on the interface, never the concrete struct.
- Allows trivial mocking and hot-swapping of backends.

### The Decorator Pattern for Observability
**Do not pollute business logic.**
- Never put `logger.Info` or `otel.Start` inside a core adapter (e.g., `s3.go`).
- Create an `Instrumented{Interface}` struct that wraps the implementation.
- This creates a clean separation of concerns: Logic vs. Insight.

### "Zero-Dependency" Testing
**Tests must run instantly.**
- Every package MUST provide an **in-memory** implementation (`pkg/{name}/test`).
- No Docker containers required for standard `go test`.
- Use `sqlite` (memory), `miniredis`, or native Go maps.

### Unified Configuration
**One struct, all environments.**
- Use a single `Config` struct with `env` tags.
- Ensure feature parity: If SQL has TLS, NoSQL must have TLS. No "toy" drivers.

### Decoupled Dependencies
**Import only what you use.**
- Adapters live in separate packages (`pkg/{name}/adapters/{driver}/`).
- The root package imports ZERO external SDKs—only standard library and internal packages.
- Users explicitly import the adapter they need, pulling only that dependency.
- This prevents "mega-dependency" scenarios where using one feature imports all SDKs.
- Example: Importing `pkg/messaging/adapters/kafka` only pulls Sarama, not RabbitMQ or AWS SDKs.

## 2. Directory Structure Standard (The "Where")

Every package in `pkg/` must follow this exact structure:

```text
pkg/
└── {capability}/                 # e.g. "database", "blob"
    ├── {capability}.go           # INTERFACES & CONFIG (The Contract)
    ├── instrumented.go           # DECORATORS (Logging/Tracing wrapper)
    ├── manager.go                # SWITCHER (Factory/Manager logic)
    ├── errors.go                 # SENTINEL ERRORS
    ├── adapters/                 # CONCRETE IMPLEMENTATIONS
    │   ├── {driver1}/            # e.g. "postgres", "s3"
    │   │   └── {driver1}.go
    │   └── {driver2}/            # e.g. "memory", "redis"
    └── tests/                    # INTEGRATION TESTS
    │   └── {capability}_test.go  # Generic test suite run against ALL adapters
    └── plugins/                  # EXTENSIONS (Optional)
        └── {plugin_name}/
```

**Rules:**
1.  **Root is minimal**: Only Interfaces, Config, and Decorators go in root.
2.  **Adapters are isolated**: Each adapter gets its own folder to keep dependencies optional (e.g. don't import AWS SDK if using only Local).
3.  **Tests are generic**: Tests should accept the *Interface*, not a specific struct, running the same suite against Memory, S3, and GCS.

---

## 3. Plugin & Extension Philosophy

We prefer **Composition** and **Hooks** over inheritance.

### Middleware / Plugins
If a package needs extension (e.g. database change events), use an "Intercept" or "Plugin" approach:
- **GORM**: Use `db.Use(plugin)` with Callbacks (`AfterCreate`).
- **HTTP**: Use Middleware `func(next http.Handler) http.Handler`.
- **General**: Use the **Event Bus**.

### Event Integration
- Components should be loosely coupled via `pkg/events`.
- Do not hardcode "Email Service" calls inside "User Service".
- Instead: User Service publishes `user.created` -> Email Service subscribes to `user.created`.

---

## 4. The Agentic Workflow (The "How")

Follow this step-by-step process when adding a new feature or package:

1.  **Define the Interface**: Write `pkg/{name}/{name}.go`. What does this do? (e.g. `Upload`, `Download`).
2.  **Define the Config**: Add a `Config` struct with `env` tags.
3.  **Create the Memory Adapter**: Write `pkg/{name}/adapters/memory`. Implement the interface using generic Go maps/channels.
4.  **Write the Test Suite**: Write `pkg/{name}/tests/{name}_test.go`. Test against the Memory adapter.
5.  **Implement Real Adapters**: Add `adapters/s3`, `adapters/postgres`, etc.
6.  **Add Instrumentation**: Write `instrumented.go` to wrap the interface with Logging/OTel.
7.  **Verify**: Run the test suite against the Real adapter (if keys provided) or rely on the logic parity with Memory adapter.

### Coding Standards

#### Internal Shared Packages (MUST USE)
The following packages already exist in `pkg/` and **MUST** be used for consistency. Do NOT create custom implementations or import external alternatives.

| Package | Purpose | Usage |
|---------|---------|-------|
| `pkg/errors` | Structured error handling with error codes | `errors.New("CODE", "message", err)`, `errors.Wrap(err, "context")` |
| `pkg/logger` | Structured logging (slog-based) | `logger.L().InfoContext(ctx, "message", "key", value)` |
| `pkg/telemetry` | OpenTelemetry setup & tracing | Tracer initialization, span creation |
| `pkg/config` | Environment-based configuration | `env` struct tags, config loading |
| `pkg/validator` | Input validation | Struct validation with tags |

#### General Rules
- **Errors**: Always use `pkg/errors`. Create error constructors (e.g., `ErrNotFound(err)`) that wrap `errors.New(code, msg, err)`.
- **Logging**: Always use `pkg/logger`. Never use `fmt.Println` or `log.Printf` in production code.
- **Tracing**: Use `otel.Tracer("pkg/{name}")` for creating tracers. Spans should be created in `instrumented.go`.
- **Context**: Every I/O method MUST accept `context.Context` as the first argument.
- **Parity**: If you add `SetMaxOpenConns` for SQL, check if Redis client has a similar setting (e.g., `PoolSize`) and map it.

