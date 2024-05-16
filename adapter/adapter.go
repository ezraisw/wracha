package adapter

import (
	"context"
	"time"
)

type Adapter interface {
	Exists(ctx context.Context, key string) (bool, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, ttl time.Duration, data []byte) error
	Delete(ctx context.Context, key string) error

	// Deprecated
	Lock(ctx context.Context, key string) error

	// Deprecated
	Unlock(ctx context.Context, key string) error

	ObtainLock(ctx context.Context, key string) (Lock, error)
}

type Lock interface {
	Release(ctx context.Context) error
}
