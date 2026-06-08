package redis

import (
	"context"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const promoteScript = `
local tasks = redis.call("zrangebyscore", KEYS[1], "-inf", ARGV[1], "limit", 0, ARGV[2])
for _, task in ipairs(tasks) do
	redis.call("xadd", KEYS[2], "*", ARGV[3], task)
	redis.call("zrem", KEYS[1], task)
end
return #tasks
`

func (q *Queue) addToStream(ctx context.Context, name string, data []byte) error {
	return q.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: q.streamKey(name),
		Values: map[string]any{
			taskField: data,
		},
	}).Err()
}

func (q *Queue) ensureGroup(ctx context.Context, name string) error {
	err := q.redis.XGroupCreateMkStream(ctx, q.streamKey(name), q.group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *Queue) promoteDue(ctx context.Context, name string) error {
	return q.redis.Eval(
		ctx, promoteScript, []string{q.delayedKey(name), q.streamKey(name)},
		time.Now().UTC().UnixNano(),
		q.promoteSize,
		taskField,
	).Err()
}
