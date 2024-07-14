package redsync

import (
	"context"

	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/adapter/util/mutex"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis"
)

type redsyncMutexFactory struct {
	rs *redsync.Redsync
}

func NewMutexFactory(pools ...redis.Pool) mutex.MutexFactory {
	return &redsyncMutexFactory{
		rs: redsync.New(pools...),
	}
}

func (f redsyncMutexFactory) Make(key string) mutex.Mutex {
	return &redsyncMutex{
		mutex: f.rs.NewMutex(key),
	}
}

type redsyncMutex struct {
	mutex *redsync.Mutex
}

func (m redsyncMutex) Lock(ctx context.Context) error {
	err := m.mutex.LockContext(ctx)
	if err != nil {
		return adapter.ErrFailedLock
	}
	return nil
}

func (m redsyncMutex) Unlock(ctx context.Context) error {
	ok, err := m.mutex.UnlockContext(ctx)
	if err != nil || !ok {
		return adapter.ErrFailedUnlock
	}
	return nil
}
