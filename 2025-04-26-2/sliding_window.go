package main


func maxSlidingWindow(nums []int, k int) []int {
	if len(nums) == 0 || k == 0 {
		return []int{}
	}
	if k == 1 {
		return nums
	}

	deque := make([]int, 0)
	result := make([]int, 0)

	for i := range nums {
		if len(deque) > 0 && deque[0] < i - k + 1 {
			deque = deque[1:]
		}

		for len(deque) > 0 && nums[deque[len(deque) - 1]] < nums[i] {
			deque = deque[:len(deque) - 1]
		}

		deque = append(deque, i)
		if i >= k - 1 {
			result = append(result, nums[deque[0]])
		}
	}
	return result
}

func main() {
	nums := []int{1, 3, -1, -3, 5, 3, 6, 7}
	k := 3
	result := maxSlidingWindow(nums, k)
	for _, v := range result {
		println(v)
	}
}