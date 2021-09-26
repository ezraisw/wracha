package adapter

import "errors"

var (
	ErrNotFound     = errors.New("wracha: not found")
	ErrFailedLock   = errors.New("wracha: failed lock")
	ErrFailedUnlock = errors.New("wracha: failed unlock")
)
