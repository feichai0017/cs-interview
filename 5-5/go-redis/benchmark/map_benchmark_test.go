package benchmark

import (
    "fmt"
    "math/rand"
    "strconv"
    "sync"
    "testing"

    cmap "github.com/orcaman/concurrent-map/v2"
    hashmap "github.com/cornelk/hashmap"
)

const (
    K = 10000 // number of distinct keys
)

var keys []string

func init() {
    keys = make([]string, K)
    for i := range K {
        keys[i] = "key" + strconv.Itoa(i)
    }
}

func randKey(r *rand.Rand) string {
    return keys[r.Intn(len(keys))]
}

// runConcurrent helper: total ops = b.N, per-goroutine ops = b.N/conc
func runConcurrent(b *testing.B, conc int, op func(key string, idx int)) {
    ops := b.N / conc
    if ops == 0 {
        ops = 1
    }
    var wg sync.WaitGroup
    wg.Add(conc)
    for g := 0; g < conc; g++ {
        go func(gid int) {
            defer wg.Done()
            start := gid * ops
            for i := range ops {
                op(keys[(start+i)%len(keys)], start+i)
            }
        }(g)
    }
    wg.Wait()
}

func BenchmarkMapsConcurrent(b *testing.B) {
    concs := []int{1, 2, 4, 8, 16, 32}

    // SyncMap benchmarks
    for _, conc := range concs {
        b.Run(fmt.Sprintf("SyncMap/Set/P%d", conc), func(b *testing.B) {
            var m sync.Map
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, v int) { m.Store(k, v) })
        })
        b.Run(fmt.Sprintf("SyncMap/Get/P%d", conc), func(b *testing.B) {
            var m sync.Map
            for _, k := range keys {
                m.Store(k, k)
            }
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, _ int) { m.Load(k) })
        })
        b.Run(fmt.Sprintf("SyncMap/Mixed/P%d", conc), func(b *testing.B) {
            var m sync.Map
            for i, k := range keys {
                if i%2 == 0 {
                    m.Store(k, i)
                }
            }
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, i int) {
                if i&1 == 0 {
                    m.Load(k)
                } else {
                    m.Store(k, i)
                }
            })
        })
    }

    // ConcurrentMap benchmarks
    for _, conc := range concs {
        b.Run(fmt.Sprintf("ConcurrentMap/Set/P%d", conc), func(b *testing.B) {
            m := cmap.New[int]()
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, v int) { m.Set(k, v) })
        })
        b.Run(fmt.Sprintf("ConcurrentMap/Get/P%d", conc), func(b *testing.B) {
            m := cmap.New[string]()
            for _, k := range keys {
                m.Set(k, k)
            }
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, _ int) { m.Get(k) })
        })
        b.Run(fmt.Sprintf("ConcurrentMap/Mixed/P%d", conc), func(b *testing.B) {
            m := cmap.New[int]()
            for i, k := range keys {
                if i%2 == 0 {
                    m.Set(k, i)
                }
            }
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, i int) {
                if i&1 == 0 {
                    m.Get(k)
                } else {
                    m.Set(k, i)
                }
            })
        })
    }

    // HashMap benchmarks
    for _, conc := range concs {
        b.Run(fmt.Sprintf("HashMap/Set/P%d", conc), func(b *testing.B) {
            m := hashmap.New[string, int]()
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, v int) { m.Set(k, v) })
        })
        b.Run(fmt.Sprintf("HashMap/Get/P%d", conc), func(b *testing.B) {
            m := hashmap.New[string, string]()
            for _, k := range keys {
                m.Set(k, k)
            }
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, _ int) { m.Get(k) })
        })
        b.Run(fmt.Sprintf("HashMap/Mixed/P%d", conc), func(b *testing.B) {
            m := hashmap.New[string, int]()
            for i, k := range keys {
                if i%2 == 0 {
                    m.Set(k, i)
                }
            }
            b.ResetTimer()
            runConcurrent(b, conc, func(k string, i int) {
                if i&1 == 0 {
                    m.Get(k)
                } else {
                    m.Set(k, i)
                }
            })
        })
    }
}
