# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Fries Components is a multi-module Go component library providing 79+ reusable modules for building applications. The repository is organized as a monorepo with independent Go modules for caching, HTTP servers, middleware, databases, logging, and more. Each module is independently versioned as v3.

**Main branch**: `3.x`

**Important**: v3 may have backward compatibility breaking changes from v2. All modules use the v3 import path suffix.

## Development Commands

**CRITICAL**: Build and lint commands take significant time due to 79+ modules. Set appropriate timeouts and NEVER cancel prematurely.

### Essential Commands

```bash
# Install development tools (30-60s, use 120s timeout)
make tools

# Build all modules (1-2 minutes, use 180s timeout)
make build

# Lint all modules (6-8 minutes, use 600s timeout)
make lint

# Auto-fix lint issues
make golangci-lint-fix

# Run go mod tidy across all modules
make go-mod-tidy

# Update cross-module dependencies
make crosslink

# Clean tools directory
make clean
```

### Testing

```bash
# Test individual module (recommended for local development)
cd <module-dir> && go test .

# Test all modules (requires Redis on :6379 and MySQL on :3306)
make test

# Test with race detection
make test-race

# Test with short flag (attempts to skip integration tests)
make test-short

# Test coverage
make test-coverage
```

**Testing Strategy**: Many modules require external services (Redis, MySQL). For local testing without services, test individual modules that don't have external dependencies (e.g., `support`, `strings`, `slices`, `constraints`, `errors`). Full integration tests run in CI with services.

### Protocol Buffers

```bash
# Lint proto files
make buf-lint

# Build proto files
make buf-build

# Generate proto code
make buf-generate
```

### Release Management

```bash
# Verify module configuration
make verify-mods

# Check API compatibility
make gorelease

# Create pre-release (requires MODSET env var)
MODSET=stable make prerelease

# Add tags (requires MODSET env var)
MODSET=stable make add-tags

# Push specific tags (requires TAG env var)
TAG=v3.11.0 make push-tags

# Upgrade Go version in all modules
GO_VERSION=1.24.0 make upgrade-go-version
```

## Repository Architecture

### Multi-Module Structure

This is a **multi-module monorepo**. Each directory with a `go.mod` file is an independent Go module with its own versioning and dependencies. Key patterns:

1. **Module Independence**: Each module is independently buildable and testable
2. **Version Suffix**: All modules use `/v3` suffix in their import paths
3. **Cross-Module Dependencies**: Use full import paths like `github.com/go-fries/fries/<module>/v3`
4. **Replace Directives**: Local development uses `replace` directives in go.mod for cross-module references
5. **Module Sets**: The `versions.yaml` file defines module groups (currently "stable" at v3.11.0)

### Module Categories

**Core Utilities**:
- `support/` - Helper functions, value proxies, byte/map/context utilities
- `errors/` - Error handling with Kratos integration
- `constraints/` - Generic type constraints
- `strings/` - String utilities
- `slices/` - Slice utilities
- `coroutines/` - Goroutine utilities

**Web Frameworks & HTTP**:
- `chi/` - Chi router integration
- `gin/` - Gin framework components
- `http/server/` - HTTP server utilities
- `kratos/middleware/` - Kratos framework middleware (cors, tracing, slowlog, protovalidate)
- `kratos/log/` - Kratos logging (stack, syslog, otel)
- `hyperf/jet/` - Jet middleware (logging, recovery, retry, timeout, tracing)

**Caching & Storage**:
- `cache/` - Caching abstractions with Repository/Store/Snapshot patterns
- `cache/redis/` - Redis-backed cache implementation
- `locker/` - Distributed locking interface
- `locker/redis/` - Redis-backed distributed locks
- `redis/` - Redis client utilities

**Databases & ORM**:
- `gorm/` - GORM utilities and plugins
- `gorm/scope/` - GORM query scopes
- `ent/` - Ent ORM utilities
- `ent/multidriver/` - Ent multi-driver support
- `mysql/canal/` - MySQL binlog change data capture
- `mysql/canal/positioner/redis/` - Redis-based binlog position tracking
- `mysql/canal/server/` - Canal server implementation

**File Systems**:
- `filesystem/` - Abstract filesystem interface
- `filesystem/local/` - Local filesystem driver
- `filesystem/s3/` - AWS S3 driver
- `filesystem/oss/` - Alibaba Cloud OSS driver

**Encoding & Serialization**:
- `codec/` - Codec abstraction
- `codec/json/`, `codec/sonic/` - JSON encoders
- `codec/msgpack/` - MessagePack
- `codec/proto/` - Protocol Buffers
- `codec/xml/`, `codec/yaml/` - XML and YAML

**Events & Messaging**:
- `event/` - Event system with dispatcher pattern
- `event/middleware/recovery/` - Event middleware for panic recovery
- `eventbus/` - Event bus implementation
- `cloudevents/protocol/amqp091/` - CloudEvents AMQP protocol
- `cloudevents/eventdispatcher/` - CloudEvents dispatcher

**AI & Embeddings (Eino)**:
- `eino/components/embedding/cached/` - Cached embedding component
- `eino/components/embedding/cached/cacher/redis/` - Redis caching for embeddings
- `eino/components/embedding/cached/cacher/gorm/` - GORM caching for embeddings

**Other Components**:
- `config/` - Configuration management
- `crontab/` - Cron job scheduling
- `encrypter/` - Encryption utilities
- `env/` - Environment variable handling
- `foundation/` - Foundation components
- `hashing/`, `hashing/md5/` - Hashing utilities
- `jsonrpc/` - JSON-RPC implementation
- `otel/otlp/` - OpenTelemetry OTLP integration
- `recovery/` - Panic recovery utilities
- `signal/` - Signal handling
- `timezone/` - Timezone utilities
- `udp/` - UDP utilities
- `x/pagination/`, `x/prints/`, `x/container/` - Extra utilities

### Important Files

- `versions.yaml` - Defines module sets and versions for releases (managed by multimod tool)
- `Makefile` - Build automation with pattern-based rules for all modules
- `internal/tools/` - Development tools (golangci-lint, buf, multimod, crosslink, etc.)
- `.github/copilot-instructions.md` - Detailed development guidelines (authoritative source)

## Development Workflow

### Making Changes

1. **Read before modifying**: Always read existing code before making changes
2. **Test locally**: Test the specific module you're changing with `cd <module> && go test .`
3. **Build validation**: Run `make build` to ensure all modules still compile
4. **Lint validation**: Run `make lint` before committing (takes 6-8 minutes)
5. **Update dependencies**: If changing cross-module dependencies, run `make crosslink`

### Module Development Patterns

When creating or modifying modules:

1. **go.mod structure**: Each module's go.mod uses v3 suffix and may have `replace` directives for local cross-module deps
2. **Interface-driven**: Many modules define interfaces (e.g., `cache.Store`, `filesystem.Driver`) with multiple implementations
3. **Repository pattern**: The cache module uses Repository/Store/Snapshot pattern for state management
4. **Middleware pattern**: Middleware modules (Kratos, Hyperf Jet, Event) follow standard middleware chaining patterns
5. **Testing**: Use `testify` for assertions, mock external dependencies, mark integration tests appropriately

### Common Patterns

**Cache Module Pattern** (`cache/`):
- `Repository` manages cache operations
- `Store` provides the backend implementation
- `Snapshot` captures cache state for transactions
- Null object pattern for disabled caching

**Filesystem Module Pattern** (`filesystem/`):
- Abstract `Driver` interface
- Concrete implementations (local, s3, oss)
- Path normalization and error handling

**Middleware Pattern** (various):
- Standard `func(handler) handler` pattern
- Options-based configuration
- Context propagation

## CI/CD

GitHub Actions workflows:
- **lint.yml**: Runs `make lint` and `make build` on push/PR to `3.x`
- **test.yml**: Full test suite with Redis/MySQL services
- **analysis.yml**: Additional code analysis

Always ensure `make lint` and `make build` pass locally before pushing to ensure CI success.

## Common Issues

**Build fails with missing tools**: Run `make tools` first (takes 30-60s)

**Cross-module dependency errors**: Run `make crosslink` to update intra-repository dependencies

**Tests fail with connection errors**: Tests requiring Redis (:6379) or MySQL (:3306) will fail without those services. This is expected for local development. Test individual modules without external deps instead.

**Lint takes too long**: This is normal - 6-8 minutes for 79+ modules. Do NOT cancel.

**Module version mismatches**: Check `versions.yaml` for current stable version, ensure go.mod files use consistent v3 suffix

## Module Import Paths

All modules use the pattern:
```go
import "github.com/go-fries/fries/<module>/v3"
```

Examples:
```go
import "github.com/go-fries/fries/support/v3"
import "github.com/go-fries/fries/cache/redis/v3"
import "github.com/go-fries/fries/kratos/middleware/cors/v3"
```