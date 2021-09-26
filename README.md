# WraCha

Easy wrapper for lazy caching results of an action for Go. Safe for multi-threaded/multi-instance use.

Supports in memory cache, [go-redis](https://github.com/go-redis/redis), and [redigo](https://github.com/gomodule/redigo).

## Usage
### Initialization

To use, create an instance of Manager with your flavor of `adapter`, `codec`, and `logger`.

```go
package main

import (
    "github.com/pwnedgod/wracha"
    "github.com/pwnedgod/wracha/adapter/memory"
    "github.com/pwnedgod/wracha/codec/json"
    "github.com/pwnedgod/wracha/logger/std"
)

func main() {
    manager := wracha.NewManager(
        memory.NewAdapter(),
        json.NewCodec(),
        std.NewLogger(),
    )

    // ...
}
```

### Defining Action

Using your manager instance. You can call `wracha.Manager.On` to get an instance of `wracha.Actor`.

The name given will be used as a prefix for cache key. Calling `wracha.Manager.On` with the same name will return the same instance of `wracha.Actor`.

```go
actor := manager.On("example")

// Set options...
actor.SetContext(ctx).
    SetTTL(time.Duration(30) * time.Minute).
    SetReturnType(new(MyStruct))
```

An action is defined as
```go
func(context.Context) (ActionResult, error)
```

The action must return a `wracha.ActionResult`.
- `Cache` determines whether to cache the given value.
- `TTL` overrides the set TTL value.
- `Value` is the value to return and possibly cache. Must be serializable.

If the action returns an error, the actor will **not** attempt to cache.

### Return Type

In order to perform a successful unmarshal when the value is obtained from cache. You must define a return type using `wracha.Actor.SetReturnType`.

```go
actor.SetReturnType(new(CustomStruct))
```

### Performing The Action

To perform an action, call `wracha.Actor.Do`.

The first parameter is the dependencies of the given action. The actor will only attempt to retrieve previous values from cache of the same dependency. If such value is found, the action will not be performed again.

`wracha.Actor.Do` will return the value from `wracha.ActionResult` and error given by the action, either directly from the action or from cache. It is always `nil` if it successfully obtained the value from cache.

```go
// Obtain dependency as parameter from either HTTP request or somewhere else...
id := "ffffffff-ffff-ffff-ffff-ffffffffffff"

v, err := actor.Do(id, func(ctx context.Context) (wracha.ActionResult, error) {
    user, err := userRepository.FindByID(ctx, id)
    if err != nil {
        return wracha.ActionResult{}, err
    }

    // Some other actions...

    return wracha.ActionResult{
        Cache: true,
        Value: user,
    }, nil
})

user := v.(model.User)

// Return value from action, write to response, etc...
```

### Invalidating Dependency
If a dependency is stale, it can be invalidated and deleted off from cache using `wracha.Actor.Invalidate`.

```go
id := "ffffffff-ffff-ffff-ffff-ffffffffffff"

err := actor.Invalidate(id)
if err != nil {
    //...
}
```

### Multiple Dependencies
If multiple dependencies are required, you can wrap your dependencies with `wracha.KeyableMap`. The map will be converted to a hashed SHA1 representation as key for the cache.

```go
deps := wracha.KeyableMap{
    "roleId": "123456abc",
    "naming": "roger*",
}

res, err := actor.Do(deps, /* ... */)
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
    //...
})

manager := wracha.NewManager(
    goredis.NewAdapter(client),
    // ...
)
```

#### redigo

```go
pool := &redis.Pool{
    //...
}

manager := wracha.NewManager(
    redigo.NewAdapter(client),
    // ...
)
```

### Codec

Codecs are used for serializing the value for storage in cache.
Currently only JSON and msgpack are provided.

Keep in mind these serializations are not perfect, especially for `time.Time`.
