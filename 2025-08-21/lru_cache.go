package main

import (
	"container/heap"
	"container/list"
	"sync"
	"time"
)

type entry struct {
	key   		string
	value 		any
	expireAt 	time.Time
	gen 		uint64
	node 		*list.Element
}

type ttlItem struct {
	expireAt 	time.Time
	key   		string
	gen   		uint64
}

type ttlheap []ttlItem
func (h ttlheap) Len() int { return len(h) }
func (h ttlheap) Less(i, j int) bool { return h[i].expireAt.Before(h[j].expireAt) }
func (h ttlheap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *ttlheap) Push(x any) { *h = append(*h, x.(ttlItem)) }
func (h *ttlheap) Pop() any { old := *h; *h = old[:len(old)-1]; return old[len(old)-1] }

type Cache struct {
	mu  		sync.Mutex
	cap 		int
	ll 			*list.List
	mp 			map[string]*entry
	heap 		ttlheap
	ttl 		time.Duration
	accessTTL 	bool
}

func New(capacity int, ttl time.Duration, expireAfterAccess bool) *Cache {
	c := &Cache{
		cap: 		capacity,
		ll: 		list.New(),
		mp:			make(map[string]*entry),
		ttl:		ttl,
		accessTTL: expireAfterAccess,
	}
	heap.Init(&c.heap)
	return c
}

func (c *Cache) Get(key string) (any, bool) {
    now := time.Now()
    c.mu.Lock()
    defer c.mu.Unlock()
    c.purgeExpired(now)

    e, ok := c.mp[key]
    if !ok {
        return nil, false
    }
    if !now.Before(e.expireAt) {
        c.removeEntry(e)
        return nil, false
    }
    c.ll.MoveToFront(e.node)
    if c.accessTTL {
        e.gen++
        e.expireAt = now.Add(c.ttl)
        heap.Push(&c.heap, ttlItem{expireAt: e.expireAt, key: key, gen: e.gen})
    }
    return e.value, true
}

func (c *Cache) Put(key string, val any) {
    now := time.Now()
    c.mu.Lock()
    defer c.mu.Unlock()
    c.purgeExpired(now)

    if e, ok := c.mp[key]; ok {
        e.value = val
        e.gen++
        e.expireAt = now.Add(c.ttl)
        c.ll.MoveToFront(e.node)
        heap.Push(&c.heap, ttlItem{expireAt: e.expireAt, key: key, gen: e.gen})
        return
    }
    if len(c.mp) >= c.cap {
        c.evictLRU()
    }
    e := &entry{key: key, value: val, gen: 1, expireAt: now.Add(c.ttl)}
    e.node = c.ll.PushFront(e)
    c.mp[key] = e
    heap.Push(&c.heap, ttlItem{expireAt: e.expireAt, key: key, gen: e.gen})
}

func (c *Cache) purgeExpired(now time.Time) {
    for c.heap.Len() > 0 {
        top := c.heap[0]
        if now.Before(top.expireAt) { break }
        heap.Pop(&c.heap)
        e, ok := c.mp[top.key]
        if !ok || e.gen != top.gen { continue } // 旧条目或已被删除
        c.removeEntry(e)
    }
}

func (c *Cache) evictLRU() {
    back := c.ll.Back()
    if back == nil { return }
    e := back.Value.(*entry)
    c.removeEntry(e)
}
func (c *Cache) removeEntry(e *entry) {
    delete(c.mp, e.key)
    c.ll.Remove(e.node)
}