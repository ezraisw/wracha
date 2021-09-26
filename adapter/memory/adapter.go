package memory

import (
	"context"
	"time"

	"github.com/karlseguin/ccache/v2"
	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/adapter/util/mutex"
	"github.com/pwnedgod/wracha/adapter/util/mutex/sync"
)

type memoryAdapter struct {
	cacheCfg   *ccache.Configuration
	cache      *ccache.Cache
	multiMutex *mutex.MultiMutex
}

func NewAdapter() adapter.Adapter {
	return &memoryAdapter{
		cacheCfg:   ccache.Configure(),
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

func (a *memoryAdapter) Lock(ctx context.Context, key string) error {
	return a.multiMutex.Lock(ctx, key)
}

func (a *memoryAdapter) Unlock(ctx context.Context, key string) error {
	return a.multiMutex.Unlock(ctx, key)
}
