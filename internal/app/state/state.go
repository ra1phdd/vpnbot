package state

import (
	"sync"
)

type Storage[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T)
	Update(key string, updater func(T) T)
	Delete(key string)
}

type memoryState[T any] struct {
	mu    sync.RWMutex
	store map[string]T
}

func NewMemoryStorage[T any]() Storage[T] {
	ms := &memoryState[T]{
		store: make(map[string]T),
	}

	return ms
}

func (ms *memoryState[T]) Get(key string) (T, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	item, exists := ms.store[key]
	if !exists {
		var zero T
		return zero, false
	}
	return item, true
}

func (ms *memoryState[T]) Set(key string, value T) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.store[key] = value
}

func (ms *memoryState[T]) Update(key string, updater func(T) T) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	oldValue, exists := ms.store[key]
	if !exists {
		return
	}

	newValue := updater(oldValue)
	ms.store[key] = newValue
}

func (ms *memoryState[T]) Delete(key string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.store, key)
}
