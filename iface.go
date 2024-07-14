package wracha

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/ezraisw/wracha/adapter"
	"github.com/ezraisw/wracha/codec"
	"github.com/ezraisw/wracha/logger"
	"github.com/vmihailenco/msgpack/v5"
)

type (
	ActionFunc[T any] func(ctx context.Context) (ActionResult[T], error)

	PreActionErrorHandlerFunc[T any] func(ctx context.Context, args PreActionErrorHandlerArgs[T]) (T, error)

	PostActionErrorHandlerFunc[T any] func(ctx context.Context, args PostActionErrorHandlerArgs[T]) (T, error)

	PreActionErrorHandlerArgs[T any] struct {
		Key         Keyable
		Action      ActionFunc[T]
		ErrCategory string
		Err         error
	}

	PostActionErrorHandlerArgs[T any] struct {
		Key         Keyable
		Action      ActionFunc[T]
		Result      ActionResult[T]
		ErrCategory string
		Err         error
	}

	ActionResult[T any] struct {
		// Whether to cache the returned values.
		Cache bool

		// The TTL of the cached values. If set to zero, defaults to the actor settings.
		TTL time.Duration

		// The values to cache and return.
		Value T
	}

	ActorOptions struct {
		Adapter adapter.Adapter
		Codec   codec.Codec
		Logger  logger.Logger
	}

	Actor[T any] interface {
		// Set default TTL of cache.
		SetTTL(ttl time.Duration) Actor[T]

		// Set error handler for handling unconventional errors thrown before action (get in cache and lock).
		//
		// Value and error returned by the handler will be forwarded as a return value for Actor.Do.
		SetPreActionErrorHandler(handler PreActionErrorHandlerFunc[T]) Actor[T]

		// Set error handler for handling unconventional errors thrown after action (store).
		//
		// Value and error returned by the handler will be forwarded as a return value for Actor.Do.
		SetPostActionErrorHandler(handler PostActionErrorHandlerFunc[T]) Actor[T]

		// Invalidate the value of the given key.
		Invalidate(ctx context.Context, key Keyable) error

		// Perform an action.
		// The action will not be executed again if the key exists in cache.
		Do(ctx context.Context, key Keyable, action ActionFunc[T]) (T, error)
	}

	Keyable interface {
		Key() (string, error)
	}

	KeyableMap map[string]any
	KeyableStr string
)

func (m KeyableMap) Key() (string, error) {
	hash := sha1.New()
	if err := msgpack.NewEncoder(hash).Encode(m); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (m KeyableStr) Key() (string, error) {
	return string(m), nil
}
