package goredis

import (
	"context"
	"errors"
	"time"

	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/adapter/util/mutex"
	"github.com/ezraisw/wracha/adapter/util/mutex/redislock"
	"github.com/redis/go-redis/v9"
)

type goredisAdapter struct {
	client redis.UniversalClient
	locker mutex.Locker

	// Deprecated
	multiMutex *mutex.MultiMutex
}

const DefaultLockTTL = 8 * time.Minute

func NewAdapter(client redis.UniversalClient) adapter.Adapter {
	return NewAdapterWithLockTTL(client, DefaultLockTTL)
}

func NewAdapterWithLockTTL(client redis.UniversalClient, lockTtl time.Duration) adapter.Adapter {
	return &goredisAdapter{
		client: client,
		locker: redislock.NewLocker(client, lockTtl),

		multiMutex: mutex.NewMultiMutex(redislock.NewMutexFactory(client, lockTtl)),
	}
}

func (a goredisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	count, err := a.client.Exists(ctx, key).Uint64()
	if err != nil {
		return false, err
	}

	return count != 0, nil
}

func (a goredisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := a.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = adapter.ErrNotFound
		}

		return nil, err
	}

	return data, nil
}

func (a goredisAdapter) Set(ctx context.Context, key string, ttl time.Duration, data []byte) error {
	return a.client.Set(ctx, key, data, ttl).Err()
}

func (a goredisAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

// Deprecated
func (a goredisAdapter) Lock(ctx context.Context, key string) error {
	return a.multiMutex.Lock(ctx, key)
}

// Deprecated
func (a goredisAdapter) Unlock(ctx context.Context, key string) error {
	return a.multiMutex.Unlock(ctx, key)
}

func (a goredisAdapter) ObtainLock(ctx context.Context, key string) (adapter.Lock, error) {
	return a.locker.Obtain(ctx, key)
}
