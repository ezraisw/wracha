package redislock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bsm/redislock"
	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/adapter/util/mutex"
)

type redislockMutexFactory struct {
	lc      *redislock.Client
	lockTtl time.Duration
}

func NewMutexFactory(client redislock.RedisClient, lockTtl time.Duration) mutex.MutexFactory {
	return &redislockMutexFactory{
		lc:      redislock.New(client),
		lockTtl: lockTtl,
	}
}

func (f redislockMutexFactory) Make(key string) mutex.Mutex {
	return &redislockMutex{
		lc:      f.lc,
		lockTtl: f.lockTtl,
		key:     key,
	}
}

type redislockMutex struct {
	mu          sync.Mutex
	currentLock *redislock.Lock

	lc      *redislock.Client
	lockTtl time.Duration
	key     string
}

func (m *redislockMutex) Lock(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lock, err := m.lc.Obtain(ctx, m.key, m.lockTtl, &redislock.Options{})
	if err != nil {
		return adapter.ErrFailedLock
	}

	m.currentLock = lock
	return nil
}

func (m *redislockMutex) Unlock(ctx context.Context) error {
	if m.currentLock == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.currentLock.Release(ctx)
	if err != nil && !errors.Is(err, redislock.ErrLockNotHeld) {
		return adapter.ErrFailedUnlock
	}

	m.currentLock = nil
	return nil
}
