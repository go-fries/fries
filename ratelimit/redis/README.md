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

If a command is committed but its response is lost, callers cannot know
whether capacity was consumed. Do not blindly retry ambiguous Store errors.
