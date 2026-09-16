# Test module inventory

Audit date: 2026-09-16. This inventory covers all 94 tracked Go modules, assigning each test file to its nearest `go.mod` rather than assuming root `go test ./...` traverses nested modules.

`make test-unit` runs all component and example modules with `-short`. `make test-integration` runs the full suites of the seven service-backed modules declared in [modules.mk](modules.mk), including their in-process tests. It uses non-short, uncached, verbose execution. Missing services fail full runs.

## Service-backed modules

| Module | Service | Setup and representative integration tests |
| --- | --- | --- |
| `cache/redis` | Redis | `newTestStore`; `TestRedis_Base`, `TestRedis_Flush`, `TestRedis_Lock` |
| `gorm/scope` | MySQL | `newTestDB`; `TestScopes`, `TestPagination`, `TestHelpers_OrderBy` |
| `idempotency/redis` | Redis | `newTestStore`; `TestStoreLifecycle`, `TestStoreOnlyGrantsOneConcurrentClaim` |
| `locker/redis` | Redis | `newRedis`; `TestLockTryAcquire`, `TestLeaseRefresh` |
| `mysql/canal/positioner/redis` | Redis | `createRedisClient`; `TestPositioner`, `TestBufferedPositioner` |
| `queue/adapter/redis` | Redis | `newRedisTestQueue`; `TestQueue_DeadLetterWritesReasonAndAcksDelivery`, `TestQueue_DelayedTaskPromotion` |
| `ratelimit/redis` | Redis | `newRedisClient`; `TestStoreBurstAndRecovery`, `TestStoreConcurrentTakeDoesNotExceedBurst` |

Redis helpers read `REDIS_ADDR` (default `localhost:6379`), skip before connecting in short mode, and otherwise require a successful Ping. They set connection/read/write timeouts, use unique namespaces, delete only owned keys with bounded cleanup contexts, and close clients after cleanup. RateLimit's Redis benchmark uses the same connection contract but runs only when explicitly selected with `-bench`.

GORM Scope reads `MYSQL_DSN`, defaults to the CI `gorm` database, and owns a randomly prefixed table per test. It requires table-level permissions in an existing database. MySQL Canal Positioner stores binlog positions in Redis; these tests do not connect to MySQL.

## Service-independent modules

The following 79 modules have tests requiring no separately provisioned service. This includes tests using local sockets, temporary files, fakes, and test-owned HTTP servers.

| Module | Module | Module |
| --- | --- | --- |
| `.` | `batcher` | `cache` |
| `chi` | `cloudevents/eventdispatcher` | `codec` |
| `codec/json` | `codec/msgpack` | `codec/proto` |
| `codec/sonic` | `codec/xml` | `codec/yaml` |
| `config` | `crontab` | `debounce` |
| `eino/components/embedding/cached` | `eino/components/embedding/cached/cacher/redis` | `encrypter` |
| `ent` | `ent/multidriver` | `env` |
| `errors` | `event` | `event/middleware/recovery` |
| `filesystem` | `filesystem/local` | `filesystem/oss` |
| `filesystem/s3` | `gin` | `gorm/logger/multi` |
| `gorm/logger/otel` | `hashing` | `hashing/md5` |
| `health` | `http/response` | `http/server` |
| `hyperf/jet` | `hyperf/jet/middleware/logger` | `hyperf/jet/middleware/otel` |
| `hyperf/jet/middleware/recovery` | `hyperf/jet/middleware/retry` | `hyperf/jet/middleware/timeout` |
| `idempotency` | `idempotency/memory` | `jsonrpc` |
| `kratos/middleware/cors` | `kratos/middleware/otel` | `kratos/middleware/protovalidate` |
| `lifecycle` | `locker` | `log/slog` |
| `log/slog/multi` | `log/slog/syslog` | `mysql/canal/server` |
| `otel/otlp` | `parallel` | `poll` |
| `ptr` | `queue` | `queue/adapter/memory` |
| `queue/adapter/rabbitmq` | `queue/kratos/server` | `queue/middleware/recovery` |
| `ratelimit` | `ratelimit/memory` | `recovery` |
| `retry` | `signal` | `slices` |
| `strings` | `support` | `time/period` |
| `timezone` | `udp` | `webhook` |
| `webhook/sender` | `x/container` | `x/pagination` |
| `x/prints` | | |

Notable audit findings:

- Eino Redis cacher tests use `mockRedisClient`; they do not use a Redis server or an embedding API.
- RabbitMQ queue tests use controllable publisher/channel fakes; filesystem S3/OSS tests use fake clients.
- HTTP/JSON-RPC/Webhook tests use test servers or supplied transports. Endpoint strings in configuration tests and examples are not evidence of outbound requests.
- OTLP tests use test exporters/transports; GORM logger tests do not open a database.
- UDP, HTTP servers, and syslog may create local listeners. Filesystem tests use temporary directories and have platform-specific symlink skips. These remain in the unit entry point.
- MySQL Canal Server tests cover construction and lifecycle adaptation without running a live Canal source.

## Modules without test files

| Module | Current validation |
| --- | --- |
| `capability` | Compiled by the unit target |
| `cloudevents/protocol/amqp091` | Compiled; no live AMQP test |
| `constraints` | Compiled by the unit target |
| `eino/components/embedding/cached/cacher/gorm` | Compiled; no database test |
| `eino/components/embedding/cached/example` | Example module compiled by the unit target |
| `mysql/canal` | Compiled; no live MySQL replication test |
| `queue/examples/tasker` | Example module compiled by the unit target |
| `internal/tools` | Excluded from component test targets; pinned build tools |

## Maintenance

When a module gains service-backed tests, update `modules.mk` and move it to the corresponding inventory section. Keep short-mode checks before service initialization, make full-mode connection failures fatal, and register cleanup before creating test data. Use `make test/MODULE` for deliberately filtered diagnostic runs; CI runs complete suites with verbose output.

This audit describes the current test suite, not backend capabilities. Missing tests for a backend remain coverage gaps rather than implicit integration coverage.

## CI grouping and coverage

The Go Test workflow partitions the 91 coverage modules using `modules.mk` and Make's existing module discovery:

| Group | Modules | Go versions | Services | Test mode |
| --- | --- | --- | --- | --- |
| `unit` | 84: all coverage modules except service modules | 1.26.x, 1.27.x | None | Short |
| `redis` | 6: `REDIS_TEST_MOD_DIRS` | 1.26.x, 1.27.x | Redis | Full |
| `mysql-latest` | 1: `MYSQL_TEST_MOD_DIRS` | 1.26.x, 1.27.x | MySQL latest | Full |
| `mysql-5.7` | Same MySQL module | 1.26.x, 1.27.x | MySQL 5.7 | Full |

Every test job uses `-race -v -count=1`. Service-module in-process tests run with their service group, so the unit group can exclude these modules entirely. New service-independent modules enter the unit group automatically. `make verify-test-groups` rejects overlapping Redis/MySQL groups or service modules outside the coverage selection. Example and tool modules retain the existing coverage exclusions; the broader contributor command `make test-unit` still includes examples.

The matrix is limited to four concurrent jobs and leaves sibling jobs running after a failure, preserving diagnostics across Go/service versions. A newer run for the same PR or branch cancels the older run. Each test job has a 20-minute job timeout, in addition to Go's per-package timeout. No module-level parallel subprocesses are added within a runner.

Successful jobs upload `coverage-GO-SUITE` artifacts. Two downstream coverage jobs each download exactly one Go version's reports and explicitly require nonempty atomic profiles for `unit`, `redis`, `mysql-latest`, and `mysql-5.7`. They merge only those four paths with the pinned `gocovmerge`, retain `merged-coverage-GO`, and upload that report to Codecov with automatic report discovery disabled. MySQL profiles are combined within each Go version; profiles from different Go versions are never passed to the same merge. Artifacts are retained for seven days.

Coverage jobs depend on the entire test matrix succeeding. Missing artifacts, empty/header-only profiles, malformed merge input, or upload failures fail the workflow; partial test runs are not published as complete coverage.

### Sizing baseline

The preceding [successful run](https://github.com/go-fries/fries/actions/runs/35056077727) ran all 91 modules in each of four jobs. Job durations were 300–369 seconds, totaling 1,353 runner-seconds; test steps took 249–320 seconds each. Measured per-module wall time (including command/build and HTML-report overhead) was largest for Gin (19–26s), the root module (18–25s), Cache Redis (17–22s), and MySQL Canal (13–19s).

Grouping reduces module-suite executions from 364 to 184 per workflow while retaining both Go versions and both MySQL versions. Only GORM Scope is repeated across the MySQL matrix. The initial concurrency cap matches the previous four runners; avoiding additional unit shards limits repeated dependency compilation and runner setup. Compare actual job time and summed runner time after CI execution rather than treating the execution-count reduction as a measured speedup.
