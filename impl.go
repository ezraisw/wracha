package wracha

import (
	"context"
	"errors"
	"time"

	"github.com/ezraisw/wracha/adapter"
)

type (
	defaultActor[T any] struct {
		o                    ActorOptions
		name                 string
		ttl                  time.Duration
		preActionErrHandler  PreActionErrorHandlerFunc[T]
		postActionErrHandler PostActionErrorHandlerFunc[T]
	}
)

const (
	TTLDefault = time.Duration(10) * time.Minute
)

func NewActor[T any](name string, options ActorOptions) Actor[T] {
	if options.Adapter == nil {
		panic("adapter not provided")
	}
	if options.Codec == nil {
		panic("codec not provided")
	}
	if options.Logger == nil {
		panic("logger not provided")
	}

	return &defaultActor[T]{
		o:                    options,
		name:                 name,
		ttl:                  TTLDefault,
		preActionErrHandler:  DefaultPreActionErrorHandler[T],
		postActionErrHandler: DefaultPostActionErrorHandler[T],
	}
}

func (a *defaultActor[T]) SetTTL(ttl time.Duration) Actor[T] {
	if ttl < 0 {
		ttl = 0
	}
	a.ttl = ttl
	return a
}

func (a *defaultActor[T]) SetPreActionErrorHandler(errHandler PreActionErrorHandlerFunc[T]) Actor[T] {
	if errHandler == nil {
		panic("nil handler")
	}

	a.preActionErrHandler = errHandler
	return a
}

func (a *defaultActor[T]) SetPostActionErrorHandler(errHandler PostActionErrorHandlerFunc[T]) Actor[T] {
	if errHandler == nil {
		panic("nil handler")
	}

	a.postActionErrHandler = errHandler
	return a
}

func (a defaultActor[T]) Invalidate(ctx context.Context, keyable Keyable) error {
	key, err := a.getKey(keyable)
	if err != nil {
		return err
	}

	// No need for lock.
	return a.o.Adapter.Delete(ctx, key)
}

func (a defaultActor[T]) Do(ctx context.Context, keyable Keyable, action ActionFunc[T]) (T, error) {
	value, err := a.handle(ctx, keyable, action)
	if err != nil {
		var preErr *preActionError
		if errors.As(err, &preErr) {
			return a.handlePreActionError(ctx, keyable, action, preErr)
		}

		var postErr *postActionError[T]
		if errors.As(err, &postErr) {
			return a.handlePostActionError(ctx, keyable, action, postErr)
		}

		// Error from action.
		return zeroOf[T](), err
	}

	return value, nil
}

func (a defaultActor[T]) handlePreActionError(ctx context.Context, keyable Keyable, action ActionFunc[T], preErr *preActionError) (T, error) {
	a.o.Logger.Error(preErr)

	args := PreActionErrorHandlerArgs[T]{
		Key:         keyable,
		Action:      action,
		ErrCategory: preErr.category,
		Err:         preErr.Unwrap(),
	}
	return a.preActionErrHandler(ctx, args)
}

func (a defaultActor[T]) handlePostActionError(ctx context.Context, keyable Keyable, action ActionFunc[T], postErr *postActionError[T]) (T, error) {
	a.o.Logger.Error(postErr)

	args := PostActionErrorHandlerArgs[T]{
		Key:         keyable,
		Action:      action,
		Result:      postErr.result,
		ErrCategory: postErr.category,
		Err:         postErr.Unwrap(),
	}
	return a.postActionErrHandler(ctx, args)
}

func (a defaultActor[T]) handle(ctx context.Context, keyable Keyable, action ActionFunc[T]) (T, error) {
	key, err := a.getKey(keyable)
	if err != nil {
		return zeroOf[T](), newPreActionError("key", "error while creating key", err)
	}

	value, err := a.getValue(ctx, key)
	if err != nil {
		// If value is not found, attempt to lazy load the value into cache.
		// To speed up future requests, only attempt the lock if the value does not exist in cache.
		if errors.Is(err, adapter.ErrNotFound) {
			lockKey := "lock###" + key

			lock, err := a.o.Adapter.ObtainLock(ctx, lockKey)
			if err != nil {
				return zeroOf[T](), newPreActionError("lock", "error while attempting to lock", err)
			}
			defer func() {
				lock.Release(ctx)
				a.o.Logger.Debug("lock released", lockKey)
			}()
			a.o.Logger.Debug("lock acquired", lockKey)

			// Check for a second time.
			// This is required because one or more processes/threads might have already reached the locking stage.
			value, err := a.getValue(ctx, key)
			if err != nil {
				if errors.Is(err, adapter.ErrNotFound) {
					a.o.Logger.Debug("perform action", key)

					result, err := action(ctx)
					if err != nil {
						return zeroOf[T](), err
					}

					if err := a.storeValue(ctx, key, result); err != nil {
						return zeroOf[T](), newPostActionError("store", "error while storing value", result, err)
					}

					return result.Value, nil
				}

				return zeroOf[T](), newPreActionError("get", "error while getting value", err)
			}

			// Post-lock value get.
			return value, nil
		}

		return zeroOf[T](), newPreActionError("get", "error while getting value", err)
	}

	// Pre-lock value get.
	return value, nil
}

func (a defaultActor[T]) getKey(keyable Keyable) (string, error) {
	key, err := keyable.Key()
	if err != nil {
		return "", err
	}

	a.o.Logger.Debug("name", a.name, "key", key)

	// Prefix the key string with name.
	return a.name + "###" + key, nil
}

func (a defaultActor[T]) getValue(ctx context.Context, key string) (T, error) {
	data, err := a.o.Adapter.Get(ctx, key)
	if err != nil {
		return zeroOf[T](), err
	}

	a.o.Logger.Debug("get value", key)

	var value T
	if err := a.o.Codec.Unmarshal(data, &value); err != nil {
		return zeroOf[T](), err
	}

	return value, nil
}

func (a defaultActor[T]) storeValue(ctx context.Context, key string, result ActionResult[T]) error {
	if !result.Cache {
		a.o.Logger.Debug("not caching", key)
		return nil
	}

	ttl := result.TTL
	if ttl <= 0 {
		// If for some reason it is also zero. Don't bother caching it.
		if a.ttl < 0 {
			return nil
		}

		ttl = a.ttl
	}

	a.o.Logger.Debug("store value", key)

	data, err := a.o.Codec.Marshal(&result.Value)
	if err != nil {
		return err
	}

	if err := a.o.Adapter.Set(ctx, key, ttl, data); err != nil {
		return err
	}

	return nil
}

func DefaultPreActionErrorHandler[T any](ctx context.Context, args PreActionErrorHandlerArgs[T]) (T, error) {
	// Allow the action to execute in case of errors made when hitting cache.
	// Does not store the result in cache.
	result, err := args.Action(ctx)
	if err != nil {
		return zeroOf[T](), err
	}

	return result.Value, nil
}

func DefaultPostActionErrorHandler[T any](ctx context.Context, args PostActionErrorHandlerArgs[T]) (T, error) {
	// Ignore error and immediately return value without error.
	return args.Result.Value, nil
}

func zeroOf[T any]() (_ T) {
	return
}
