package rcumap

import (
	"hash/maphash"
	"maps"
	"sync"
	"sync/atomic"
)

type RCUMap[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
	Del(K)
	Size() int
}

type rcuMap[K comparable, V any] struct {
	// 使用指针可以有效避免slice扩容时进行大量拷贝，并且sync.Mutex本身也不允许拷贝操作
	shards []*rcuShard[K, V]
	len    int32
	cap    int
	h      maphash.Seed
}

type rcuShard[K comparable, V any] struct {
	holder atomic.Pointer[map[K]V]
	mu     sync.Mutex
}

func NewRCUMap[K comparable, V any]() RCUMap[K, V] {
	cap := 10
	shards := make([]*rcuShard[K, V], 0, 10)
	for range cap {
		m := map[K]V{}
		r := rcuShard[K, V]{}
		r.holder.Store(&m)
		shards = append(shards, &r)
	}
	r := rcuMap[K, V]{shards: shards, len: 0, cap: cap, h: maphash.MakeSeed()}
	return &r
}

func (m *rcuMap[K, V]) hash(key K) uint {
	return uint(maphash.Comparable(m.h, key))
}

func (m *rcuMap[K, V]) Size() int {
	return int(atomic.LoadInt32(&m.len))
}

func (m *rcuMap[K, V]) Set(key K, value V) {
	shardIndex := m.hash(key) % uint(m.cap)
	shard := m.shards[shardIndex]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	oldMap := *shard.holder.Load()
	newMap := make(map[K]V, len(oldMap))
	maps.Copy(newMap, oldMap)
	newMap[key] = value
	shard.holder.Store(&newMap)
	atomic.AddInt32(&m.len, 1)
}

func (m *rcuMap[K, V]) Get(key K) (V, bool) {
	shardIndex := m.hash(key) % uint(m.cap)
	shard := m.shards[shardIndex]
	holder := *shard.holder.Load()
	v, ok := holder[key]
	return v, ok
}

func (m *rcuMap[K, V]) Del(key K) {
	shardIndex := m.hash(key) % uint(m.cap)
	shard := m.shards[shardIndex]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	oldMap := *shard.holder.Load()
	newMap := make(map[K]V, len(oldMap))
	maps.Copy(newMap, oldMap)
	delete(newMap, key)
	shard.holder.Store(&newMap)
	atomic.AddInt32(&m.len, -1)
}
