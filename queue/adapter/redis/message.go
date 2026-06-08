package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-fries/fries/queue/v3"
	goredis "github.com/redis/go-redis/v9"
)

func (q *Queue) claimPending(ctx context.Context, name string, visibilityTimeout time.Duration) (queue.Lease, error) {
	messages, _, err := q.redis.XAutoClaim(ctx, &goredis.XAutoClaimArgs{
		Stream:   q.streamKey(name),
		Group:    q.group,
		Consumer: q.consumer,
		MinIdle:  visibilityTimeout,
		Start:    "0-0",
		Count:    1,
	}).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, nil
	}
	return q.leaseFromClaimedMessage(ctx, name, messages[0])
}

func (q *Queue) leaseFromMessage(message goredis.XMessage) (queue.Lease, error) {
	return q.leaseFromMessageWithDeliveryCount(message, 0)
}

func (q *Queue) leaseFromClaimedMessage(ctx context.Context, name string, message goredis.XMessage) (queue.Lease, error) {
	deliveryCount, err := q.deliveryCount(ctx, name, message.ID)
	if err != nil {
		return nil, err
	}
	return q.leaseFromMessageWithDeliveryCount(message, deliveryCount)
}

func (q *Queue) deliveryCount(ctx context.Context, name, messageID string) (int64, error) {
	pending, err := q.redis.XPendingExt(ctx, &goredis.XPendingExtArgs{
		Stream: q.streamKey(name),
		Group:  q.group,
		Start:  messageID,
		End:    messageID,
		Count:  1,
	}).Result()
	if errors.Is(err, goredis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	return pending[0].RetryCount, nil
}

func (q *Queue) leaseFromMessageWithDeliveryCount(message goredis.XMessage, deliveryCount int64) (queue.Lease, error) {
	value, ok := message.Values[taskField]
	if !ok {
		return nil, fmt.Errorf("queue/adapter/redis: message %s missing %q field", message.ID, taskField)
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("queue/adapter/redis: message %s has unsupported %q field", message.ID, taskField)
	}

	var task queue.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	task.Attempt = attemptWithDeliveryCount(task.Attempt, deliveryCount)
	return &redisLease{
		task:     &task,
		streamID: message.ID,
	}, nil
}

func attemptWithDeliveryCount(baseAttempt int, deliveryCount int64) int {
	if baseAttempt < 0 {
		baseAttempt = 0
	}
	if deliveryCount <= 0 {
		if baseAttempt == math.MaxInt {
			return math.MaxInt
		}
		return baseAttempt + 1
	}

	if baseAttempt >= math.MaxInt {
		return math.MaxInt
	}
	maxRemaining := math.MaxInt - baseAttempt
	if deliveryCount > int64(maxRemaining) {
		return math.MaxInt
	}
	return baseAttempt + int(deliveryCount)
}
