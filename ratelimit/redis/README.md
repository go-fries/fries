# Rate Limit Redis Store

`ratelimit/redis` shares GCRA state across application processes through Redis.
Its Lua script uses one Redis key and Redis server time for each atomic
decision.

## Installation

```bash
go get github.com/go-fries/fries/ratelimit/v4
go get github.com/go-fries/fries/ratelimit/redis/v4
go get github.com/redis/go-redis/v9
```

## Usage

```go
client := redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

store := ratelimitredis.New(client)
limiter, err := ratelimit.New(store, ratelimit.PerMinute(100))
if err != nil {
	return err
}

decision, err := limiter.Allow(ctx, "api:user:"+userID)
```

The default Redis key prefix is `fries:ratelimit:`. Use a distinct application
prefix when Redis is shared:

```go
store := ratelimitredis.New(
	client,
	ratelimitredis.WithPrefix("my-service:ratelimit"),
)
```

Trailing colons are normalized. An empty prefix option is ignored.

Accepted decisions update the key and set a TTL equal to its recovery time.
Rejected decisions do not change the stored state. `Reset` deletes the
prefixed key.

The Store accepts `redis.UniversalClient`, including standalone, sentinel, and
cluster clients. The Lua script accesses one key, but deployment-specific Redis
permissions must allow `EVAL`, `TIME`, `GET`, `SET`, and `DEL`.

## Performance

The Lua script benchmark measures the complete go-redis round trip for allowed,
rejected, and concurrent decisions. On an Apple M1 Pro using Go 1.26.2 and a
local Redis instance, three two-second runs produced:

| Path | Time | Memory | Allocations |
| --- | ---: | ---: | ---: |
| Allowed | 218–344 µs/op | 480 B/op | 14 allocs/op |
| Rejected | 205–229 µs/op | 488 B/op | 15 allocs/op |
| Parallel, same key | 54–63 µs/op | 472 B/op | 13 allocs/op |

The parallel result used eight benchmark workers and represents aggregate
throughput of approximately 16,000–18,500 decisions per second. Results depend
on network latency, Redis configuration, client pool settings, and hardware;
benchmark the intended deployment before selecting production limits.

From a repository checkout, run the benchmark against Redis at
`localhost:6379`, or set `REDIS_ADDR`:

```bash
cd ratelimit/redis
go test -run '^$' \
  -bench '^BenchmarkTakeScript$' \
  -benchmem \
  -benchtime=2s \
  -count=3
```

If a command is committed but its response is lost, callers cannot know
whether capacity was consumed. Do not blindly retry ambiguous Store errors.
