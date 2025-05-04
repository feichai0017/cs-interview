package main

func subarraySum(nums []int, k int) int {
	currSum := 0
	countMap := make(map[int]int)
	result := 0

	countMap[0] = 1

	for _, val := range nums {
		currSum += val
		rest := currSum - k
		if count, ok := countMap[rest]; ok {
			result += count
		}
		countMap[currSum]++
	}
	return result
}