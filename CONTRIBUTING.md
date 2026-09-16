# Contributing

This document defines contribution conventions for this repository. It applies to new components, component refactors, public API design, and tests across all modules in this repository.

The goal is to keep components consistent, maintainable, and predictable for both users and maintainers.

## Module Layout

This is a multi-module Go repository. The root module is `github.com/go-fries/fries/v4`, and many component directories have their own `go.mod` files.

Keep module boundaries explicit:

- update the nearest `go.mod` for the module being changed
- keep module paths in `go.mod` suffixed with `/v4`
- avoid unnecessary dependencies across components
- keep framework integrations under the framework or component they adapt

Framework and library integrations should follow the existing package layout for the integration target. Prefer placing integration code near the component or framework it adapts instead of introducing a new top-level area without a clear repository-wide reason.

## Design Principles

- Keep component boundaries clear. A component should own its configuration, runtime behavior, and tests within its package or module boundary.
- Prefer explicit behavior over implicit side effects. Errors should be returned where they occur and should not be hidden until later runtime behavior.
- Keep public APIs small and stable. Add exported identifiers only when they describe a real user-facing capability.
- Prefer established repository patterns over new local styles unless the component has a clear reason to differ.
- Avoid over-design. Add abstractions only when they reduce real complexity, remove meaningful duplication, or make public behavior easier to understand.

## Public Modules

New releasable component modules should update repository release and reporting metadata in the same change.

For public modules:

- add the module path to `versions.yaml`
- add the matching Codecov path rewrite to `codecov.yml`
- include package documentation in `doc.go`
- include a `README.md` when the module is intended for direct use
- add Go doc comments for exported identifiers
- provide a module-local `Version()` helper when the component exposes or reports version metadata

Public modules should be usable from Go documentation alone. README files should focus on installation, common usage, and behavior that is not obvious from type signatures.

## Examples

Keep examples close to the APIs they document:

- put common usage and required setup in the component README
- use package-local `example_test.go` files for compile-checked Go documentation examples
- keep external-service scenarios in integration tests beside the component and honor `testing.Short()`

Avoid standalone example modules unless the example has an independent release or dependency lifecycle that cannot be represented clearly inside the component.

## Component Configuration

Configuration state and option parsing should live in a package-local `config.go` when a component has configurable behavior.

Use `config.go` for:

- the internal `config` type
- default values
- the public `Option` contract
- the private `optionFunc` implementation
- public `WithXxx(...)` option helpers
- `newConfig(...)`
- small config-owned factory helpers such as `newLogger(...)`, `newResource(...)`, or `newTracerProvider(...)`

Runtime types should focus on lifecycle and behavior. The primary runtime type in a component should generally call into its `config` for configured dependencies instead of carrying many option fields or scattered defaulting logic directly.

Config-owned factory helpers are appropriate when the created value is determined by configuration. Keep business logic, request handling, and runtime orchestration outside `config.go`.

## Options

Use an interface-based option contract with an unexported apply method by default:

```go
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}
```

Public option helpers should use `WithXxx(...) Option` naming and return `optionFunc`:

```go
func WithServiceName(name string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = name
	})
}
```

Place option definitions in this order:

1. `Option`
2. `optionFunc`
3. `optionFunc.apply`
4. public `WithXxx(...)` helpers
5. `newConfig(...)`
6. config helper methods

Prefer this form over exporting or aliasing a raw `func(*config)` option type. It keeps option construction controlled inside the package while leaving room for future specialized option implementations.

When option application has a real failure path, use an error-returning option contract:

```go
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (f optionFunc) apply(c *config) error {
	return f(c)
}
```

Use the non-error form for simple assignment and defaulting. Use the error-returning form when option-time validation or setup can fail. Return validation errors explicitly instead of deferring them to later runtime behavior.

## Testing

Tests should live beside the package under test and follow Go's standard `testing` package conventions.

Use `github.com/stretchr/testify` when it improves readability. This is encouraged for grouped assertions, error checks, and setup requirements, but it is not required for trivial standard-library checks.

Use `require` for conditions that must stop the current test before continuing, including:

- constructor errors
- setup failures
- nil checks before dereferencing
- preconditions for later assertions

Use `assert` for independent value checks where the test can continue and report multiple failures in one run.

Prefer explicit assertions that describe intent:

- `require.NoError`
- `require.ErrorIs`
- `assert.ErrorIs`
- `assert.Equal`
- `assert.Same`
- `assert.Contains`

Use table tests when they make behavior easier to scan. Keep each case focused on one observable behavior, and avoid broad table tests that hide setup complexity or make failures hard to diagnose.

## Validation

Run commands from the repository root unless working in a specific module.

Use repository-level commands for broad changes:

- `make build`
- `make test`
- `make test-short`
- `make test-race`
- `make test-coverage`
- `make lint`

Use module-specific Make targets for focused work, such as `make test/cache`, `make lint/cache`, and `make lint-fix/cache`.

Run `make lint` before submitting broad or public API changes. It runs `go mod tidy` across modules and then `golangci-lint`.

For protobuf changes, use the Buf targets defined by the repository:

- `make buf-lint`
- `make buf-build`
- `make buf-validate`
- `make buf-generate`

### Test command options

Test targets run each selected module with a default per-package timeout of 60 seconds. Override it with `TIMEOUT=120` (in seconds), and select a Go executable with `GO=/path/to/go`.

| Target | Default test arguments |
| --- | --- |
| `make test`, `make test/MODULE` | None |
| `make test-default`, `make test-race` | `-race` |
| `make test-short` | `-short` |
| `make test-verbose` | `-v -race` |
| `make test-concurrent-safe` | `-run=ConcurrentSafe -count=100 -race`, with a 120-second timeout |
| `make test-coverage` | `-race`, plus coverage collection flags |
| `make test-unit` | `-race`, with `-short` appended after `ARGS` |
| `make test-integration` | `-race`, with `-short=false -count=1 -v` appended after `ARGS` |

Command-line `ARGS` replaces the target's default test arguments. Include `-race` explicitly when overriding `ARGS` if you want race detection; `ARGS=` clears the default arguments. Coverage collection flags and the configured timeout are still passed.

```sh
make test/cache ARGS='-short -race -count=1' TIMEOUT=120
make test-coverage ARGS='-race -count=1'
```

Keep shell quoting inside `ARGS` for test-name regular expressions containing shell metacharacters:

```sh
make test/cache ARGS="-short -race -run='TestSnapshotWithExpireAndErr|TestUtils_Remember'"
```

Make also interprets dollar signs: an end anchor must reach Make as `$$`, protected from expansion by the invoking shell. Omitting the anchor is simpler when the test-name prefix is already unique. Test logs show the module and the argument values passed to Go.

### Service-independent and integration suites

Run `make test-unit` to test all component and example modules without separately provisioned services. Tests may still use temporary files, local sockets, or test HTTP servers. The tools module is excluded. This target appends `-short` after `ARGS` so argument customization does not accidentally enable service tests.

Run `make test-integration` with Redis and MySQL available to execute the full suites of the service-backed modules listed in [`internal/testing/modules.mk`](internal/testing/modules.mk). It appends `-short=false -count=1 -v` so integration runs execute rather than use cached results, and report individual test outcomes. Invoke unit and integration targets separately; both reuse the module test targets.

```sh
make test-unit
REDIS_ADDR=localhost:6379 \
MYSQL_DSN='gorm:gorm@tcp(localhost:3306)/gorm?charset=utf8&parseTime=True&loc=Local' \
  make test-integration
```

All service-backed tests skip only in short mode. Otherwise a connection, setup, or cleanup error fails the test. `REDIS_ADDR` and `MYSQL_DSN` default to the values shown above. Redis tests use unique key namespaces; MySQL tests use randomly prefixed tables within an existing database.

Use `make test/MODULE` with explicit flags for filtered debugging runs. CI runs service-independent modules in short mode and service-backed modules in full mode, with race detection, uncached execution, and verbose output in every group. See the [module test inventory](internal/testing/README.md) for the audit and integration entry points. When introducing service-backed tests, update the manifest and inventory in the same change.

### Coverage artifacts

`make test-coverage` includes repository modules except `internal/tools` and modules within `example` or `examples` directories. Override `ALL_COVERAGE_MOD_DIRS` to select specific modules:

```sh
make test-coverage ALL_COVERAGE_MOD_DIRS='./cache ./retry'
```

`COVERAGE_GROUP` selects `all` (the default), `unit`, `redis`, or `mysql`. The unit group excludes service-backed modules; Redis and MySQL groups each run the full suites of their listed modules, including in-process tests. Groups filter `ALL_COVERAGE_MOD_DIRS`, so an explicit module selection can narrow them further. An unknown group or empty selection fails. Group selection does not change test flags:

```sh
make verify-test-groups
make test-coverage COVERAGE_GROUP=unit ARGS='-race -short -count=1'
REDIS_ADDR=localhost:6379 make test-coverage COVERAGE_GROUP=redis ARGS='-race -short=false -count=1 -v'
```

CI uses these groups for both supported Go versions, with only the MySQL group repeated for MySQL latest and 5.7. At most four test jobs run concurrently. Each job uploads a uniquely named report; a downstream job requires all four reports for its Go version before merging and uploading to Codecov. Reports from different Go versions are kept separate because their coverage instrumentation can differ. See [CI grouping and coverage](internal/testing/README.md#ci-grouping-and-coverage) for failure and cancellation behavior.

Each invocation writes profiles under a unique `.coverage/run.XXXXXX/` directory, preserving module paths. `COVERAGE_PROFILE` selects the profile filename (default `coverage.out`, without directory components); HTML reports use that filename plus `.html`. The artifact directory is logged and retained for inspection, including on failure. Old profiles elsewhere in the repository are never merged.

The final report defaults to `coverage.txt`, as expected by CI. Use `COVERAGE_OUTPUT` to choose another destination. Concurrent invocations must use different output paths; a second invocation targeting an output already in use fails without touching the active run's report.

```sh
make test-coverage ALL_COVERAGE_MOD_DIRS='./cache' COVERAGE_OUTPUT=coverage.cache.txt
```

After acquiring its output lock, a run removes the previous final report. It publishes a replacement only after tests, HTML generation, and merging all succeed. Missing profiles, an empty module selection, or a run with no covered statements fail without publishing a final report. Modules with header-only profiles contribute no statements and are skipped during HTML generation and merging.

## Release Checks

`versions.yaml` is the release manifest. Its module sets define candidate versions and published module paths; `excluded-modules` lists tools and standalone examples. `make verify-mods` checks that every local module is accounted for, set versions are valid, and intra-repository dependency constraints are consistent. It does not check public API compatibility.

The release check targets first run `verify-mods`, then resolve manifest module paths to local directories. They skip excluded modules, reject unknown sets/directories and empty selections, and check selected modules sequentially. Each module's full diagnostic output and exit status are retained. Checks continue after a module fails, and the overall command exits nonzero if any check fails. Metadata, selection, and tool-build failures stop before module checks.

| Command | Purpose |
| --- | --- |
| `make gorelease` | Compare every release module with an inferred published baseline and suggest its next version |
| `make gorelease MODSET=stable` | Compare only the named module set |
| `make gorelease/cache` | Compare a single manifest-listed directory; use `gorelease/.` for the root module |
| `make release-check MODSET=stable` | Validate every selected module against its candidate version from `versions.yaml` |

Omitting `MODSET` selects all sets. `MODSET` accepts one set name. For a focused candidate check, use `make release-check MODSET=stable RELEASE_DIR=cache`. `release-check` always uses the manifest version; it does not use `GORELEASE_VERSION`.

### Baselines and candidate versions

An empty `GORELEASE_BASE` preserves the pinned gorelease tool's inference: it selects an applicable released version, taking the proposed version into account when one is supplied. Set an explicit baseline for reproducible comparisons. `GORELEASE_VERSION` supplies a proposed version to the diagnostic `gorelease` targets:

```sh
make gorelease/cache GORELEASE_BASE=v4.2.0
make gorelease/cache GORELEASE_BASE=v4.2.0 GORELEASE_VERSION=v4.3.0
make release-check MODSET=stable GORELEASE_BASE=v4.2.0
```

Candidate versions must be unpublished, valid for each module path, and consistent with its API changes. Updating only the version number cannot make an incompatible `/v4` API change acceptable. For a new major release, migrate module paths and dependencies first. Compare one migrated module to its old path explicitly, for example `GORELEASE_BASE=github.com/go-fries/fries/cache/v4@v4.2.0` on a future `/v5` module. Use `GORELEASE_BASE=none` only for an intentional first release or new major with no API comparison. A missing baseline or download error remains a failure; it never falls back to `none`.

### Release sequence and prerequisites

The repository uses Make and multimod for release preparation and tagging:

1. Set the intended candidate version in `versions.yaml` and commit it.
2. Run `make prerelease MODSET=stable`. Multimod prepares version/dependency changes and creates a release branch and commit.
3. On the prepared candidate, run `make release-check MODSET=stable` with the chosen baseline and module source available. Resolve all reported failures before tagging that candidate.
4. Use `make add-tags MODSET=stable COMMIT=HEAD`, then the existing `make push-tags TAG=...` command when publishing the checked candidate.

The check commands are read-only with respect to repository files and tags. They require a clean committed working tree, the Go toolchain, and access to baseline modules and dependencies through `GOPROXY`/VCS and the configured checksum service. Gorelease loads the module as a consumer would, so local `replace` directives do not substitute unpublished sibling modules. For a coordinated release whose updated sibling versions are not yet published, provide those candidate modules through a staging module proxy before running the final check. Dependency-resolution failures must be resolved rather than suppressed.

Run checks from the exact candidate commit you intend to tag. `add-tags COMMIT=...` can target another commit; a check of the current checkout does not validate that other snapshot. The strict release entry point is `make release-check`; `prerelease` and `add-tags` retain their metadata checks and preparation/tagging roles.

Gorelease reports both incompatible APIs and loading/tool failures as nonzero results (some share exit code 1); use its preserved diagnostics to identify the cause. Ordinary CI runs metadata verification and the release-selection tool tests. Candidate compatibility checks run through the release entry point with an explicit candidate and available dependencies.
