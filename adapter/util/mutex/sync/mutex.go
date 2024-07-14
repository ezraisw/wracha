package sync

import (
	"context"
	"sync"

	"github.com/ezraisw/wracha/adapter/util/mutex"
)

type syncMutexFactory struct {
}

func NewMutexFactory() mutex.MutexFactory {
	return &syncMutexFactory{}
}

func (m syncMutexFactory) Make(key string) mutex.Mutex {
	return &syncMutex{
		mutex: &sync.Mutex{},
	}
}

type syncMutex struct {
	mutex *sync.Mutex
}

func (m syncMutex) Lock(ctx context.Context) error {
	m.mutex.Lock()
	return nil
}

func (m syncMutex) Unlock(ctx context.Context) error {
	m.mutex.Unlock()
	return nil
}
