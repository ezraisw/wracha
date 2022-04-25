package redigo

import (
	"context"
	"errors"
	"time"

	rsredigo "github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/gomodule/redigo/redis"
	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/adapter/util/mutex"
	"github.com/pwnedgod/wracha/adapter/util/mutex/redsync"
)

type redigoAdapter struct {
	pool       *redis.Pool
	multiMutex *mutex.MultiMutex
}

func NewAdapter(pool *redis.Pool) adapter.Adapter {
	return &redigoAdapter{
		pool:       pool,
		multiMutex: mutex.NewMultiMutex(redsync.NewMutexFactory(rsredigo.NewPool(pool))),
	}
}

func (a redigoAdapter) Exists(ctx context.Context, key string) (bool, error) {
	conn := a.pool.Get()
	defer a.pool.Close()

	count, err := redis.Int64(conn.Do(CommandExists, key))
	if err != nil {
		return false, err
	}

	return count != 0, nil
}

func (a redigoAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	conn := a.pool.Get()
	defer a.pool.Close()

	data, err := redis.Bytes(conn.Do(CommandGet, key))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			err = adapter.ErrNotFound
		}

		return nil, err
	}

	return data, nil
}

func (a redigoAdapter) Set(ctx context.Context, key string, ttl time.Duration, value []byte) error {
	args := []any{
		key, value,
	}

	if ttl > 0 {
		args = append(args, formatExpirationArgs(ttl)...)
	}

	conn := a.pool.Get()
	defer a.pool.Close()

	_, err := conn.Do(CommandSet, args...)
	return err
}

func (a redigoAdapter) Delete(ctx context.Context, key string) error {
	conn := a.pool.Get()
	defer a.pool.Close()

	_, err := conn.Do(CommandDel, key)
	return err
}

func (a redigoAdapter) Lock(ctx context.Context, key string) error {
	return a.multiMutex.Lock(ctx, key)
}

func (a redigoAdapter) Unlock(ctx context.Context, key string) error {
	return a.multiMutex.Unlock(ctx, key)
}
