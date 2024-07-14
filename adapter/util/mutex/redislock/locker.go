package redislock

import (
	"context"
	"time"

	"github.com/bsm/redislock"
	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/adapter/util/mutex"
)

type redislockLocker struct {
	lc      *redislock.Client
	lockTtl time.Duration
}

func NewLocker(client redislock.RedisClient, lockTtl time.Duration) mutex.Locker {
	return &redislockLocker{
		lc:      redislock.New(client),
		lockTtl: lockTtl,
	}
}

func (lr redislockLocker) Obtain(ctx context.Context, key string) (mutex.Lock, error) {
	lock, err := lr.lc.Obtain(ctx, key, lr.lockTtl, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.ExponentialBackoff(16*time.Millisecond, 4096*time.Millisecond), 32),
	})
	if err != nil {
		return nil, adapter.ErrFailedLock
	}
	return lock, nil
}
