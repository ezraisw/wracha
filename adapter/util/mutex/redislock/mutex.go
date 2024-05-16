package redislock

import (
	"context"
	"sync"
	"time"

	"github.com/bsm/redislock"
	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/adapter/util/mutex"
)

type redislockMutexFactory struct {
	lc      *redislock.Client
	lockTTL time.Duration
}

func NewMutexFactory(client redislock.RedisClient, lockTTL time.Duration) mutex.MutexFactory {
	return &redislockMutexFactory{
		lc:      redislock.New(client),
		lockTTL: lockTTL,
	}
}

func (f redislockMutexFactory) Make(key string) mutex.Mutex {
	return &redislockMutex{
		lc:      f.lc,
		lockTTL: f.lockTTL,
		key:     key,
	}
}

type redislockMutex struct {
	mu          sync.Mutex
	currentLock *redislock.Lock

	lc      *redislock.Client
	lockTTL time.Duration
	key     string
}

func (m *redislockMutex) Lock(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lock, err := m.lc.Obtain(ctx, m.key, m.lockTTL, &redislock.Options{})
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
	if err != nil {
		return adapter.ErrFailedUnlock
	}

	m.currentLock = nil
	return nil
}
