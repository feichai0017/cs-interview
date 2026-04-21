// 判定字符串是否互为字符重排
package main

import (
	"fmt"
	"sort"
)

// 排序法: O(n log n)
func isAnagramSort(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	b1, b2 := []byte(s1), []byte(s2)
	sort.Slice(b1, func(i, j int) bool { return b1[i] < b1[j] })
	sort.Slice(b2, func(i, j int) bool { return b2[i] < b2[j] })
	return string(b1) == string(b2)
}

// 计数数组法: O(n), O(1)
// 按 byte 统计。仅对 ASCII 正确。
// UTF-8 下会有假阳性: 例如 "一你" vs "丠佀" 字节多重集相同但是不同的字。
// 详见 counterexample/main.go
func isAnagramCount(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	var cnt [256]int
	for i := 0; i < len(s1); i++ {
		cnt[s1[i]]++
		cnt[s2[i]]--
	}
	for _, v := range cnt {
		if v != 0 {
			return false
		}
	}
	return true
}

// map 计数法 (支持 Unicode): O(n), O(k)
func isAnagramMap(s1, s2 string) bool {
	r1, r2 := []rune(s1), []rune(s2)
	if len(r1) != len(r2) {
		return false
	}
	m := make(map[rune]int, len(r1))
	for _, r := range r1 {
		m[r]++
	}
	for _, r := range r2 {
		m[r]--
		if m[r] < 0 {
			return false
		}
	}
	return true
}

func main() {
	cases := []struct {
		s1, s2 string
		want   bool
	}{
		{"abc", "bca", true},
		{"abc", "bad", false},
		{"abc", "abcd", false},
		{"", "", true},
		{"aabbcc", "abcabc", true},
		{"Abc", "abc", false}, // 大小写敏感
		{"你好世界", "界世好你", true},
		{"你好", "abc", false},
		// 反例: count (按 byte) 会错误地返回 true, map (按 rune) 正确返回 false
		{"一你", "丠佀", false},
	}

	for _, c := range cases {
		fmt.Printf("sort(%q,%q)=%v count=%v map=%v  want=%v\n",
			c.s1, c.s2,
			isAnagramSort(c.s1, c.s2),
			isAnagramCount(c.s1, c.s2),
			isAnagramMap(c.s1, c.s2),
			c.want,
		)
	}
}
