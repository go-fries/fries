# Go Fries Components

![Supported Go Versions](https://img.shields.io/badge/Go-%3E%3D1.25.0-blue)
[![Package Version](https://badgen.net/github/release/go-fries/fries/stable)](https://github.com/go-fries/fries/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/go-fries/fries/v4)](https://pkg.go.dev/github.com/go-fries/fries/v4)
[![codecov](https://codecov.io/gh/go-fries/fries/graph/badge.svg?token=QPTHZ5L9GT)](https://codecov.io/gh/go-fries/fries)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-fries/fries)](https://goreportcard.com/report/github.com/go-fries/fries)
[![lint](https://github.com/go-fries/fries/actions/workflows/lint.yml/badge.svg)](https://github.com/go-fries/fries/actions/workflows/lint.yml)
[![tests](https://github.com/go-fries/fries/actions/workflows/test.yml/badge.svg)](https://github.com/go-fries/fries/actions/workflows/test.yml)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

> This repository has been migrated from the original `github.com/go-kratos-ecosystem/components`.

> [!IMPORTANT]
> The v4 line may include breaking changes compared with v3, please use with caution.
> Backward compatibility is the default behavior within v4, and any incompatibilities will be noted in the release.

## Installation

```bash
go get github.com/go-fries/fries/v4
```

## Components

<!-- Keep this catalog aligned with module-sets.stable in versions.yaml. -->

The catalog below lists the public modules in the stable release set. Examples
and internal development tools are not included.

| Family | Modules | Description |
| --- | --- | --- |
| Core | [`fries`](https://pkg.go.dev/github.com/go-fries/fries/v4)<br>[`errors`](https://pkg.go.dev/github.com/go-fries/fries/errors/v4)<br>[`parallel`](https://pkg.go.dev/github.com/go-fries/fries/parallel/v4)<br>[`constraints`](https://pkg.go.dev/github.com/go-fries/fries/constraints/v4)<br>[`capability`](https://pkg.go.dev/github.com/go-fries/fries/capability/v4) | Release metadata, shared error helpers, parallel processing utilities, generic constraints, and capability contracts. |
| Cache | [`cache`](https://pkg.go.dev/github.com/go-fries/fries/cache/v4)<br>[`cache/redis`](https://pkg.go.dev/github.com/go-fries/fries/cache/redis/v4) | Cache abstractions and a Redis-backed store. |
| Crontab | [`crontab`](https://pkg.go.dev/github.com/go-fries/fries/crontab/v4) | Adapts go-cron schedulers to the Kratos server lifecycle. |
| Codec | [`codec`](https://pkg.go.dev/github.com/go-fries/fries/codec/v4)<br>[`codec/json`](https://pkg.go.dev/github.com/go-fries/fries/codec/json/v4)<br>[`codec/msgpack`](https://pkg.go.dev/github.com/go-fries/fries/codec/msgpack/v4)<br>[`codec/sonic`](https://pkg.go.dev/github.com/go-fries/fries/codec/sonic/v4)<br>[`codec/proto`](https://pkg.go.dev/github.com/go-fries/fries/codec/proto/v4)<br>[`codec/xml`](https://pkg.go.dev/github.com/go-fries/fries/codec/xml/v4)<br>[`codec/yaml`](https://pkg.go.dev/github.com/go-fries/fries/codec/yaml/v4) | A shared encoding contract with JSON, MessagePack, Sonic, Protocol Buffers, XML, and YAML implementations. |
| Config | [`config`](https://pkg.go.dev/github.com/go-fries/fries/config/v4) | Type-safe configuration propagation through `context.Context`. |
| Chi | [`chi`](https://pkg.go.dev/github.com/go-fries/fries/chi/v4) | A Chi-based HTTP server adapter. |
| CloudEvents | [`cloudevents/protocol/amqp091`](https://pkg.go.dev/github.com/go-fries/fries/cloudevents/protocol/amqp091/v4)<br>[`cloudevents/eventdispatcher`](https://pkg.go.dev/github.com/go-fries/fries/cloudevents/eventdispatcher/v4) | AMQP 0.9.1 transport and event dispatching for CloudEvents. |
| Env | [`env`](https://pkg.go.dev/github.com/go-fries/fries/env/v4) | Utilities for representing and propagating application environments. |
| Encrypter | [`encrypter`](https://pkg.go.dev/github.com/go-fries/fries/encrypter/v4) | AES-CTR string encryption and decryption helpers. |
| Ent | [`ent`](https://pkg.go.dev/github.com/go-fries/fries/ent/v4)<br>[`ent/multidriver`](https://pkg.go.dev/github.com/go-fries/fries/ent/multidriver/v4) | Ent integrations for logging and routing operations across multiple drivers. |
| Event | [`event`](https://pkg.go.dev/github.com/go-fries/fries/event/v4)<br>[`event/middleware/recovery`](https://pkg.go.dev/github.com/go-fries/fries/event/middleware/recovery/v4) | Synchronous type-aware event dispatching with optional panic recovery middleware. |
| Eino | [`eino/components/embedding/cached`](https://pkg.go.dev/github.com/go-fries/fries/eino/components/embedding/cached/v4)<br>[`eino/components/embedding/cached/cacher/redis`](https://pkg.go.dev/github.com/go-fries/fries/eino/components/embedding/cached/cacher/redis/v4)<br>[`eino/components/embedding/cached/cacher/gorm`](https://pkg.go.dev/github.com/go-fries/fries/eino/components/embedding/cached/cacher/gorm/v4) | Cached Eino embeddings with Redis and GORM cache backends. |
| Lifecycle | [`lifecycle`](https://pkg.go.dev/github.com/go-fries/fries/lifecycle/v4) | Context-aware application startup, execution, rollback, and graceful shutdown. |
| Filesystem | [`filesystem`](https://pkg.go.dev/github.com/go-fries/fries/filesystem/v4)<br>[`filesystem/s3`](https://pkg.go.dev/github.com/go-fries/fries/filesystem/s3/v4)<br>[`filesystem/local`](https://pkg.go.dev/github.com/go-fries/fries/filesystem/local/v4)<br>[`filesystem/oss`](https://pkg.go.dev/github.com/go-fries/fries/filesystem/oss/v4) | A logical-path storage contract with Amazon S3, local, and Alibaba Cloud OSS drivers. |
| Gin | [`gin`](https://pkg.go.dev/github.com/go-fries/fries/gin/v4) | A Gin-based HTTP server adapter. |
| GORM | [`gorm/logger/multi`](https://pkg.go.dev/github.com/go-fries/fries/gorm/logger/multi/v4)<br>[`gorm/logger/otel`](https://pkg.go.dev/github.com/go-fries/fries/gorm/logger/otel/v4)<br>[`gorm/scope`](https://pkg.go.dev/github.com/go-fries/fries/gorm/scope/v4) | Multi-target and OpenTelemetry loggers plus reusable query scopes for GORM. |
| Hyperf | [`hyperf/jet`](https://pkg.go.dev/github.com/go-fries/fries/hyperf/jet/v4)<br>[`hyperf/jet/middleware/logger`](https://pkg.go.dev/github.com/go-fries/fries/hyperf/jet/middleware/logger/v4)<br>[`hyperf/jet/middleware/recovery`](https://pkg.go.dev/github.com/go-fries/fries/hyperf/jet/middleware/recovery/v4)<br>[`hyperf/jet/middleware/retry`](https://pkg.go.dev/github.com/go-fries/fries/hyperf/jet/middleware/retry/v4)<br>[`hyperf/jet/middleware/timeout`](https://pkg.go.dev/github.com/go-fries/fries/hyperf/jet/middleware/timeout/v4)<br>[`hyperf/jet/middleware/otel`](https://pkg.go.dev/github.com/go-fries/fries/hyperf/jet/middleware/otel/v4) | A Hyperf-compatible RPC client with logging, recovery, retry, timeout, and OpenTelemetry middleware. |
| Hashing | [`hashing`](https://pkg.go.dev/github.com/go-fries/fries/hashing/v4)<br>[`hashing/md5`](https://pkg.go.dev/github.com/go-fries/fries/hashing/md5/v4) | Reusable deterministic digest helpers and an MD5 implementation. |
| Health | [`health`](https://pkg.go.dev/github.com/go-fries/fries/health/v4) | Named, context-aware application health checks with structured reports and HTTP probe handling. |
| HTTP | [`http/server`](https://pkg.go.dev/github.com/go-fries/fries/http/server/v4) | A common HTTP server wrapper that can participate in the Kratos lifecycle. |
| JSON-RPC | [`jsonrpc`](https://pkg.go.dev/github.com/go-fries/fries/jsonrpc/v4) | A lightweight JSON-RPC 2.0 client. |
| Log | [`log/slog/multi`](https://pkg.go.dev/github.com/go-fries/fries/log/slog/multi/v4)<br>[`log/slog/syslog`](https://pkg.go.dev/github.com/go-fries/fries/log/slog/syslog/v4) | `log/slog` handlers for fan-out logging and syslog. |
| Kratos | [`kratos/middleware/cors`](https://pkg.go.dev/github.com/go-fries/fries/kratos/middleware/cors/v4)<br>[`kratos/middleware/protovalidate`](https://pkg.go.dev/github.com/go-fries/fries/kratos/middleware/protovalidate/v4)<br>[`kratos/middleware/otel`](https://pkg.go.dev/github.com/go-fries/fries/kratos/middleware/otel/v4) | CORS, protobuf validation, and OpenTelemetry middleware for Kratos. |
| Locker | [`locker`](https://pkg.go.dev/github.com/go-fries/fries/locker/v4)<br>[`locker/redis`](https://pkg.go.dev/github.com/go-fries/fries/locker/redis/v4) | Distributed lock contracts and a Redis implementation. |
| Queue | [`queue`](https://pkg.go.dev/github.com/go-fries/fries/queue/v4)<br>[`queue/adapter/memory`](https://pkg.go.dev/github.com/go-fries/fries/queue/adapter/memory/v4)<br>[`queue/adapter/rabbitmq`](https://pkg.go.dev/github.com/go-fries/fries/queue/adapter/rabbitmq/v4)<br>[`queue/adapter/redis`](https://pkg.go.dev/github.com/go-fries/fries/queue/adapter/redis/v4)<br>[`queue/kratos/server`](https://pkg.go.dev/github.com/go-fries/fries/queue/kratos/server/v4)<br>[`queue/middleware/recovery`](https://pkg.go.dev/github.com/go-fries/fries/queue/middleware/recovery/v4) | Task queue primitives, memory, RabbitMQ and Redis adapters, Kratos lifecycle integration, and recovery middleware. |
| MySQL | [`mysql/canal`](https://pkg.go.dev/github.com/go-fries/fries/mysql/canal/v4)<br>[`mysql/canal/positioner/redis`](https://pkg.go.dev/github.com/go-fries/fries/mysql/canal/positioner/redis/v4)<br>[`mysql/canal/server`](https://pkg.go.dev/github.com/go-fries/fries/mysql/canal/server/v4) | MySQL binlog processing with Redis position storage and server lifecycle integration. |
| OpenTelemetry | [`otel/otlp`](https://pkg.go.dev/github.com/go-fries/fries/otel/otlp/v4) | Global OpenTelemetry provider configuration using OTLP exporters. |
| Poll | [`poll`](https://pkg.go.dev/github.com/go-fries/fries/poll/v4) | Context-aware condition polling for eventually consistent state and asynchronous work. |
| Ptr | [`ptr`](https://pkg.go.dev/github.com/go-fries/fries/ptr/v4) | Generic pointer construction helper. |
| Recovery | [`recovery`](https://pkg.go.dev/github.com/go-fries/fries/recovery/v4) | Shared panic recovery utilities. |
| Signal | [`signal`](https://pkg.go.dev/github.com/go-fries/fries/signal/v4) | Signal handling integrated with application server lifecycles. |
| Slices | [`slices`](https://pkg.go.dev/github.com/go-fries/fries/slices/v4) | Generic slice helpers. |
| Strings | [`strings`](https://pkg.go.dev/github.com/go-fries/fries/strings/v4) | String helpers for common application code. |
| Support | [`support`](https://pkg.go.dev/github.com/go-fries/fries/support/v4) | General-purpose helper types and value utilities. |
| Timezone | [`timezone`](https://pkg.go.dev/github.com/go-fries/fries/timezone/v4) | Time zone propagation through application contexts. |
| UDP | [`udp`](https://pkg.go.dev/github.com/go-fries/fries/udp/v4) | A lifecycle-aware UDP server. |
| X | [`x/pagination`](https://pkg.go.dev/github.com/go-fries/fries/x/pagination/v4)<br>[`x/prints`](https://pkg.go.dev/github.com/go-fries/fries/x/prints/v4)<br>[`x/container`](https://pkg.go.dev/github.com/go-fries/fries/x/container/v4) | Experimental pagination, terminal output, and dependency container utilities without compatibility guarantees. |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and component design conventions.

## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
