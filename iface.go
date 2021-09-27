package wracha

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type (
	ActionFunc func(ctx context.Context) (ActionResult, error)

	PreActionErrorHandlerFunc func(ctx context.Context, args PreActionErrorHandlerArgs) (interface{}, error)

	PostActionErrorHandlerFunc func(ctx context.Context, args PostActionErrorHandlerArgs) (interface{}, error)

	PreActionErrorHandlerArgs struct {
		Key         interface{}
		Action      ActionFunc
		ErrCategory string
		Err         error
	}

	PostActionErrorHandlerArgs struct {
		Key         interface{}
		Action      ActionFunc
		Result      ActionResult
		ErrCategory string
		Err         error
	}

	ActionResult struct {
		// Whether to cache the returned values.
		Cache bool

		// The TTL of the cached values. If set to zero, defaults to the actor settings.
		TTL time.Duration

		// The values to cache and return.
		Value interface{}
	}

	Actor interface {
		// Set default TTL of cache.
		SetTTL(ttl time.Duration) Actor

		// Set context for action and adapter.
		SetContext(context.Context) Actor

		// Set the return type of the cached values.
		// This will be used to unmarshal values from cache.
		SetReturnType(interface{}) Actor

		// Set error handler for handling unconventional errors thrown before action (get in cache and lock).
		//
		// Value and error returned by the handler will be forwarded as a return value for Actor.Do.
		SetPreActionErrorHandler(PreActionErrorHandlerFunc) Actor

		// Set error handler for handling unconventional errors thrown after action (store).
		//
		// Value and error returned by the handler will be forwarded as a return value for Actor.Do.
		SetPostActionErrorHandler(PostActionErrorHandlerFunc) Actor

		// Invalidate the value of the given key.
		Invalidate(key interface{}) error

		// Perform an action.
		// The action will not be executed again if the key exists in cache.
		Do(key interface{}, action ActionFunc) (interface{}, error)
	}

	Manager interface {
		// Create an action proposal. The name will be used as a base for the caching key.
		On(name string) Actor
	}

	Keyable interface {
		Key() (string, error)
	}

	KeyableMap map[string]interface{}
)

func (m *KeyableMap) Key() (string, error) {
	hash := sha1.New()
	if err := msgpack.NewEncoder(hash).Encode(m); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
