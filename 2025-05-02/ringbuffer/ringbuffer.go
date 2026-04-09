package ringbuffer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// lock-free and genric type
type RingBuffer[T any] struct {
	buffer 		[]T
	head 		atomic.Uint32	
	tail		atomic.Uint32
	capacity	uint32
	mu 			sync.Mutex
	notEmpty	*sync.Cond
	notFull		*sync.Cond
}

func NewRingBuffer[T any](capacity uint32) *RingBuffer[T] {
	rf := &RingBuffer[T]{
		buffer: make([]T, capacity),
		capacity: capacity,
	}
	rf.notEmpty = sync.NewCond(&rf.mu)
	rf.notFull = sync.NewCond(&rf.mu)
	return rf
}

func (rf *RingBuffer[T]) Read() (T, bool) {
	var zero T

	for {
		head := rf.head.Load()
		tail := rf.tail.Load()

		if head == tail {
			return zero, false
		}

		pos := head % rf.capacity
		val := rf.buffer[pos]

		if rf.head.CompareAndSwap(head, head + 1) {
			return val, true
		}
	}
}

func (rf *RingBuffer[T]) ReadBlocking() T {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	for rf.IsEmpty() {
		rf.notEmpty.Wait()
	}

	pos := rf.head.Load() % rf.capacity
	val := rf.buffer[pos]

	rf.head.Add(1)

	rf.notFull.Signal()

	return val
}

func (rf *RingBuffer[T]) Write(val T) bool {
	for {
		tail := rf.tail.Load()
		head := rf.head.Load()

		if tail - head >= rf.capacity {
			return false // full
		}

		pos := tail % rf.capacity
		rf.buffer[pos] = val

		if rf.tail.CompareAndSwap(tail, tail+1) {
			return true
		}
	}
}

func (rf *RingBuffer[T]) WriteBlocking(val T) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	for rf.IsFull() {
		rf.notFull.Wait()
	}

	pos := rf.tail.Load() % rf.capacity
	rf.buffer[pos] = val

	rf.tail.Add(1)

	rf.notEmpty.Signal()
}

func (rf *RingBuffer[T]) Peek() (T, bool) {
	var zero T
	head := rf.head.Load()
	tail := rf.tail.Load()

	if head == tail {
		return zero, false
	}

	pos := head % rf.capacity
	return rf.buffer[pos], true
}


func (rf *RingBuffer[T]) IsEmpty() bool {
	head := rf.head.Load()
	tail := rf.tail.Load()
	return head == tail
}

func (rf *RingBuffer[T]) IsFull() bool {
	head := rf.head.Load()
	tail := rf.tail.Load()
	return tail - head >= rf.capacity
}

func (rf *RingBuffer[T]) Len() uint32 {
	head := rf.head.Load()
	tail := rf.tail.Load()
	return tail - head
}



func main() {
	rb := NewRingBuffer[int](3)

	fmt.Println("== Write ==")
	fmt.Println(rb.Write(1)) // true
	fmt.Println(rb.Write(2)) // true
	fmt.Println(rb.Write(3)) // true
	fmt.Println(rb.Write(4)) // false (full)

	fmt.Println("== IsFull ==")
	fmt.Println(rb.IsFull()) // true

	fmt.Println("== Read ==")
	fmt.Println(rb.Read()) // 1 true
	fmt.Println(rb.Read()) // 2 true

	fmt.Println("== Write After Read ==")
	fmt.Println(rb.Write(4)) // true
	fmt.Println(rb.Write(5)) // true
	fmt.Println(rb.Write(6)) // false

	fmt.Println("== Final Reads ==")
	for !rb.IsEmpty() {
		val, _ := rb.Read()
		fmt.Println(val)
	}

	const (
		bufferSize   = 5
		writerCount  = 3
		readerCount  = 2
		elementsEach = 10
	)

	rf := NewRingBuffer[int](bufferSize)
	var wg sync.WaitGroup

	// 多个写 goroutine
	for i := range writerCount {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range elementsEach {
				val := id*100 + j
				rf.WriteBlocking(val)
				fmt.Printf("Writer %d wrote %d\n", id, val)
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	// 多个读 goroutine
	for i := range readerCount {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for range writerCount*elementsEach/readerCount {
				val := rf.ReadBlocking()
				fmt.Printf("Reader %d read %d\n", id, val)
				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	// 等待所有写入/读取完成
	wg.Wait()
	fmt.Println("All goroutines finished.")
}
