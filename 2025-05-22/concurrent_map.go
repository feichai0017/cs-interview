package main

import (
	"context"
	"sync"
	"time"
)

type entry struct {
    val   int
    ready chan struct{} // 关闭后代表 value 可读
    once  sync.Once     // 保证 ready 只关一次
}

type CMap struct {
    mu   sync.RWMutex
    data map[int]*entry
}

func NewCMap() *CMap { return &CMap{data: make(map[int]*entry)} }

// Put writes the value and wakes all waiters.
func (m *CMap) Put(k, v int) {
    m.mu.Lock()
    e, ok := m.data[k]
    if !ok {
        e = &entry{ready: make(chan struct{})}
        m.data[k] = e
    }
    e.val = v
    e.once.Do(func() { close(e.ready) }) // safe, only once
    //delete(m.data, k)                    // optional: reclaim
    m.mu.Unlock()
}

// Get waits for value (or times out) and returns it.
func (m *CMap) Get(k int, d time.Duration) (int, error) {
    m.mu.RLock()
    e, ok := m.data[k]
    if !ok { // first reader: create placeholder
        m.mu.RUnlock()
        m.mu.Lock()
        e, ok = m.data[k]
        if !ok {
            e = &entry{ready: make(chan struct{})}
            m.data[k] = e
        }
        m.mu.Unlock()
    } else {
        m.mu.RUnlock()
    }

    ctx, cancel := context.WithTimeout(context.Background(), d)
    defer cancel()

    select {
    case <-e.ready:
        return e.val, nil
    case <-ctx.Done():
        return 0, ctx.Err()
    }
}
