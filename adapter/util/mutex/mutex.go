package mutex

import "context"

type MutexFactory interface {
	Make(key string) Mutex
}

type Mutex interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
}
