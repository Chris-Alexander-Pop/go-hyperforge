# Package Standards & Compliance Guide

> **The definitive rulebook for building high-quality, consistent packages in the System Design Library.**

This document defines the complete requirements for every package in `pkg/`. It supersedes and replaces the original `pkg-design-philosophy.md` with comprehensive, enforceable standards that cover architecture, file structure, code format, documentation, testing, and integration with core infrastructure packages.

---

## Table of Contents

1. [Core Architecture Principles](#1-core-architecture-principles)
2. [File Structure Standards](#2-file-structure-standards)
3. [File Naming Conventions](#3-file-naming-conventions)
4. [Code Format Standards](#4-code-format-standards)
5. [Documentation & Comment Standards](#5-documentation--comment-standards)
6. [Core Package Integration](#6-core-package-integration)
7. [Logging Level Standards](#7-logging-level-standards)
8. [Error Code Standards](#8-error-code-standards)
9. [Event Standards](#9-event-standards)
10. [Plugin & Extension Architecture](#10-plugin--extension-architecture)
11. [Testing Standards](#11-testing-standards)
12. [Code Quality Gates](#12-code-quality-gates)
13. [Compliance Checklist](#13-compliance-checklist)

---

## 1. Core Architecture Principles

### 1.1 Interface-First Design

Every package MUST define its core capabilities as Go interfaces.

**Rules:**
- The primary interface MUST be defined in `{package}.go` (the root file)
- Consumers depend on **interfaces**, never concrete types
- Interfaces enable trivial mocking, hot-swapping, and the Decorator pattern

```go
// ✅ CORRECT: Interface in root file
// pkg/cache/cache.go
type Cache interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Close() error
}

// ❌ WRONG: No interface, only concrete type
type RedisCache struct { ... }
```

### 1.2 The Decorator Pattern for Observability

**NEVER** put logging, tracing, or metrics inside core adapter implementations.

**Rules:**
- Business logic goes in adapters
- Observability goes in `instrumented.go` wrapper
- The wrapper takes the interface and decorates it with logging/tracing

```go
// ✅ CORRECT: Instrumented wrapper in instrumented.go
type InstrumentedCache struct {
    next   Cache           // The actual implementation
    tracer trace.Tracer
}

func (c *InstrumentedCache) Get(ctx context.Context, key string, dest interface{}) error {
    ctx, span := c.tracer.Start(ctx, "cache.Get", ...)
    defer span.End()
    
    logger.L().InfoContext(ctx, "cache get", "key", key)
    
    err := c.next.Get(ctx, key, dest)
    if err != nil {
        span.RecordError(err)
        logger.L().ErrorContext(ctx, "cache get failed", "key", key, "error", err)
    }
    return err
}

// ❌ WRONG: Logging/tracing in adapter
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
    logger.L().Info("getting key", "key", key)  // DON'T DO THIS
    // ...
}
```

### 1.3 Decoupled Dependencies

External SDKs MUST be isolated in adapter packages.

**Rules:**
- Root package imports ONLY standard library + internal `pkg/` packages
- Each adapter has its own package under `adapters/`
- Users explicitly import only the adapters they need

```go
// ✅ CORRECT: Root package has zero external SDK imports
// pkg/cache/cache.go
package cache

import (
    "context"
    "time"
)  // No external imports!

// ✅ CORRECT: Adapter package imports the SDK
// pkg/cache/adapters/redis/redis.go
package redis

import (
    "github.com/redis/go-redis/v9"  // SDK isolated here
    "github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)
```

### 1.4 Context-First Methods

Every I/O method MUST accept `context.Context` as the first parameter.

```go
// ✅ CORRECT
Get(ctx context.Context, key string) ([]byte, error)

// ❌ WRONG
Get(key string) ([]byte, error)
```

### 1.5 Unified Configuration

Every package MUST have a `Config` struct with `env` tags.

```go
type Config struct {
    Driver   string `env:"CACHE_DRIVER" env-default:"memory"`
    Host     string `env:"CACHE_HOST" env-default:"localhost"`
    Port     int    `env:"CACHE_PORT" env-default:"6379"`
    Password string `env:"CACHE_PASSWORD"`
    DB       int    `env:"CACHE_DB" env-default:"0"`
    
    // Pool configuration (parity across drivers)
    PoolSize    int           `env:"CACHE_POOL_SIZE" env-default:"10"`
    PoolTimeout time.Duration `env:"CACHE_POOL_TIMEOUT" env-default:"30s"`
}
```

**Parity Rule:** If one driver supports a config (e.g., TLS, PoolSize), ALL drivers must support it or explicitly document the gap.

---

## 2. File Structure Standards

### 2.1 Standard Package Layout

```
pkg/
└── {capability}/                    # e.g., "cache", "messaging", "blob"
    ├── {capability}.go              # [REQUIRED] Interfaces, Config, types
    ├── errors.go                    # [REQUIRED if errors] Sentinel errors, error constructors
    ├── instrumented.go              # [REQUIRED] Observability wrapper (logging/tracing)
    ├── manager.go                   # [OPTIONAL] Factory, switcher, or manager logic
    ├── doc.go                       # [OPTIONAL] Package-level documentation
    │
    ├── adapters/                    # [REQUIRED] Concrete implementations
    │   ├── memory/                  # [REQUIRED] In-memory adapter for testing
    │   │   └── memory.go
    │   ├── {driver}/                # e.g., redis/, postgres/, s3/
    │   │   └── {driver}.go
    │   └── ...
    │
    ├── plugins/                     # [OPTIONAL] Extension modules
    │   └── {plugin_name}/
    │       └── {plugin_name}.go
    │
    └── tests/                       # [RECOMMENDED] Integration test suite
        └── {capability}_test.go     # Tests run against ALL adapters
```

### 2.2 File Responsibilities

| File | Purpose | Required |
|------|---------|----------|
| `{capability}.go` | Primary interfaces, `Config` struct, core types | **Yes** |
| `errors.go` | Sentinel errors with `pkg/errors` constructors | If package has errors |
| `instrumented.go` | Decorator for logging/tracing all interface methods | **Yes** |
| `manager.go` | Factory pattern, driver switching, connection pooling | If multi-adapter |
| `adapters/memory/` | In-memory implementation for zero-Docker testing | **Yes** |
| `adapters/{driver}/` | Real implementations (Redis, Postgres, S3, etc.) | Per driver |
| `tests/` | Generic test suite run against ALL adapters | Recommended |
| `plugins/` | Optional extensions (middleware, hooks) | Optional |

### 2.3 Data Structure Package Layout

For packages in `pkg/algorithms`, `pkg/datastructures/`, and similar, use this simpler layout:

Here is an example layout for a data structure package, different variations of the same data structure can be in different directories within the {structure} directory. Same logic applies to algorithms packages:

```
pkg/datastructures/
└── {structure}/                     # e.g., "heap", "queue", "tree"
    ├── {structure}.go               # Implementation
    ├── {structure}_test.go          # Unit tests
    └── doc.go                       # [OPTIONAL] Package documentation
```

### 2.4 Domain Subpackage Layout

For domain packages that encompass **multiple distinct types or backends** with fundamentally different interfaces, use subpackages. Each subpackage follows the standard layout with its own adapters.

#### When to Use Subpackages

| Criteria | Flat Adapters | Subpackages |
|----------|---------------|-------------|
| Single interface, multiple backends | ✅ Use | ❌ Avoid |
| Multiple distinct interfaces | ❌ Avoid | ✅ Use |
| Backends share same operations | ✅ Use | ❌ Avoid |
| Backends have unique operations | ❌ Avoid | ✅ Use |
| Configuration differs significantly | ❌ Avoid | ✅ Use |

**Examples:**

| Package | Structure | Reason |
|---------|-----------|--------|
| `pkg/cache` | Flat adapters | One `Cache` interface, Redis/Memory backends have same API |
| `pkg/database` | Subpackages | SQL, Document, Vector, Graph have **different** interfaces |
| `pkg/storage` | Subpackages | Blob and File storage have **different** operations |
| `pkg/messaging` | Flat adapters | One `Bus` interface, Kafka/RabbitMQ backends have same API |

#### Subpackage Layout

```
pkg/{domain}/
├── {domain}.go              # [REQUIRED] Shared types, enums, driver constants
├── doc.go                   # [REQUIRED] Domain-level documentation
├── errors.go                # [OPTIONAL] Domain-wide errors (if any)
│
├── {type1}/                 # e.g., "sql", "document", "vector"
│   ├── {type1}.go           # [REQUIRED] Interface, Config for this type
│   ├── errors.go            # [REQUIRED if errors] Type-specific errors
│   ├── instrumented.go      # [REQUIRED] Observability wrapper
│   ├── doc.go               # [OPTIONAL] Type documentation
│   └── adapters/            # [REQUIRED] Implementations for this type
│       ├── memory/          # [REQUIRED] In-memory adapter for testing
│       │   └── memory.go
│       └── {driver}/        # Real implementations
│           └── {driver}.go
│
├── {type2}/                 # Another type with its own adapters
│   ├── {type2}.go
│   ├── instrumented.go
│   └── adapters/
│       └── ...
│
├── tests/                   # [RECOMMENDED] Integration tests
└── plugins/                 # [OPTIONAL] Extensions
```

#### Example: Database Domain

```
pkg/database/
├── database.go              # StoreType enum, Driver constants, shared utilities
├── doc.go                   # "Package database provides unified database access"
├── tls.go                   # Shared TLS helper
│
├── sql/
│   ├── sql.go               # SQL interface, Config
│   ├── errors.go            # ErrConnectionFailed, ErrQueryFailed
│   ├── instrumented.go      # InstrumentedSQL wrapper
│   └── adapters/
│       ├── memory/          # SQLite in-memory for testing
│       ├── postgres/
│       ├── mysql/
│       └── sqlite/
│
├── document/
│   ├── document.go          # Interface (Insert, Find, Update, Delete)
│   ├── errors.go
│   ├── instrumented.go
│   └── adapters/
│       ├── memory/          # In-memory doc store for testing
│       ├── dynamodb/
│       ├── mongodb/
│       └── firestore/
│
├── vector/
│   ├── vector.go            # Store interface (Search, Upsert, Delete)
│   ├── errors.go
│   ├── instrumented.go
│   └── adapters/
│       ├── memory/
│       └── pinecone/
│
└── timeseries/
    ├── timeseries.go        # Timeseries interface (Write, Query)
    └── adapters/
        ├── memory/
        ├── influxdb/
        └── timestream/
```

#### Root Package Responsibilities

When using subpackages, the root package (`{domain}.go`) should contain:

1. **Driver/Type Constants**: Enums for all supported drivers across all types
2. **Shared Types**: Types used by multiple subpackages (e.g., `StoreType`)
3. **Shared Utilities**: Helper functions used across subpackages (e.g., TLS loading)
4. **Re-exports** (optional): Convenience aliases for common types

```go
// pkg/database/database.go
package database

// Driver constants for all database types
const (
    // SQL Drivers
    DriverPostgres = "postgres"
    DriverMySQL    = "mysql"
    
    // Document Drivers
    DriverDynamoDB = "dynamodb"
    DriverMongoDB  = "mongodb"
    
    // Vector Drivers
    DriverPinecone = "pinecone"
)

// StoreType identifies the category of database
type StoreType string

const (
    StoreTypeSQL      StoreType = "sql"
    StoreTypeDocument StoreType = "document"
    StoreTypeVector   StoreType = "vector"
)
```

---

## 3. File Naming Conventions

### 3.1 General Rules

| Type | Convention | Example |
|------|------------|---------|
| Package directories | `lowercase` | `pkg/cache/`, `pkg/messaging/` |
| Go files | `lowercase.go` or `lowercase_suffix.go` | `cache.go`, `cache_test.go` |
| Test files | `{name}_test.go` | `validator_test.go` |
| Instrumented wrappers | `instrumented.go` | Always this exact name |
| Error files | `errors.go` | Always this exact name |
| Adapter packages | `adapters/{drivername}/` | `adapters/redis/`, `adapters/memory/` |
| Adapter files | `{drivername}.go` | `redis.go`, `memory.go` |

### 3.2 Type Naming

| Type | Convention | Example |
|------|------------|---------|
| Interfaces | `{Capability}` or `{Action}er` | `Cache`, `Bus`, `Publisher` |
| Structs implementing interface | `{Driver}{Capability}` | `MemoryCache`, `RedisBus` |
| Instrumented wrappers | `Instrumented{Interface}` | `InstrumentedCache`, `InstrumentedBus` |
| Config structs | `Config` | Always this exact name |
| Error types | `Err{Name}` for variables | `ErrNotFound`, `ErrTimeout` |

---

## 4. Code Format Standards

### 4.1 Import Organization

Imports MUST be organized into groups separated by blank lines:

```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "time"

    // 2. Internal pkg/ packages (alphabetical)
    "github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/logger"

    // 3. External dependencies (alphabetical)
    "github.com/redis/go-redis/v9"
    "go.opentelemetry.io/otel"
)
```

### 4.2 Struct Field Organization

Struct fields MUST be ordered by:
1. Required fields first
2. Optional/config fields second
3. Internal/private fields last

```go
type Config struct {
    // Required fields
    Driver string `env:"CACHE_DRIVER" validate:"required"`
    Host   string `env:"CACHE_HOST" validate:"required"`
    
    // Optional fields with defaults
    Port     int           `env:"CACHE_PORT" env-default:"6379"`
    Password string        `env:"CACHE_PASSWORD"`
    Timeout  time.Duration `env:"CACHE_TIMEOUT" env-default:"30s"`
    
    // Advanced/tuning fields
    PoolSize      int  `env:"CACHE_POOL_SIZE" env-default:"10"`
    EnableMetrics bool `env:"CACHE_METRICS" env-default:"true"`
}
```

### 4.3 Method Organization

Methods on a type MUST be organized in this order:

1. **Constructor** (`New`, `NewXxx`)
2. **Primary interface methods** (in interface order)
3. **Helper/utility methods** (private)
4. **Lifecycle methods** (`Close`, `Shutdown`)

```go
// 1. Constructor
func New(cfg Config) *RedisCache { ... }

// 2. Primary interface methods (same order as interface)
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error { ... }
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error { ... }
func (c *RedisCache) Delete(ctx context.Context, key string) error { ... }

// 3. Helper methods
func (c *RedisCache) serialize(v interface{}) ([]byte, error) { ... }
func (c *RedisCache) deserialize(data []byte, dest interface{}) error { ... }

// 4. Lifecycle
func (c *RedisCache) Close() error { ... }
```

### 4.4 Error Handling

**Always** use `pkg/errors` for error creation and wrapping:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// ✅ CORRECT: Use pkg/errors
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
    val, err := c.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return errors.NotFound("cache key not found", nil)
    }
    if err != nil {
        return errors.Internal("redis get failed", err)
    }
    
    if err := json.Unmarshal([]byte(val), dest); err != nil {
        return errors.InvalidArgument("failed to unmarshal cached value", err)
    }
    return nil
}

// ❌ WRONG: Using standard fmt.Errorf
return fmt.Errorf("failed to get: %w", err)
```

### 4.5 Constructor Functions

Every adapter MUST have a `New` constructor function:

```go
// New creates a new MemoryCache with default configuration
func New() *MemoryCache {
    return &MemoryCache{
        items: make(map[string]item),
        mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-cache"}),
    }
}

// NewWithConfig creates a new MemoryCache with custom configuration
func NewWithConfig(cfg Config) (*MemoryCache, error) {
    if err := validator.New().ValidateStruct(cfg); err != nil {
        return nil, errors.InvalidArgument("invalid config", err)
    }
    // ...
}
```

---

## 5. Documentation & Comment Standards

### 5.1 Package Documentation

Every package MUST have a package-level doc comment (in `{capability}.go` or `doc.go`):

```go
// Package cache provides a unified interface for caching with multiple backend support.
//
// This package supports the following backends:
//   - Memory: In-memory cache for testing and development
//   - Redis: Production-grade distributed cache
//
// Basic usage:
//
//     import "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/redis"
//     
//     cache := redis.New(cfg)
//     defer cache.Close()
//     
//     err := cache.Set(ctx, "key", value, time.Hour)
package cache
```

### 5.2 Interface Documentation

Every interface method MUST have a doc comment explaining:
- What it does
- Parameters (if not obvious)
- Return values (especially error conditions)

```go
// Cache defines the standard caching interface.
type Cache interface {
    // Get retrieves a value by key and unmarshals it into dest.
    // Returns errors.NotFound if the key does not exist or has expired.
    Get(ctx context.Context, key string, dest interface{}) error

    // Set stores a value with the given TTL.
    // A TTL of 0 means no expiration.
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

    // Delete removes a key from the cache.
    // Returns nil if the key does not exist.
    Delete(ctx context.Context, key string) error

    // Close releases all resources.
    // The cache should not be used after calling Close.
    Close() error
}
```

### 5.3 Struct Documentation

Config structs MUST document each field:

```go
// Config holds configuration for the Cache.
type Config struct {
    // Driver specifies the cache backend: "memory" or "redis".
    Driver string `env:"CACHE_DRIVER" env-default:"memory"`

    // Host is the cache server hostname.
    Host string `env:"CACHE_HOST" env-default:"localhost"`

    // Port is the cache server port.
    Port int `env:"CACHE_PORT" env-default:"6379"`

    // Password is the authentication password (optional).
    Password string `env:"CACHE_PASSWORD"`

    // PoolSize is the maximum number of connections in the pool.
    PoolSize int `env:"CACHE_POOL_SIZE" env-default:"10"`
}
```

### 5.4 Inline Comments

Use inline comments sparingly for:
- Non-obvious logic
- Workarounds/hacks (with issue references)
- Important invariants

```go
// ✅ GOOD: Explains non-obvious behavior
// Redis MULTI/EXEC is not atomic across cluster nodes, so we use Lua scripting
val, err := c.client.Eval(ctx, script, keys, args...).Result()

// ✅ GOOD: Documents known issue
// TODO(#123): This timeout should be configurable
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)

// ❌ BAD: States the obvious
// Increment the counter
counter++
```

---

## 6. Core Package Integration

Every package MUST integrate with the following core packages where applicable.

### 6.1 `pkg/errors` - Error Handling

**MANDATORY** for all error creation and wrapping.

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Use predefined error codes
errors.NotFound("user not found", nil)
errors.InvalidArgument("email is required", nil)
errors.Internal("database connection failed", err)
errors.Unauthorized("invalid token", nil)
errors.Forbidden("access denied", nil)
errors.Conflict("resource already exists", nil)

// Wrap errors with context
errors.Wrap(err, "failed to process request")

// Check errors
if errors.Is(err, targetErr) { ... }
errors.As(err, &appErr)

// Convert to HTTP/gRPC status
status := errors.HTTPStatus(err)
grpcStatus := errors.GRPCStatus(err)
```

### 6.2 `pkg/logger` - Logging

**MANDATORY** for all logging. NEVER use `fmt.Println` or `log.Printf`.

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/logger"

// Use context-aware logging (preferred)
logger.L().InfoContext(ctx, "processing request", 
    "user_id", userID,
    "action", "create",
)

logger.L().ErrorContext(ctx, "operation failed",
    "error", err,
    "key", key,
)

// Available levels
logger.L().DebugContext(ctx, "verbose debugging info", ...)
logger.L().InfoContext(ctx, "informational message", ...)
logger.L().WarnContext(ctx, "warning condition", ...)
logger.L().ErrorContext(ctx, "error occurred", ...)
```

### 6.3 `pkg/telemetry` - Tracing

**MANDATORY** in `instrumented.go` for all packages.

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

type InstrumentedCache struct {
    next   Cache
    tracer trace.Tracer
}

func NewInstrumentedCache(next Cache) *InstrumentedCache {
    return &InstrumentedCache{
        next:   next,
        tracer: otel.Tracer("pkg/cache"), // Always use pkg/{name}
    }
}

func (c *InstrumentedCache) Get(ctx context.Context, key string, dest interface{}) error {
    ctx, span := c.tracer.Start(ctx, "cache.Get", trace.WithAttributes(
        attribute.String("cache.key", key),
    ))
    defer span.End()

    err := c.next.Get(ctx, key, dest)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    }
    return err
}
```

### 6.4 `pkg/config` - Configuration Loading

Use for loading configs from environment:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/config"

var cfg cache.Config
if err := config.Load(&cfg); err != nil {
    return nil, errors.Internal("failed to load config", err)
}
```

### 6.5 `pkg/validator` - Input Validation

Use for validating structs and inputs:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/validator"

v := validator.New()

// Validate struct
if err := v.ValidateStruct(cfg); err != nil {
    return errors.InvalidArgument("invalid configuration", err)
}

// Validate single value
if err := v.ValidateVar(email, "required,email"); err != nil {
    return errors.InvalidArgument("invalid email", err)
}
```

### 6.6 `pkg/concurrency` - Thread Safety

Use for mutex and concurrency primitives:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"

type MemoryCache struct {
    items map[string]item
    mu    *concurrency.SmartRWMutex  // Use SmartRWMutex, not sync.RWMutex
}

func New() *MemoryCache {
    return &MemoryCache{
        items: make(map[string]item),
        mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
            Name: "memory-cache",
        }),
    }
}

func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // ...
}
```

### 6.7 `pkg/resilience` - Fault Tolerance

Use for circuit breakers and retries in adapters that make external calls:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"

type ResilientCache struct {
    next Cache
    cb   *resilience.CircuitBreaker
}

func NewResilientCache(next Cache) *ResilientCache {
    cfg := resilience.DefaultCircuitBreakerConfig("cache")
    return &ResilientCache{
        next: next,
        cb:   resilience.NewCircuitBreaker(cfg),
    }
}

func (c *ResilientCache) Get(ctx context.Context, key string, dest interface{}) error {
    return c.cb.Execute(ctx, func(ctx context.Context) error {
        return c.next.Get(ctx, key, dest)
    })
}
```

### 6.8 `pkg/events` - Event Publishing

Use for emitting domain events:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/events"

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }

    // Emit event instead of direct coupling
    event := events.Event{
        ID:        uuid.NewString(),
        Type:      "user.created",
        Source:    "user-service",
        Timestamp: time.Now(),
        Payload:   user,
    }
    return s.bus.Publish(ctx, "users", event)
}
```

### 6.9 `pkg/cache` - Caching

Use for adding caching layers:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/cache"

type CachedUserRepo struct {
    repo  UserRepository
    cache cache.Cache
}

func (r *CachedUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    var user User
    cacheKey := fmt.Sprintf("user:%s", id)
    
    // Try cache first
    if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
        return &user, nil
    }
    
    // Fallback to database
    u, err := r.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Populate cache
    _ = r.cache.Set(ctx, cacheKey, u, time.Hour)
    return u, nil
}
```

### 6.10 `pkg/test` - Testing Utilities

Use for test suites:

```go
import (
    "testing"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type CacheTestSuite struct {
    test.Suite
    cache cache.Cache
}

func (s *CacheTestSuite) SetupTest() {
    s.Suite.SetupTest()
    s.cache = memory.New()
}

func (s *CacheTestSuite) TearDownTest() {
    s.cache.Close()
}

func (s *CacheTestSuite) TestGetSet() {
    err := s.cache.Set(s.Ctx, "key", "value", time.Hour)
    s.NoError(err)

    var result string
    err = s.cache.Get(s.Ctx, "key", &result)
    s.NoError(err)
    s.Equal("value", result)
}

func TestCacheSuite(t *testing.T) {
    test.Run(t, new(CacheTestSuite))
}
```

### 6.11 `pkg/analytics` - Uniqueness Tracking

Approximate unique-element counts via HyperLogLog (not warehouse analytics).
Adapters: `adapters/memory` (pkg/datastructures/hyperloglog) and `adapters/redis` (PFADD/PFCOUNT/PFMERGE).

```go
import (
    "github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/memory"
)

tracker, err := memory.New(analytics.DefaultConfig())
if err != nil {
    return err
}
defer tracker.Close()

_ = tracker.Add(ctx, "visitors", userID)
count, err := tracker.Count(ctx, "visitors") // missing counter → (0, nil)
_ = tracker.Merge(ctx, "week", "monday")
_ = tracker.Reset(ctx, "visitors")
```

### 6.12 `pkg/crypto` - Encryption

Use for encryption/hashing:

```go
import "github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"

// Encryption
enc := crypto.NewAESEncryptor(key)
ciphertext, err := enc.Encrypt(plaintext)

// Hashing
hasher := crypto.NewHasher(crypto.SHA256)
hash := hasher.Hash(data)
```

### 6.13 `pkg/datastructures` - Data Structures

Use built-in data structures — **prefer reuse over local copies**:

| Need | Prefer |
|------|--------|
| Priority queue / Dijkstra-style PQ | `datastructures/heap` (`NewMinHeap`) |
| Probabilistic membership / negative cache | `datastructures/bloomfilter` |
| Bounded hot-key cache | `datastructures/lru` (or LFU/ARC) |
| FIFO / ring / delay | `datastructures/queue` |

Known reuse: `algorithms/graph/{dijkstra,astar,prim}` → heap; `cache` bloom wrapper + `messaging` dedup → bloomfilter; `workflow/adapters/memory` definition hot-cache → lru.

```go
import (
    "github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/bloomfilter"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/heap"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/lru"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/queue"
)
```

### 6.14 `pkg/algorithms` - Algorithms

Use built-in algorithms:

```go
import (
    "github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/search/binarysearch"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/graph/dfs"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/graph/bfs"
)
```

---

## 7. Logging Level Standards

This section defines **when** and **where** to use each log level to ensure consistent observability across all packages.

### 7.1 Log Level Definitions

| Level | Purpose | When to Use |
|-------|---------|-------------|
| **DEBUG** | Verbose internal state | Development only; detailed variable values, intermediate states |
| **INFO** | Normal operations | Key business events, successful completions, state transitions |
| **WARN** | Potential problems | Recoverable issues, deprecated usage, approaching limits |
| **ERROR** | Failures requiring attention | Unrecoverable errors, failed operations, system degradation |

### 7.2 Log Level Usage by Context

#### DEBUG - Development & Troubleshooting

Use for information only developers need during debugging. Should be disabled in production.

```go
// ✅ CORRECT: Useful for debugging
logger.L().DebugContext(ctx, "cache lookup details",
    "key", key,
    "ttl_remaining", ttl,
    "cache_size", len(c.items),
)

logger.L().DebugContext(ctx, "query execution plan",
    "sql", query,
    "params", params,
    "estimated_rows", plan.EstimatedRows,
)

// ❌ WRONG: Too verbose, use for things that happen per-request
logger.L().DebugContext(ctx, "entering function")  // No value
logger.L().DebugContext(ctx, "loop iteration", "i", i)  // Too noisy
```

#### INFO - Business Operations

Use for significant business events and successful operations. This is the default level in production.

```go
// ✅ CORRECT: Significant business events
logger.L().InfoContext(ctx, "user created",
    "user_id", user.ID,
    "email", user.Email,
)

logger.L().InfoContext(ctx, "order completed",
    "order_id", order.ID,
    "total", order.Total,
    "items_count", len(order.Items),
)

logger.L().InfoContext(ctx, "cache warmed",
    "entries_loaded", count,
    "duration_ms", elapsed.Milliseconds(),
)

// ✅ CORRECT in instrumented.go: Operation start/end
logger.L().InfoContext(ctx, "processing request",
    "operation", "cache.Get",
    "key", key,
)

// ❌ WRONG: Internal implementation details
logger.L().InfoContext(ctx, "acquired lock")  // Too low-level
logger.L().InfoContext(ctx, "checking cache")  // Use DEBUG
```

#### WARN - Potential Issues

Use when something unexpected happened but the operation can continue. Warns should be actionable.

```go
// ✅ CORRECT: Actionable warnings
logger.L().WarnContext(ctx, "rate limit approaching",
    "current_rate", rate,
    "limit", limit,
    "utilization_pct", float64(rate)/float64(limit)*100,
)

logger.L().WarnContext(ctx, "deprecated API called",
    "endpoint", endpoint,
    "recommended", newEndpoint,
    "removal_version", "v3.0",
)

logger.L().WarnContext(ctx, "retry succeeded after failure",
    "operation", "database.Query",
    "attempts", attempts,
    "last_error", lastErr.Error(),
)

logger.L().WarnContext(ctx, "connection pool exhausted, waiting",
    "pool_name", pool.Name,
    "max_connections", pool.MaxSize,
    "wait_count", waitCount,
)

// ❌ WRONG: Not actionable or not a problem
logger.L().WarnContext(ctx, "cache miss")  // Expected behavior
logger.L().WarnContext(ctx, "no results found")  // Normal empty result
```

#### ERROR - Operation Failures

Use when an operation failed and requires attention. Errors should be logged only once at the appropriate level.

```go
// ✅ CORRECT: Failed operations
logger.L().ErrorContext(ctx, "database connection failed",
    "host", cfg.Host,
    "error", err,
    "retry_count", retries,
)

logger.L().ErrorContext(ctx, "payment processing failed",
    "order_id", order.ID,
    "payment_method", method,
    "error", err,
)

logger.L().ErrorContext(ctx, "message publish failed",
    "topic", topic,
    "message_id", msg.ID,
    "error", err,
)

// ❌ WRONG: Errors that are expected or handled
logger.L().ErrorContext(ctx, "user not found")  // Use NotFound error, not ERROR log
logger.L().ErrorContext(ctx, "invalid input")  // Use WARN or return error
```

### 7.3 Logging Location Rules

| Location | Allowed Levels | Notes |
|----------|---------------|-------|
| **Adapters** (business logic) | NONE | Never log in adapters; use `instrumented.go` |
| **Instrumented wrappers** | INFO, ERROR | Log operation start, success metrics, failures |
| **Constructors/Init** | INFO, WARN, ERROR | Log initialization, config issues |
| **Background workers** | DEBUG, INFO, WARN, ERROR | All levels appropriate |
| **Tests** | DEBUG (via t.Log) | Use test logging, not logger.L() |

### 7.4 Structured Logging Keys

Use consistent key names across all packages:

| Key | Type | Usage |
|-----|------|-------|
| `error` | `error` | The error that occurred |
| `duration_ms` | `int64` | Operation duration in milliseconds |
| `operation` | `string` | Operation name (e.g., "cache.Get") |
| `user_id` | `string` | User identifier |
| `request_id` | `string` | Request correlation ID |
| `count` | `int` | Number of items/results |
| `key` | `string` | Cache/storage key |
| `topic` | `string` | Message/event topic |
| `driver` | `string` | Backend driver name |

```go
// ✅ CORRECT: Consistent keys
logger.L().InfoContext(ctx, "operation completed",
    "operation", "cache.Set",
    "key", key,
    "duration_ms", time.Since(start).Milliseconds(),
)

// ❌ WRONG: Inconsistent naming
logger.L().InfoContext(ctx, "done",
    "op", "set",           // Use "operation"
    "time_taken", elapsed, // Use "duration_ms"
    "k", key,              // Use "key"
)
```

---

## 8. Error Code Standards

This section defines **when** to use each error code and the conventions for error handling.

### 8.1 Error Code Definitions

| Code | HTTP | gRPC | When to Use |
|------|------|------|-------------|
| `NOT_FOUND` | 404 | NotFound | Resource doesn't exist |
| `INVALID_ARGUMENT` | 400 | InvalidArgument | Malformed request, validation failure |
| `INTERNAL` | 500 | Internal | Unexpected server errors, panics |
| `UNAUTHORIZED` | 401 | Unauthenticated | Missing or invalid authentication |
| `FORBIDDEN` | 403 | PermissionDenied | Valid auth but insufficient permissions |
| `CONFLICT` | 409 | AlreadyExists | Resource already exists, concurrent modification |

### 8.2 Error Code Usage Guide

#### NOT_FOUND

Use when a requested resource does not exist or has expired.

```go
// ✅ CORRECT: Resource not found
func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    user, err := r.db.FindByID(ctx, id)
    if err == sql.ErrNoRows {
        return nil, errors.NotFound("user not found", nil)
    }
    if err != nil {
        return nil, errors.Internal("database error", err)
    }
    return user, nil
}

// ✅ CORRECT: Cache miss
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
    item, ok := c.items[key]
    if !ok {
        return errors.NotFound("cache key not found", nil)
    }
    if time.Now().After(item.expiresAt) {
        return errors.NotFound("cache key expired", nil)
    }
    return json.Unmarshal(item.value, dest)
}
```

#### INVALID_ARGUMENT

Use when input validation fails or request is malformed.

```go
// ✅ CORRECT: Validation failures
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // Struct validation
    if err := s.validator.ValidateStruct(req); err != nil {
        return nil, errors.InvalidArgument("validation failed", err)
    }
    
    // Business rule validation
    if req.Age < 18 {
        return nil, errors.InvalidArgument("user must be 18 or older", nil)
    }
    
    // ...
}

// ✅ CORRECT: Malformed data
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    if key == "" {
        return errors.InvalidArgument("key cannot be empty", nil)
    }
    if ttl < 0 {
        return errors.InvalidArgument("TTL cannot be negative", nil)
    }
    // ...
}
```

#### INTERNAL

Use for unexpected errors that indicate bugs or system failures.

```go
// ✅ CORRECT: Unexpected failures
func (r *UserRepo) Create(ctx context.Context, user *User) error {
    _, err := r.db.Exec(ctx, insertQuery, user.ID, user.Email)
    if err != nil {
        // Unexpected database error
        return errors.Internal("failed to create user", err)
    }
    return nil
}

// ✅ CORRECT: Serialization failures (shouldn't happen)
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return errors.Internal("failed to serialize value", err)
    }
    // ...
}

// ❌ WRONG: Don't use for expected errors
return errors.Internal("user not found", nil)  // Use NotFound
return errors.Internal("invalid email", nil)   // Use InvalidArgument
```

#### UNAUTHORIZED

Use when authentication is missing or invalid.

```go
// ✅ CORRECT: Missing auth
func (m *AuthMiddleware) Handle(ctx context.Context, req Request) error {
    token := req.Header.Get("Authorization")
    if token == "" {
        return errors.Unauthorized("missing authorization header", nil)
    }
    
    claims, err := m.verifier.Verify(token)
    if err != nil {
        return errors.Unauthorized("invalid token", err)
    }
    
    // Continue with valid auth...
}

// ✅ CORRECT: Expired credentials
if claims.ExpiresAt.Before(time.Now()) {
    return errors.Unauthorized("token expired", nil)
}
```

#### FORBIDDEN

Use when authentication is valid but user lacks permission.

```go
// ✅ CORRECT: Permission denied
func (s *DocumentService) Delete(ctx context.Context, docID string) error {
    user := auth.UserFromContext(ctx)
    doc, err := s.repo.GetByID(ctx, docID)
    if err != nil {
        return err
    }
    
    if doc.OwnerID != user.ID && !user.IsAdmin {
        return errors.Forbidden("cannot delete document owned by another user", nil)
    }
    
    return s.repo.Delete(ctx, docID)
}
```

#### CONFLICT

Use for concurrent modification conflicts or duplicate resources.

```go
// ✅ CORRECT: Duplicate resource
func (r *UserRepo) Create(ctx context.Context, user *User) error {
    _, err := r.db.Exec(ctx, insertQuery, user.ID, user.Email)
    if isUniqueViolation(err) {
        return errors.Conflict("user with this email already exists", err)
    }
    if err != nil {
        return errors.Internal("failed to create user", err)
    }
    return nil
}

// ✅ CORRECT: Optimistic locking failure
func (r *DocumentRepo) Update(ctx context.Context, doc *Document) error {
    result, err := r.db.Exec(ctx, updateQuery, doc.Content, doc.Version+1, doc.ID, doc.Version)
    if result.RowsAffected() == 0 {
        return errors.Conflict("document was modified by another user", nil)
    }
    return err
}
```

### 8.3 Error Wrapping Conventions

```go
// ✅ CORRECT: Wrap with context
data, err := ioutil.ReadFile(path)
if err != nil {
    return errors.Wrap(err, "failed to read config file")
}

// ✅ CORRECT: Wrap external errors with appropriate code
resp, err := http.Get(url)
if err != nil {
    return errors.Internal("HTTP request failed", err)
}

// ✅ CORRECT: Preserve original error code when re-wrapping
user, err := s.repo.GetByID(ctx, id)
if err != nil {
    // Don't change NotFound to Internal
    return err  // Let the original error propagate
}

// ❌ WRONG: Double logging
if err != nil {
    logger.L().Error("failed", "error", err)  // Don't log here
    return errors.Internal("failed", err)      // Error will be logged in instrumented layer
}
```

### 8.4 Error Message Guidelines

```go
// ✅ CORRECT: Descriptive, actionable messages
errors.NotFound("user with ID 'abc123' not found", nil)
errors.InvalidArgument("email must be a valid email address", nil)
errors.Forbidden("admin role required to access this resource", nil)

// ❌ WRONG: Vague or technical messages
errors.NotFound("not found", nil)              // What wasn't found?
errors.Internal("error occurred", nil)          // What error?
errors.InvalidArgument("bad request", nil)      // What's wrong with it?
```

---

## 9. Event Standards

This section defines conventions for event-driven communication using `pkg/events`.

### 9.1 Event Type Naming

Use dot-notation with format: `{domain}.{action}` or `{domain}.{entity}.{action}`

```go
// ✅ CORRECT: Clear, domain-specific event types
"user.created"
"user.updated"
"user.deleted"
"order.placed"
"order.payment.completed"
"order.payment.failed"
"inventory.item.low_stock"
"cache.invalidated"

// ❌ WRONG: Vague or inconsistent naming
"created"           // Missing domain
"UserCreated"       // Wrong format (PascalCase)
"user-created"      // Wrong separator (use dots)
"user_was_created"  // Too verbose
```

### 9.2 Event Structure

Every event MUST follow the standard `events.Event` structure:

```go
event := events.Event{
    // REQUIRED: Unique identifier for this event instance
    ID: uuid.NewString(),
    
    // REQUIRED: Event type (dot-notation)
    Type: "user.created",
    
    // REQUIRED: Service/package that generated the event
    Source: "user-service",
    
    // REQUIRED: When the event occurred
    Timestamp: time.Now().UTC(),
    
    // REQUIRED: Event-specific data
    Payload: UserCreatedPayload{
        UserID:    user.ID,
        Email:     user.Email,
        CreatedAt: user.CreatedAt,
    },
}
```

### 9.3 Event Payload Guidelines

#### Define Typed Payloads

Always use typed structs for payloads, never raw maps or generic interfaces.

```go
// ✅ CORRECT: Typed payload
type UserCreatedPayload struct {
    UserID    string    `json:"user_id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

event := events.Event{
    Type:    "user.created",
    Payload: UserCreatedPayload{...},
}

// ❌ WRONG: Untyped payload
event := events.Event{
    Type: "user.created",
    Payload: map[string]interface{}{
        "user_id": user.ID,
        "email":   user.Email,
    },
}
```

#### Payload Content Rules

1. **Include only necessary data** - Don't include entire entities if only IDs are needed
2. **Use IDs, not full objects** - Let consumers fetch full data if needed
3. **Include timestamps** - When did the action occur?
4. **Avoid sensitive data** - No passwords, tokens, or PII unless necessary

```go
// ✅ CORRECT: Minimal, focused payload
type OrderPlacedPayload struct {
    OrderID     string    `json:"order_id"`
    CustomerID  string    `json:"customer_id"`
    TotalAmount float64   `json:"total_amount"`
    ItemCount   int       `json:"item_count"`
    PlacedAt    time.Time `json:"placed_at"`
}

// ❌ WRONG: Too much data
type OrderPlacedPayload struct {
    Order    Order      `json:"order"`     // Full entity - too heavy
    Customer Customer   `json:"customer"`  // Include customerID instead
    Items    []Item     `json:"items"`     // Include itemCount instead
}
```

### 9.4 Topic Naming

Topics group related events. Use domain-based naming:

```go
// ✅ CORRECT: Domain-based topics
"users"           // All user-related events
"orders"          // All order-related events
"payments"        // All payment-related events
"notifications"   // All notification events

// Publish to appropriate topic
bus.Publish(ctx, "users", event)      // for user.created, user.updated
bus.Publish(ctx, "orders", event)     // for order.placed, order.shipped
```

### 9.5 Event Publishing Patterns

#### Publish After Success

Only publish events after the primary operation succeeds:

```go
// ✅ CORRECT: Publish after successful operation
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // 1. Perform the operation
    user, err := s.repo.Create(ctx, req.ToUser())
    if err != nil {
        return nil, err
    }
    
    // 2. Publish event AFTER success
    event := events.Event{
        ID:        uuid.NewString(),
        Type:      "user.created",
        Source:    "user-service",
        Timestamp: time.Now().UTC(),
        Payload:   UserCreatedPayload{UserID: user.ID, Email: user.Email},
    }
    
    if err := s.bus.Publish(ctx, "users", event); err != nil {
        // Log but don't fail the request - event publishing is best-effort
        logger.L().WarnContext(ctx, "failed to publish user.created event",
            "user_id", user.ID,
            "error", err,
        )
    }
    
    return user, nil
}

// ❌ WRONG: Publish before operation completes
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    s.bus.Publish(ctx, "users", event)  // Don't publish yet!
    return s.repo.Create(ctx, req.ToUser())
}
```

#### Event Sourcing Pattern

For critical events where durability matters:

```go
// ✅ CORRECT: Transactional outbox pattern
func (s *OrderService) PlaceOrder(ctx context.Context, order *Order) error {
    return s.db.Transaction(ctx, func(tx *sql.Tx) error {
        // 1. Save the order
        if err := s.repo.CreateWithTx(ctx, tx, order); err != nil {
            return err
        }
        
        // 2. Save the event to outbox (same transaction)
        event := events.Event{
            Type:    "order.placed",
            Payload: OrderPlacedPayload{OrderID: order.ID},
        }
        if err := s.outbox.SaveWithTx(ctx, tx, event); err != nil {
            return err
        }
        
        return nil
    })
    // Background worker publishes from outbox
}
```

### 9.6 Event Handling Patterns

#### Idempotent Handlers

Event handlers MUST be idempotent (safe to process multiple times):

```go
// ✅ CORRECT: Idempotent handler
func (h *WelcomeEmailHandler) Handle(ctx context.Context, event events.Event) error {
    var payload UserCreatedPayload
    if err := mapstructure.Decode(event.Payload, &payload); err != nil {
        return errors.InvalidArgument("invalid payload", err)
    }
    
    // Check if already processed (idempotency)
    processed, err := h.cache.Get(ctx, "email:welcome:"+payload.UserID, nil)
    if err == nil && processed {
        return nil  // Already sent, skip
    }
    
    // Send email
    if err := h.emailer.SendWelcome(ctx, payload.Email); err != nil {
        return err
    }
    
    // Mark as processed
    _ = h.cache.Set(ctx, "email:welcome:"+payload.UserID, true, 24*time.Hour)
    return nil
}
```

#### Error Handling in Handlers

```go
func (h *Handler) Handle(ctx context.Context, event events.Event) error {
    // Validate payload
    var payload OrderPlacedPayload
    if err := mapstructure.Decode(event.Payload, &payload); err != nil {
        // Log and return nil - don't retry malformed events
        logger.L().ErrorContext(ctx, "invalid event payload",
            "event_id", event.ID,
            "event_type", event.Type,
            "error", err,
        )
        return nil  // Acknowledge the malformed event
    }
    
    // Process event
    if err := h.process(ctx, payload); err != nil {
        // Return error to trigger retry
        return errors.Wrap(err, "failed to process order.placed")
    }
    
    return nil
}
```

### 9.7 Standard Event Types

Common event types used across the library:

| Event Type | Topic | Description |
|------------|-------|-------------|
| `user.created` | `users` | New user registered |
| `user.updated` | `users` | User profile updated |
| `user.deleted` | `users` | User account deleted |
| `cache.invalidated` | `cache` | Cache entry invalidated |
| `config.reloaded` | `system` | Configuration reloaded |
| `health.degraded` | `system` | System health degraded |
| `health.recovered` | `system` | System health recovered |

---

## 10. Plugin & Extension Architecture

### 10.1 Middleware Pattern

For packages that need extensibility, use middleware:

```go
// Middleware signature
type Middleware func(next Handler) Handler

// Apply middleware
handler = LoggingMiddleware(handler)
handler = AuthMiddleware(handler)
handler = RateLimitMiddleware(handler)
```

### 10.2 Hook/Callback Pattern

For specific extension points:

```go
type Config struct {
    // Hooks
    OnSuccess func(ctx context.Context, key string)
    OnError   func(ctx context.Context, key string, err error)
    OnEvict   func(ctx context.Context, key string)
}
```

### 10.3 Plugin Interface Pattern

For complex plugins:

```go
// Plugin defines a cache plugin interface
type Plugin interface {
    Name() string
    Init(cache Cache) error
    BeforeGet(ctx context.Context, key string) error
    AfterGet(ctx context.Context, key string, err error)
    BeforeSet(ctx context.Context, key string, value interface{}) error
    AfterSet(ctx context.Context, key string, err error)
}
```

### 10.4 Event Bus Integration

Prefer loose coupling via events:

```go
// ❌ WRONG: Tight coupling
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }
    s.emailService.SendWelcome(user.Email)  // Tight coupling
    s.analyticsService.TrackSignup(user.ID) // Tight coupling
    return nil
}

// ✅ CORRECT: Event-driven
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    if err := s.repo.Create(ctx, user); err != nil {
        return err
    }
    return s.bus.Publish(ctx, "users", events.Event{
        Type:    "user.created",
        Payload: user,
    })
}
// EmailService and AnalyticsService subscribe to "user.created"
```

---

## 11. Testing Standards

### 11.1 Required Test Coverage

| Test Type | Requirement | Location |
|-----------|-------------|----------|
| Unit tests | **Required** for all adapters | `adapters/{driver}/{driver}_test.go` |
| Interface tests | **Recommended** | `tests/{capability}_test.go` |
| Memory adapter | **Required** | `adapters/memory/` |
| Integration tests | Optional | `tests/integration_test.go` |

### 11.2 Memory Adapter Requirement

Every package with adapters MUST provide an in-memory implementation for testing:

```go
// pkg/cache/adapters/memory/memory.go
func New() *MemoryCache {
    return &MemoryCache{
        items: make(map[string]item),
        mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-cache"}),
    }
}
```

This enables:
- `go test` runs without Docker/external services
- Fast CI/CD pipelines
- Easy mocking in consumer tests

### 11.3 Interface Testing Pattern

Write tests against the interface, run against all adapters:

```go
func runCacheTests(t *testing.T, cache cache.Cache) {
    t.Run("Set and Get", func(t *testing.T) {
        ctx := context.Background()
        err := cache.Set(ctx, "key", "value", time.Hour)
        require.NoError(t, err)

        var result string
        err = cache.Get(ctx, "key", &result)
        require.NoError(t, err)
        assert.Equal(t, "value", result)
    })

    t.Run("Get missing key", func(t *testing.T) {
        ctx := context.Background()
        var result string
        err := cache.Get(ctx, "nonexistent", &result)
        require.Error(t, err)
        
        var appErr *errors.AppError
        require.True(t, errors.As(err, &appErr))
        assert.Equal(t, errors.CodeNotFound, appErr.Code)
    })
}

func TestMemoryCache(t *testing.T) {
    cache := memory.New()
    defer cache.Close()
    runCacheTests(t, cache)
}

func TestRedisCache(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    cache := redis.New(cfg)
    defer cache.Close()
    runCacheTests(t, cache)
}
```

### 11.4 Test Suite Pattern

Use `pkg/test` for test suites:

```go
type CacheTestSuite struct {
    test.Suite
    cache cache.Cache
}

func (s *CacheTestSuite) SetupTest() {
    s.Suite.SetupTest()
    s.cache = memory.New()
}

func (s *CacheTestSuite) TearDownTest() {
    s.cache.Close()
}

func (s *CacheTestSuite) TestSetGet() {
    err := s.cache.Set(s.Ctx, "key", "value", time.Hour)
    s.NoError(err)
    
    var result string
    err = s.cache.Get(s.Ctx, "key", &result)
    s.NoError(err)
    s.Equal("value", result)
}

func TestCacheSuite(t *testing.T) {
    test.Run(t, new(CacheTestSuite))
}
```

---

## 12. Code Quality Gates

All code MUST pass quality gates before being pushed to the repository.

### 12.1 Required Checks

Every push must pass these checks (enforced by CI and pre-push hook):

| Check | Tool | Requirement |
|-------|------|-------------|
| **Formatting** | `gofmt` | All files formatted with `gofmt` |
| **Static Analysis** | `go vet` | No issues reported |
| **Build** | `go build` | All packages compile without errors |
| **Tests** | `go test` | All tests pass with race detector enabled |

### 12.2 Code Formatting

Go files MUST be formatted with `gofmt`. No exceptions.

```bash
# Format all files
make fmt

# Check formatting without modifying
make fmt-check
```

**Important:** Never commit unformatted code. The pre-push hook and CI will reject it.

### 12.3 Build Verification

All packages MUST compile successfully:

```bash
# Verify build
make build

# Or directly
go build ./...
```

**Rule:** If it doesn't compile, it doesn't get pushed.

### 12.4 Test Requirements

All tests MUST pass:

```bash
# Run all tests
make test

# Run with coverage
make test-cover
```

**Rules:**
- Tests run with `-race` flag to detect race conditions
- Tests must not require external services (use in-memory adapters)
- Flaky tests are not acceptable

### 12.5 Local Quality Gate (Pre-Push Hook)

Install the pre-push hook to automatically check code before pushing:

```bash
# Copy hook to git hooks directory
cp .github/hooks/pre-push .git/hooks/pre-push
chmod +x .git/hooks/pre-push
```

The hook runs:
1. `gofmt` formatting check
2. `go vet` static analysis
3. `go build` compilation check
4. `go test` test suite

If any check fails, the push is blocked.

**To bypass (not recommended):**
```bash
git push --no-verify
```

### 12.6 Running Quality Checks Manually

Before pushing, run the full quality gate:

```bash
# Full check (format, vet, build, tests)
make check

# Quick check (skip tests for speed)
make check-quick
```

### 12.7 CI Pipeline

The GitHub Actions CI pipeline runs on every push and pull request:

```yaml
# .github/workflows/ci.yml triggers:
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]
```

**CI Checks:**
1. `gofmt` formatting verification
2. `go vet` static analysis
3. `staticcheck` advanced linting
4. `go build` compilation
5. `go test -race` tests with race detection
6. `golangci-lint` comprehensive linting

PRs with failing checks cannot be merged.

### 12.8 Installing Development Tools

Install required development tools:

```bash
make install-tools
```

This installs:
- `staticcheck` - Advanced static analysis

---

## 13. Compliance Checklist

Use this checklist when creating or auditing packages:

### Architecture
- [ ] Primary interface defined in `{package}.go`
- [ ] `Config` struct with `env` tags
- [ ] Consumers use interface, not concrete types
- [ ] All I/O methods accept `context.Context` first

### File Structure
- [ ] `{capability}.go` - interfaces and config
- [ ] `instrumented.go` - observability wrapper
- [ ] `errors.go` - error constructors (if applicable)
- [ ] `adapters/memory/` - in-memory adapter exists
- [ ] Adapters in isolated packages under `adapters/`

### Naming Conventions
- [ ] Files are `lowercase.go` or `lowercase_suffix.go`
- [ ] Types follow naming conventions
- [ ] Imports organized in groups

### Code Quality
- [ ] No logging/tracing in adapters (only in `instrumented.go`)
- [ ] Using `pkg/errors` for all errors
- [ ] Using `pkg/logger` for all logging
- [ ] Using `pkg/concurrency` mutexes (not `sync.RWMutex`)
- [ ] Constructor functions named `New` or `NewXxx`

### Documentation
- [ ] Package-level doc comment
- [ ] All exported functions documented
- [ ] Interface methods documented
- [ ] Config fields documented

### Integration
- [ ] `pkg/errors` - error handling
- [ ] `pkg/logger` - logging
- [ ] `pkg/telemetry` - tracing in `instrumented.go`
- [ ] `pkg/validator` - config validation
- [ ] `pkg/concurrency` - thread safety (if applicable)
- [ ] `pkg/resilience` - circuit breakers (if external calls)
- [ ] `pkg/events` - event publishing (if domain events)
- [ ] `pkg/test` - test suite (if tests exist)

### Testing
- [ ] Memory adapter for zero-Docker testing
- [ ] Unit tests for all adapters
- [ ] Tests use interface, not concrete types
- [ ] Tests use `pkg/test` suite where appropriate

### Quality Gates
- [ ] All files formatted with `gofmt`
- [ ] `go vet` passes with no issues
- [ ] All packages compile successfully
- [ ] All tests pass with race detector
- [ ] Pre-push hook installed (`make install-hooks`)

---

## Appendix: Quick Reference Templates

### Template: New Package Root File

```go
// Package {name} provides ... .
//
// This package supports:
//   - ...
package {name}

import (
    "context"
)

// {Name} defines the interface for ...
type {Name} interface {
    // Method does ...
    Method(ctx context.Context, ...) error
    
    // Close releases resources.
    Close() error
}

// Config holds configuration for {Name}.
type Config struct {
    Driver string `env:"{NAME}_DRIVER" env-default:"memory"`
    // ...
}
```

### Template: New Adapter

```go
package {driver}

import (
    "context"
    
    "github.com/chris-alexander-pop/go-hyperforge/pkg/{capability}"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
    "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type {Driver}{Capability} struct {
    // fields
    mu *concurrency.SmartRWMutex
}

func New() *{Driver}{Capability} {
    return &{Driver}{Capability}{
        mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
            Name: "{driver}-{capability}",
        }),
    }
}

func (x *{Driver}{Capability}) Method(ctx context.Context, ...) error {
    // implementation using pkg/errors
}

func (x *{Driver}{Capability}) Close() error {
    return nil
}
```

### Template: Instrumented Wrapper

```go
package {capability}

import (
    "context"
    
    "github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

type Instrumented{Name} struct {
    next   {Name}
    tracer trace.Tracer
}

func NewInstrumented{Name}(next {Name}) *Instrumented{Name} {
    return &Instrumented{Name}{
        next:   next,
        tracer: otel.Tracer("pkg/{capability}"),
    }
}

func (i *Instrumented{Name}) Method(ctx context.Context, ...) error {
    ctx, span := i.tracer.Start(ctx, "{capability}.Method", trace.WithAttributes(
        attribute.String("key", value),
    ))
    defer span.End()

    logger.L().InfoContext(ctx, "method called", "key", value)

    err := i.next.Method(ctx, ...)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        logger.L().ErrorContext(ctx, "method failed", "error", err)
    }
    return err
}

func (i *Instrumented{Name}) Close() error {
    return i.next.Close()
}
```

---

**Document Version:** 1.0.0  
**Last Updated:** January 2026  
**Maintainer:** System Design Library Team
