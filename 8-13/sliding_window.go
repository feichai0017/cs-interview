package main

type que struct {
	queue []int
}

func NewQue() *que {
	return &que{
		queue: make([]int, 0),
	}
}

func (q *que) Back() int {
	return q.queue[len(q.queue) - 1]
}

func (q *que) Empty() bool {
	return len(q.queue) == 0
}

// 删除窗口左端“过期值”
func (q *que) Pop(val int) {
	if !q.Empty() && val == q.Front() {
		q.queue = q.queue[1:]
	}
}

// 按照单调递减入队
func (q *que) Push(val int) {
	for !q.Empty() && val > q.Back() {
		q.queue = q.queue[:len(q.queue)-1]
	}
	q.queue = append(q.queue, val)
}

func (q *que) Front() int {
	return q.queue[0]
}


func maxSlidingWindow(nums []int, k int) []int {
	queue := NewQue()
	length := len(nums)
	res := make([]int, 0)

	for i := 0; i < k; i++ {
		queue.Push(nums[i])
	}

	res = append(res, queue.Front())

	for i := k; i < length; i++ {
		queue.Pop(nums[i-k])
		queue.Push(nums[i])
		res = append(res, queue.Front())
	}

	return res

}



func main() {
	nums1 := []int{1, 3, -1, -3, 5, 3, 6, 7}

	result1 := maxSlidingWindow(nums1, 3)

	for i := range(len(result1)) {
		println(result1[i])
	}
}
