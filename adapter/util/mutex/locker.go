package mutex

import "context"

type Locker interface {
	Obtain(ctx context.Context, key string) (Lock, error)
}

type Lock interface {
	Release(ctx context.Context) error
}
