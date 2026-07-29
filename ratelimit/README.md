# Rate Limit

`ratelimit` provides immediate, key-based rate-limit decisions with consistent
memory and Redis stores. A limiter binds one policy and can be shared safely
across concurrent requests.

## Installation

Choose a Store for the deployment model:

```bash
go get github.com/go-fries/fries/ratelimit/v4
go get github.com/go-fries/fries/ratelimit/memory/v4
```

Use `ratelimit/redis` instead of `ratelimit/memory` when multiple application
instances must share capacity.

## Usage

Create one Limiter for each policy and reuse it:

```go
store := memory.New()
limiter, err := ratelimit.New(store, ratelimit.PerMinute(100))
if err != nil {
	return err
}

decision, err := limiter.Allow(ctx, "api:user:"+userID)
if err != nil {
	return err
}
if !decision.Allowed {
	return newTooManyRequestsError(decision.RetryAfter)
}
```

A rejected request is a normal decision and returns a nil error. Errors are
reserved for invalid arguments, canceled contexts, and Store failures.

Use a custom burst when the maximum immediate capacity should differ from the
period rate:

```go
limit := ratelimit.PerMinute(100)
limit.Burst = 20

limiter, err := ratelimit.New(store, limit)
```

Use `AllowN` when one operation consumes multiple units:

```go
decision, err := limiter.AllowN(
	ctx,
	"exports:tenant:"+tenantID,
	request.ItemCount,
)
```

The requested cost is consumed completely or not at all. A cost below one or
above the configured burst returns `ErrInvalidCost`.

`Reset` explicitly restores the initial capacity for one key:

```go
err := limiter.Reset(ctx, "api:user:"+userID)
```

## Decision fields

- `Allowed` reports whether the requested units were consumed.
- `Remaining` is the maximum number of units available immediately afterward.
- `RetryAfter` is the minimum delay before a rejected cost can be accepted. It
  is zero for an allowed request.
- `ResetAfter` is the time until the key has recovered its complete burst.
- `Limit` is the policy used for the decision.

The component uses GCRA and rounds each replenishment interval up to a
microsecond. Limits whose exact interval is below one microsecond are rejected.

## Key design

Keys identify independent rate-limit state. Use stable application namespaces:

```text
login:account:123
api:tenant:42
exports:user:9
```

Do not place passwords, access tokens, phone numbers, or other sensitive values
directly in a key. Different policies should use different key namespaces.

## Boundaries

`ratelimit` does not wait, reserve future capacity, retry operations, generate
keys, or write HTTP responses and headers. Applications decide whether a
rejected request should return HTTP 429, be delayed, or be rescheduled.

If a Redis request is committed but its response is lost, the caller cannot
know whether capacity was consumed. Avoid automatically retrying Store errors
when duplicate consumption would matter.
