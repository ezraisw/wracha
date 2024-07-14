package redsync

import (
	"context"

	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/adapter/util/mutex"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis"
)

type redsyncLocker struct {
	rs *redsync.Redsync
}

func NewLocker(pools ...redis.Pool) mutex.Locker {
	return &redsyncLocker{
		rs: redsync.New(pools...),
	}
}

func (lr redsyncLocker) Obtain(ctx context.Context, key string) (mutex.Lock, error) {
	mutex := lr.rs.NewMutex(key)

	if err := mutex.LockContext(ctx); err != nil {
		return nil, adapter.ErrFailedLock
	}

	return &redsyncLock{mutex: mutex}, nil
}

type redsyncLock struct {
	mutex *redsync.Mutex
}

func (l redsyncLock) Release(ctx context.Context) error {
	ok, err := l.mutex.UnlockContext(ctx)
	if err != nil || !ok {
		return adapter.ErrFailedUnlock
	}
	return nil
}
