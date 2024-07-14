package sync

import (
	"context"

	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/adapter/util/mutex"
)

type syncMutexLocker struct {
	mm *mutex.MultiMutex
}

func NewLocker() mutex.Locker {
	return &syncMutexLocker{
		mm: mutex.NewMultiMutex(NewMutexFactory()),
	}
}

func (lr syncMutexLocker) Obtain(ctx context.Context, key string) (mutex.Lock, error) {
	if err := lr.mm.Lock(ctx, key); err != nil {
		return nil, adapter.ErrFailedLock
	}

	return &syncMutexLock{
		mm:  lr.mm,
		key: key,
	}, nil
}

type syncMutexLock struct {
	mm  *mutex.MultiMutex
	key string
}

func (l syncMutexLock) Release(ctx context.Context) error {
	if err := l.mm.Unlock(ctx, l.key); err != nil {
		return adapter.ErrFailedUnlock
	}
	return nil
}
