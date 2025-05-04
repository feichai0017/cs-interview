package ringbuffer

// basic implementation
type RingBuffer_basic struct {
	buffer 		[]int
	head 		int
	tail		int
	size 		int
	capacity	int
}

func NewRingBuffer_basic(capacity int) *RingBuffer_basic {
	rf := &RingBuffer_basic{
		make([]int, capacity),
		0,
		0,
		0,
		capacity,
	}
	return rf
}

func (rf *RingBuffer_basic) Read() (any, bool) {
	if rf.IsEmpty() {
		return 0, false
	}

	val := rf.buffer[rf.head]
	rf.head = (rf.head + 1) % rf.capacity
	rf.head++
	return val, true
}

func (rf *RingBuffer_basic) Write(val int) bool {
	if rf.IsFull() {
		return false
	}

	rf.buffer[rf.tail] = val
	rf.tail = (rf.tail + 1) % rf.capacity
	rf.tail++
	return true
}

func (rf *RingBuffer_basic) Peek() (int, bool) {
	if rf.IsEmpty() {
		return 0, false
	}
	return rf.buffer[rf.head], true
}


func (rf *RingBuffer_basic) IsEmpty() bool {
	return rf.size == 0
}

func (rf *RingBuffer_basic) IsFull() bool {
	return rf.size >= rf.capacity
}

func (rf *RingBuffer_basic) Len() int {
	return rf.size
}


