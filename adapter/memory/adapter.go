package memory

import (
	"context"
	"time"

	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/adapter/util/mutex"
	"github.com/ezraisw/wracha/adapter/util/mutex/sync"
	"github.com/karlseguin/ccache/v2"
)

type memoryAdapter struct {
	cacheCfg *ccache.Configuration
	cache    *ccache.Cache
	locker   mutex.Locker

	// Deprecated
	multiMutex *mutex.MultiMutex
}

func NewAdapter() adapter.Adapter {
	return &memoryAdapter{
		cacheCfg: ccache.Configure(),
		locker:   sync.NewLocker(),

		multiMutex: mutex.NewMultiMutex(sync.NewMutexFactory()),
	}
}

func NewAdapterWithConfiguration(cacheCfg *ccache.Configuration) adapter.Adapter {
	return &memoryAdapter{
		cacheCfg: cacheCfg,
	}
}

func (a *memoryAdapter) getCache() *ccache.Cache {
	// Lazily create the instance.
	if a.cache == nil {
		a.cache = ccache.New(a.cacheCfg)
	}

	return a.cache
}

func (a *memoryAdapter) Exists(ctx context.Context, key string) (bool, error) {
	item := a.getCache().Get(key)
	return item != nil && !item.Expired(), nil
}

func (a *memoryAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	item := a.getCache().Get(key)
	if item == nil || item.Expired() {
		return nil, adapter.ErrNotFound
	}

	// Ignore casting errors.
	value := item.Value().([]byte)

	return value, nil
}

func (a *memoryAdapter) Set(ctx context.Context, key string, ttl time.Duration, data []byte) error {
	a.getCache().Set(key, data, ttl)
	return nil
}

func (a *memoryAdapter) Delete(ctx context.Context, key string) error {
	a.getCache().Delete(key)
	return nil
}

func (a memoryAdapter) Lock(ctx context.Context, key string) error {
	return a.multiMutex.Lock(ctx, key)
}

func (a memoryAdapter) Unlock(ctx context.Context, key string) error {
	return a.multiMutex.Unlock(ctx, key)
}

func (a memoryAdapter) ObtainLock(ctx context.Context, key string) (adapter.Lock, error) {
	return a.locker.Obtain(ctx, key)
}
