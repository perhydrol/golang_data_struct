package rcumap

import (
	"math/rand"
	"sync"
	"testing"
)

// ==========================================
// 2. 对比对象: Standard RWMap 实现
// ==========================================

type RWMap[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewRWMap[K comparable, V any]() *RWMap[K, V] {
	return &RWMap[K, V]{
		data: make(map[K]V),
	}
}

func (m *RWMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

func (m *RWMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *RWMap[K, V]) Del(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// ==========================================
// 3. 单元测试: 基础逻辑验证
// ==========================================

func TestRCUMap_BasicCRUD(t *testing.T) {
	m := NewRCUMap[string, int]()

	// 1. Test Set & Get
	m.Set("a", 1)
	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Fatalf("Expected 1, got %d, found: %v", v, ok)
	}

	// 2. Test Update
	m.Set("a", 2)
	if v, ok := m.Get("a"); !ok || v != 2 {
		t.Fatalf("Expected 2, got %d", v)
	}

	// 3. Test Isolation (New key)
	m.Set("b", 10)
	if v, ok := m.Get("a"); !ok || v != 2 {
		t.Error("Setting 'b' affected 'a'")
	}
	if v, ok := m.Get("b"); !ok || v != 10 {
		t.Error("Failed to get 'b'")
	}

	// 4. Test Del
	m.Del("a")
	if _, ok := m.Get("a"); ok {
		t.Fatal("Expected 'a' to be deleted")
	}
	if v, ok := m.Get("b"); !ok || v != 10 {
		t.Error("Deleting 'a' affected 'b'")
	}
}

// ==========================================
// 4. 并发测试: 随机读写 (检测 Race)
// ==========================================

func TestRCUMap_ConcurrentRandom(t *testing.T) {
	// 运行此测试时请使用: go test -race
	m := NewRCUMap[int, int]()
	var wg sync.WaitGroup
	const numOps = 1000

	// 启动写协程
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				k := rand.Intn(100)
				if rand.Float32() < 0.5 {
					m.Set(k, j)
				} else {
					m.Del(k)
				}
			}
		}()
	}

	// 启动读协程
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				k := rand.Intn(100)
				_, _ = m.Get(k)
			}
		}()
	}

	wg.Wait()
}

// ==========================================
// 5. 性能基准测试 (Benchmarks)
// ==========================================

const (
	MapSize = 1000 // 初始 Map 大小，影响 Copy 开销
)

// 初始化辅助函数
func initMaps() (RCUMap[int, int], *RWMap[int, int], *sync.Map) {
	rcu := NewRCUMap[int, int]()
	rw := NewRWMap[int, int]()
	sm := &sync.Map{}

	for i := 0; i < MapSize; i++ {
		rcu.Set(i, i)
		rw.Set(i, i)
		sm.Store(i, i)
	}
	return rcu, rw, sm
}

// --- 场景 A: 读多写少 (99.9% 读, 0.1% 写) ---
// RCU 的绝对主场

func Benchmark_ReadHeavy_RCU(b *testing.B) {
	m, _, _ := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 模拟偶尔的写操作 (0.1%)
			if rand.Intn(1000) == 0 {
				m.Set(rand.Intn(MapSize), 1)
			} else {
				m.Get(rand.Intn(MapSize))
			}
		}
	})
}

func Benchmark_ReadHeavy_RWMutex(b *testing.B) {
	_, m, _ := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(1000) == 0 {
				m.Set(rand.Intn(MapSize), 1)
			} else {
				m.Get(rand.Intn(MapSize))
			}
		}
	})
}

func Benchmark_ReadHeavy_SyncMap(b *testing.B) {
	_, _, m := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(1000) == 0 {
				m.Store(rand.Intn(MapSize), 1)
			} else {
				m.Load(rand.Intn(MapSize))
			}
		}
	})
}

// --- 场景 B: 读写混合 (90% 读, 10% 写) ---
// RCU 开始由于 Copy 开销性能下降

func Benchmark_Mixed_RCU(b *testing.B) {
	m, _, _ := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(10) == 0 {
				m.Set(rand.Intn(MapSize), 1)
			} else {
				m.Get(rand.Intn(MapSize))
			}
		}
	})
}

func Benchmark_Mixed_RWMutex(b *testing.B) {
	_, m, _ := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(10) == 0 {
				m.Set(rand.Intn(MapSize), 1)
			} else {
				m.Get(rand.Intn(MapSize))
			}
		}
	})
}

func Benchmark_Mixed_SyncMap(b *testing.B) {
	_, _, m := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(10) == 0 {
				m.Store(rand.Intn(MapSize), 1)
			} else {
				m.Load(rand.Intn(MapSize))
			}
		}
	})
}

// --- 场景 C: 写多读少 (50% 读, 50% 写) ---
// RCU 的“灾难”现场

func Benchmark_WriteHeavy_RCU(b *testing.B) {
	m, _, _ := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(2) == 0 {
				m.Set(rand.Intn(MapSize), 1)
			} else {
				m.Get(rand.Intn(MapSize))
			}
		}
	})
}

func Benchmark_WriteHeavy_RWMutex(b *testing.B) {
	_, m, _ := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(2) == 0 {
				m.Set(rand.Intn(MapSize), 1)
			} else {
				m.Get(rand.Intn(MapSize))
			}
		}
	})
}

func Benchmark_WriteHeavy_SyncMap(b *testing.B) {
	_, _, m := initMaps()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Intn(2) == 0 {
				m.Store(rand.Intn(MapSize), 1)
			} else {
				m.Load(rand.Intn(MapSize))
			}
		}
	})
}
