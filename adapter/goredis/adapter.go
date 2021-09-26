package goredis

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	rsgoredis "github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/adapter/util/mutex"
	redsync "github.com/pwnedgod/wracha/adapter/util/mutex/redsync"
)

type goredisAdapter struct {
	client     redis.UniversalClient
	multiMutex *mutex.MultiMutex
}

func NewAdapter(client redis.UniversalClient) adapter.Adapter {
	return &goredisAdapter{
		client:     client,
		multiMutex: mutex.NewMultiMutex(redsync.NewMutexFactory(rsgoredis.NewPool(client))),
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

func (a goredisAdapter) Lock(ctx context.Context, key string) error {
	return a.multiMutex.Lock(ctx, key)
}

func (a goredisAdapter) Unlock(ctx context.Context, key string) error {
	return a.multiMutex.Unlock(ctx, key)
}
