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
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
}
