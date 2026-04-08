package medianfinder

import "container/heap"

// 小顶堆：存较大的一半
type MinHeap []int

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h MinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *MinHeap) Push(x any) {
	*h = append(*h, x.(int))
}

func (h *MinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// 大顶堆：存较小的一半
type MaxHeap []int

func (h MaxHeap) Len() int { return len(h) }
func (h MaxHeap) Less(i, j int) bool { return h[i] > h[j] } // 这里反过来，就是大顶堆
func (h MaxHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *MaxHeap) Push(x any) {
	*h = append(*h, x.(int))
}

func (h *MaxHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type MedianFinder struct {
	left  *MaxHeap // 较小的一半
	right *MinHeap // 较大的一半
}

func Constructor() MedianFinder {
	left := &MaxHeap{}
	right := &MinHeap{}
	heap.Init(left)
	heap.Init(right)
	return MedianFinder{
		left:  left,
		right: right,
	}
}

func (mf *MedianFinder) AddNum(num int) {
	// 1. 先放到合适的一边
	if mf.left.Len() == 0 || num <= (*mf.left)[0] {
		heap.Push(mf.left, num)
	} else {
		heap.Push(mf.right, num)
	}

	// 2. 平衡大小
	if mf.left.Len() > mf.right.Len()+1 {
		heap.Push(mf.right, heap.Pop(mf.left))
	} else if mf.right.Len() > mf.left.Len()+1 {
		heap.Push(mf.left, heap.Pop(mf.right))
	}
}

func (mf *MedianFinder) FindMedian() float64 {
	if mf.left.Len() > mf.right.Len() {
		return float64((*mf.left)[0])
	}
	if mf.right.Len() > mf.left.Len() {
		return float64((*mf.right)[0])
	}
	return float64((*mf.left)[0]+(*mf.right)[0]) / 2.0
}