package main

// MaxWindow：固定窗口大小 k 的在线最大值队列
type MaxWindow struct {
	k   int
	idx int // 已处理元素个数，用作当前元素的下标
	dq  []pair
}
type pair struct{ i, v int } // 既存值也存下标

func NewMaxWindow(k int) *MaxWindow { return &MaxWindow{k: k, dq: make([]pair, 0, k)} }

func (w *MaxWindow) Push(x int) {
	i := w.idx
	w.idx++

	// 1) 先清理过期元素：队首下标 <= i - k
	for len(w.dq) > 0 && w.dq[0].i <= i-w.k {
		// 避免底层数组长时间持有引用，帮助 GC
		w.dq[0] = pair{}
		w.dq = w.dq[1:]
	}

	// 2) 维护单调递减：把尾部比 x 小的都弹掉
	for len(w.dq) > 0 && w.dq[len(w.dq)-1].v < x {
		w.dq[len(w.dq)-1] = pair{}
		w.dq = w.dq[:len(w.dq)-1]
	}

	// 3) 压入新元素
	w.dq = append(w.dq, pair{i: i, v: x})
}

// 当前窗口是否已满（即是否能读到合法的最大值）
func (w *MaxWindow) Ready() bool { return w.idx >= w.k }

// 返回当前窗口最大值；未满返回 false
func (w *MaxWindow) Max() (int, bool) {
	if !w.Ready() || len(w.dq) == 0 {
		return 0, false
	}
	return w.dq[0].v, true
}

