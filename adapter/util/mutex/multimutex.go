package mutex

import (
	"context"
	"sync"
)

type MultiMutex struct {
	mutexFactory MutexFactory
	mutexes      map[string]Mutex
	mutexCounts  map[string]int
	syncMutex    *sync.Mutex
}

func NewMultiMutex(mutexFactory MutexFactory) *MultiMutex {
	return &MultiMutex{
		mutexFactory: mutexFactory,
		mutexes:      make(map[string]Mutex),
		mutexCounts:  make(map[string]int),
		syncMutex:    &sync.Mutex{},
	}
}

func (m MultiMutex) Lock(ctx context.Context, key string) error {
	return m.getMutexForLock(key).Lock(ctx)
}

func (m MultiMutex) Unlock(ctx context.Context, key string) error {
	return m.getMutexForUnlock(key).Unlock(ctx)
}

func (m MultiMutex) getMutexForLock(key string) Mutex {
	m.syncMutex.Lock()
	defer m.syncMutex.Unlock()

	var mutex Mutex
	if m.mutexCounts[key] == 0 {
		mutex = m.mutexFactory.Make(key)
		m.mutexes[key] = mutex
	} else {
		mutex = m.mutexes[key]
	}
	m.mutexCounts[key]++

	return mutex
}

func (m MultiMutex) getMutexForUnlock(key string) Mutex {
	m.syncMutex.Lock()
	defer m.syncMutex.Unlock()

	mutex, ok := m.mutexes[key]
	if !ok {
		panic("attempting to obtain unset mutex for unlock: " + key)
	}

	m.mutexCounts[key]--
	if m.mutexCounts[key] == 0 {
		delete(m.mutexes, key)
		delete(m.mutexCounts, key)
	}

	return mutex
}
