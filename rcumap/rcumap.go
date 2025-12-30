package rcumap

import (
	"maps"
	"sync"
	"sync/atomic"
)

type RCUMap[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
	Del(K)
}

type rcuMap[K comparable, V any] struct {
	holder atomic.Pointer[map[K]V]

	mu sync.Mutex
}

func NewRCUMap[K comparable, V any]() RCUMap[K, V] {
	m := make(map[K]V)
	r := &rcuMap[K, V]{}
	r.holder.Store(&m)
	return r
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	newMap := make(map[K]V, len(m)+1)
	maps.Copy(newMap, m)
	return newMap
}

func (m *rcuMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldMap := *m.holder.Load()
	newMap := copyMap(oldMap)
	newMap[key] = value
	m.holder.Store(&newMap)
}

func (m *rcuMap[K, V]) Get(key K) (V, bool) {
	h := *m.holder.Load()
	val, ok := h[key]
	return val, ok
}

func (m *rcuMap[K, V]) Del(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldMap := *m.holder.Load()
	newMap := copyMap(oldMap)
	delete(newMap, key)
	m.holder.Store(&newMap)
}
