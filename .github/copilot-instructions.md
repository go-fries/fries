# Go Fries Components

Go Fries Components is a multi-module Go component library providing reusable components for building applications. This repository contains 79+ individual Go modules organized by functionality including caching, HTTP servers, middleware, databases, logging, and more.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

Bootstrap, build, and test the repository:

- **Install development tools**: `make tools` -- takes 30-60 seconds to complete. NEVER CANCEL. Set timeout to 120+ seconds.
- **Build all modules**: `make build` -- takes 1-2 minutes to complete. NEVER CANCEL. Set timeout to 180+ seconds.
- **Lint all modules**: `make lint` -- takes 6-8 minutes to complete. NEVER CANCEL. Set timeout to 600+ seconds.
- **Test individual modules**: `cd <module-dir> && go test .` -- takes 1-10 seconds per module.
- **Test with external services**: Some tests require Redis (port 6379) and MySQL (port 3306). Use CI environment for complete testing.

## Key Commands and Timeouts

**CRITICAL**: All build and lint commands take significant time. Do NOT cancel them prematurely.

- `make tools` -- 30-60s (installs golangci-lint, buf, and other development tools)
- `make build` -- 80-120s (builds all 79+ Go modules)  
- `make lint` -- 6-8 minutes (runs golangci-lint on all modules with go mod tidy)
- `make test` -- varies (some modules need Redis/MySQL services)
- `make test-short` -- same as test but attempts to skip integration tests
- `make golangci-lint` -- runs linter only (part of make lint)
- `make clean` -- removes .tools directory

## Testing Strategy

**Without External Services:**
- Test individual modules: `cd <module> && go test .`
- Example modules that work independently: `support`, `strings`, `slices`, `constraints`, `errors`

**With External Services (CI environment):**
- Full test suite requires Redis on :6379 and MySQL on :3306
- Tests will fail locally without these services running
- This is expected behavior - document service requirements

## Repository Structure

This is a multi-module repository with the following organization:

### Core Modules
- `cache/` - Caching abstractions and Redis implementation  
- `errors/` - Error handling utilities
- `config/` - Configuration management
- `support/` - Helper functions and utilities

### Web Frameworks & HTTP
- `chi/` - Chi router components
- `gin/` - Gin framework components  
- `http/server/` - HTTP server utilities
- `kratos/` - Kratos framework middleware and utilities

### Middleware
- `kratos/middleware/cors/` - CORS middleware
- `kratos/middleware/tracing/` - Tracing middleware
- `hyperf/jet/middleware/` - Various Jet middleware (logging, retry, timeout, etc.)

### Databases & Storage
- `gorm/` - GORM utilities and scopes
- `mysql/canal/` - MySQL binlog parsing
- `redis/` - Redis utilities  
- `ent/` - Ent ORM utilities
- `filesystem/` - File system abstractions (local, S3, OSS)

### Examples
- `examples/cache/` - Cache usage example
- `examples/otel/otlp/` - OpenTelemetry example
- `examples/cloudevents/` - CloudEvents examples

## Validation

**Always run validation steps after making changes:**

1. **Build validation**: `make build` - ensure all modules compile
2. **Lint validation**: `make lint` - ensure code quality standards  
3. **Module-specific testing**: `cd <changed-module> && go test .`
4. **Integration testing**: Only in CI environment with services

**Manual Testing Scenarios:**
- Build and run example applications in `examples/` directory: `cd examples/cache && go build .` (builds successfully, requires Redis to run)
- Test that imports work with module functionality: Create a simple test program importing key modules like `strings/v3` and `support/v3`
- Validate module interfaces haven't broken by spot-checking key modules like `support`, `strings`, `errors`
- Run example test: `cd /tmp && mkdir test_app && cd test_app && go mod init test && echo 'package main; import "github.com/go-fries/fries/strings/v3"; func main() { println(strings.MD5("test")) }' > main.go` then `go mod tidy && go run main.go`

## Module Development Patterns

This repository follows Go module best practices:

- Each directory with `go.mod` is an independent module
- Modules are versioned as `v3` (breaking change from v2)  
- Cross-module dependencies use full import paths: `github.com/go-fries/fries/<module>/v3`
- All modules maintain backward compatibility within v3

## Common Issues

**Build Issues:**
- Missing tools: Run `make tools` first
- Module dependency issues: Check `go.mod` files in affected modules
- Cross-module dependency issues: Use `make crosslink` to update internal dependencies

**Test Issues:**  
- Redis connection failures: Tests require Redis on :6379 (document as known limitation)
- MySQL connection failures: Tests require MySQL on :3306 (document as known limitation)
- Use `cd <module> && go test .` for unit tests without external dependencies

**Lint Issues:**
- Takes 6+ minutes - this is normal for 79+ modules
- Failures usually indicate code quality issues that must be fixed
- Use `make golangci-lint-fix` to auto-fix some issues

## CI/CD Integration

The repository uses GitHub Actions:

- **Lint workflow** (`.github/workflows/lint.yml`): Runs `make lint` and `make build`
- **Test workflow** (`.github/workflows/test.yml`): Runs full test suite with Redis/MySQL services
- **Analysis workflow** (`.github/workflows/analysis.yml`): Additional code analysis

Always run `make lint` and `make build` before committing to ensure CI will pass.

## Performance Notes

**Expected Command Durations:**
- Tool installation: 7-10 seconds (clean) to 30-60 seconds (first time)
- Full build: 8-12 seconds (clean) to 1-2 minutes (first time with downloads)
- Full lint: 6-8 minutes (always takes this long due to 79+ modules)
- Individual module test: 1-10 seconds
- Individual module build: 5-30 seconds

Do NOT cancel long-running commands. The multi-module nature of this repository requires significant processing time, especially for linting which processes all modules individually.