package lru

type Node struct {
	key, val 	int
	prev, next 	*Node
}


type LRUCache struct {
	capacity 	int
	cache 		map[int]*Node
	head 		*Node // dummy head
	tail 		*Node // dummy tail
}


func NewLRUCache(capacity int) *LRUCache {
	head := &Node{}
	tail := &Node{}

	head.next = tail
	tail.prev = head


	lru := &LRUCache{
		capacity: capacity,
		cache: make(map[int]*Node),
		head: head,
		tail: tail,
	}
	return lru
}


func (lru *LRUCache) Get(key int) int {
	if node, ok := lru.cache[key]; ok {
		lru.moveToHead(node)
		return node.val
	}
	return -1
}

func (lru *LRUCache) Put(key int, val int) {
	if node, ok := lru.cache[key]; ok {
		node.val = val
		lru.moveToHead(node)
	} else {
		newNode := &Node{}
		newNode.key = key
		newNode.val = val

		lru.cache[key] = newNode
		lru.addToHead(newNode)
		if len(lru.cache) > lru.capacity {
			tail := lru.removeTail()
			delete(lru.cache, tail.key)
		}
	}
}

func (lru *LRUCache) addToHead(node *Node) {
	node.prev = lru.head
	node.next = lru.head.next
	lru.head.next.prev = node
	lru.head.next = node
}

func (lru *LRUCache) removeNode(node *Node) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (lru *LRUCache) moveToHead(node *Node) {
	lru.removeNode(node)
	lru.addToHead(node)
}

func (lru *LRUCache) removeTail() *Node {
	node := lru.tail.prev
	lru.removeNode(node)
	return node
}