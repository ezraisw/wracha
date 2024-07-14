# WraCha

Easy wrapper for lazy caching results of an action for Go. Safe for multi-threaded/multi-instance use.

Supports in memory cache, [go-redis](https://github.com/go-redis/redis), and [redigo](https://github.com/gomodule/redigo).

## Installation

Simply run the following command to install:

```
go get github.com/ezraisw/wracha
```

## Usage

### Initialization

To use, prepare a `wracha.ActorOptions` with your flavor of `adapter`, `codec`, and `logger`.

```go
package main

import (
    "github.com/ezraisw/wracha"
    "github.com/ezraisw/wracha/adapter/memory"
    "github.com/ezraisw/wracha/codec/json"
    "github.com/ezraisw/wracha/logger/std"
)

func main() {
    opts := wracha.ActorOptions(
        memory.NewAdapter(),
        json.NewCodec(),
        std.NewLogger(),
    )

    // ...
}
```

### Defining Action

Create an `wracha.Actor` through `wracha.NewActor` with the options.

The name given will be used as a prefix for cache key.

```go
actor := wracha.NewActor[MyStruct]("example", opts)

// Set options...
actor.SetTTL(time.Duration(30) * time.Minute).
    SetPreActionErrorHandler(preActionErrorHandler).
    SetPostActionErrorHandler(postActionErrorHandler)
```

An action is defined as

```go
func[T any](context.Context) (ActionResult[T], error)
```

The action must return a `wracha.ActionResult[T any]`.

- `Cache` determines whether to cache the given value.
- `TTL` overrides the set TTL value.
- `Value` is the value to return and possibly cache. Must be serializable.

If the action returns an error, the actor will **not** attempt to cache.

### Performing The Action

To perform an action, call `wracha.Actor[T any].Do`.

The first parameter is the key of the given action. The actor will only attempt to retrieve previous values from cache of the same key. If such value is found in cache, the action will not be performed again.

`wracha.Actor[T any].Do` will return the value from `wracha.ActionResult[T any]` and error. The returned error can either be from the action or from the caching process.

```go
// Obtain key from either HTTP request or somewhere else...
id := "ffffffff-ffff-ffff-ffff-ffffffffffff"

user, err := actor.Do(ctx, wracha.KeyableStr(id), func[model.User](ctx context.Context) (wracha.ActionResult[model.User], error) {
    user, err := userRepository.FindByID(ctx, id)
    if err != nil {
        return wracha.ActionResult[model.User]{}, err
    }

    // Some other actions...

    return wracha.ActionResult[model.User]{
        Cache: true,
        Value: user,
    }, nil
})

// Return value from action, write to response, etc...
```

### Invalidating Dependency/Cache Entry

If a dependency/cache entry is stale, it can be invalidated and deleted off from cache using `wracha.Actor[T any].Invalidate`.

```go
id := "ffffffff-ffff-ffff-ffff-ffffffffffff"

err := actor.Invalidate(ctx, wracha.KeyableStr(id))
if err != nil {
    // ...
}
```

### Error Handling

By default, errors thrown before calling the action (value retrieval or locking) immediately executes the action without an attempt to store the value in cache. All errors thrown after calling the action (value storage) is also ignored.

You can override this behaviour by setting either `wracha.Actor[T any].SetPreActionErrorHandler` or `wracha.Actor[T any].SetPostActionErrorHandler`.

### Multiple Dependencies

If multiple dependencies are required, you can wrap your dependencies with `wracha.KeyableMap`. The map will be converted to a hashed SHA1 representation as key for the cache.

```go
deps := wracha.KeyableMap{
    "roleId": "123456abc",
    "naming": "roger*",
}

res, err := actor.Do(ctx, deps, /* ... */)
```

### Adapters

Adapters are used for storing cache data. Out of the box, three adapters are provided:

- memory (uses [ccache](https://github.com/karlseguin/ccache))
- goredis
- redigo

You can create your own adapter by satisfying the following interface:

```go
type Adapter interface {
	Exists(ctx context.Context, key string) (bool, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, ttl time.Duration, data []byte) error
	Delete(ctx context.Context, key string) error
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
}

```

#### go-redis

```go
client := redis.NewClient(&redis.Options{
    // ...
})

opts := wracha.ActorOptions{
    goredis.NewAdapter(client),
    // ...
}
```

#### redigo

```go
pool := &redis.Pool{
    // ...
}

opts := wracha.ActorOptions{
    redigo.NewAdapter(pool),
    // ...
}
```

### Codec

Codecs are used for serializing the value for storage in cache.
Currently only JSON and msgpack are provided.

Keep in mind these serializations are not perfect, especially for `time.Time`.
