package wracha

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/pwnedgod/wracha/adapter"
	"github.com/pwnedgod/wracha/codec"
	"github.com/pwnedgod/wracha/logger"
)

type (
	actorDeps struct {
		adapter adapter.Adapter
		codec   codec.Codec
		logger  logger.Logger
	}

	defaultActor struct {
		d          actorDeps
		name       string
		ttl        time.Duration
		ctx        context.Context
		returnType reflect.Type
	}

	defaultManager struct {
		adapter adapter.Adapter
		codec   codec.Codec
		logger  logger.Logger
		actors  map[string]Actor
	}
)

const (
	TTLDefault = time.Duration(10) * time.Minute
)

func NewManager(adapter adapter.Adapter, codec codec.Codec, logger logger.Logger) Manager {
	return &defaultManager{
		adapter: adapter,
		codec:   codec,
		logger:  logger,
		actors:  make(map[string]Actor),
	}
}

func (m defaultManager) On(name string) Actor {
	if actor, ok := m.actors[name]; ok {
		return actor
	}
	actor := &defaultActor{
		d: actorDeps{
			adapter: m.adapter,
			codec:   m.codec,
			logger:  m.logger,
		},
		name:       name,
		ttl:        TTLDefault,
		ctx:        context.Background(),
		returnType: nil,
	}
	m.actors[name] = actor
	return actor
}

func (a *defaultActor) SetTTL(ttl time.Duration) Actor {
	if ttl < 0 {
		ttl = 0
	}
	a.ttl = ttl
	return a
}

func (a *defaultActor) SetContext(ctx context.Context) Actor {
	a.ctx = ctx
	return a
}

func (a *defaultActor) SetReturnType(returnType interface{}) Actor {
	rt := reflect.TypeOf(returnType)
	if rt.Kind() != reflect.Ptr {
		panic("not a pointer")
	}

	a.returnType = rt.Elem()
	return a
}

func (a defaultActor) Invalidate(kv interface{}) error {
	key, err := a.getKey(kv)
	if err != nil {
		return err
	}

	// No need for lock.
	return a.d.adapter.Delete(a.ctx, key)
}

func (a defaultActor) Do(kv interface{}, actionFn ActionFunc) (interface{}, error) {
	value, err := a.handle(kv, actionFn)
	if err != nil {
		var cErr *cachingError
		if errors.As(err, &cErr) {
			a.d.logger.Error(err)

			// Allow the action to execute in case of errors made when hitting cache.
			// Does not store the result in cache.
			result, err := actionFn(a.ctx)
			if err != nil {
				return nil, err
			}

			return result.Value, nil
		}

		return nil, err
	}

	return value, nil
}

func (a defaultActor) handle(kv interface{}, actionFn ActionFunc) (interface{}, error) {
	key, err := a.getKey(kv)
	if err != nil {
		return nil, &cachingError{
			message:     "error while creating key",
			previousErr: err,
		}
	}

	value, err := a.getValue(key)
	if err != nil {
		// If value is not found, attempt to lazy load the value into cache.
		// To speed up future requests, only attempt the lock if the value does not exist in cache.
		if errors.Is(err, adapter.ErrNotFound) {
			lockKey := "key###" + key

			if err := a.d.adapter.Lock(a.ctx, lockKey); err != nil {
				return nil, &cachingError{
					message:     "error while attempting to lock",
					previousErr: err,
				}
			}
			defer a.d.adapter.Unlock(a.ctx, lockKey)

			// Check for a second time.
			// This is required because one or more processes/threads might have already reached the locking stage.
			value, err := a.getValue(key)
			if err != nil {
				if errors.Is(err, adapter.ErrNotFound) {
					a.d.logger.Debug("perform action", key)

					result, err := actionFn(a.ctx)
					if err != nil {
						return nil, err
					}

					if err := a.storeValue(key, result); err != nil {
						// Ignore errors thrown while storing in cache.
						// In other words, do not return with error.
						err = &cachingError{
							message:     "error while storing value",
							previousErr: err,
						}
						a.d.logger.Error(err)
					}

					return result.Value, nil
				}

				return nil, &cachingError{
					message:     "error while obtaining value from cache",
					previousErr: err,
				}
			}

			// Post-lock value get.
			return value, nil
		}

		return nil, &cachingError{
			message:     "error while obtaining value from cache",
			previousErr: err,
		}
	}

	// Pre-lock value get.
	return value, nil
}

func (a defaultActor) getKey(v interface{}) (string, error) {
	key, err := makeKey(v)
	if err != nil {
		return "", err
	}

	a.d.logger.Debug("name", a.name, "key", key)

	// Prefix the key string with name.
	return a.name + "###" + key, nil
}

func (a defaultActor) getValue(key string) (interface{}, error) {
	data, err := a.d.adapter.Get(a.ctx, key)
	if err != nil {
		return nil, err
	}

	a.d.logger.Debug("get value", key)

	if a.returnType == nil {
		var value interface{}
		if err := a.d.codec.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		return value, nil
	}

	rv := reflect.New(a.returnType)
	if err := a.d.codec.Unmarshal(data, rv.Interface()); err != nil {
		return nil, err
	}
	return rv.Elem().Interface(), nil
}

func (a defaultActor) storeValue(key string, result ActionResult) error {
	if !result.Cache {
		a.d.logger.Debug("not caching", key)
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

	a.d.logger.Debug("store value", key)

	data, err := a.d.codec.Marshal(&result.Value)
	if err != nil {
		return err
	}

	if err := a.d.adapter.Set(a.ctx, key, ttl, data); err != nil {
		return err
	}

	return nil
}

func makeKey(key interface{}) (string, error) {
	if keyable, ok := key.(Keyable); ok {
		return keyable.Key()
	}

	// Naive way to obtain string from a value with an unknown type.
	return fmt.Sprintf("%v", key), nil
}
