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
> See [MIGRATION.md](MIGRATION.md) for the 4.x migration guide.

## 4.x Notes

Fries 4.x targets Kratos v3 and standard library `log/slog`.

- Fries import paths use `/v4`.
- Kratos integrations use `github.com/go-kratos/kratos/v3`.
- Component lifecycle loggers use `*slog.Logger`.
- Kratos-specific log modules moved to `log/slog/*`; OpenTelemetry logs should
  use the official `otelslog` bridge.
- The shared interface module moved from `contract/v4` to `capability/v4`.

## Installation

```bash
go get github.com/go-fries/fries/v4
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and component design conventions.

## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
