// LeetCode 567 - 字符串的排列
package main

import "fmt"

// 朴素版: O(n * 26), 每次窗口滑动做一次数组相等比较
func checkInclusion(s1, s2 string) bool {
	m, n := len(s1), len(s2)
	if m > n {
		return false
	}
	var need, have [26]int
	for i := 0; i < m; i++ {
		need[s1[i]-'a']++
		have[s2[i]-'a']++
	}
	if need == have {
		return true
	}
	for i := m; i < n; i++ {
		have[s2[i]-'a']++
		have[s2[i-m]-'a']--
		if need == have {
			return true
		}
	}
	return false
}

// 优化版: 维护 matches 计数器, 严格 O(n)
func checkInclusionOpt(s1, s2 string) bool {
	m, n := len(s1), len(s2)
	if m > n {
		return false
	}
	var need, have [26]int
	for i := 0; i < m; i++ {
		need[s1[i]-'a']++
	}

	matches := 0
	// 统计初始 matches: 遍历 26 个字符看哪些 need==have(都是0的也算)
	// 为简洁, 先不算 0=0 的匹配, 而是在窗口构建时动态更新
	// -> 改用: 所有 need[c]==0 的字符初始就算 matches, 然后构建窗口时动态调整
	for c := 0; c < 26; c++ {
		if need[c] == 0 {
			matches++
		}
	}

	update := func(idx byte, delta int) {
		c := idx - 'a'
		before := have[c] == need[c]
		have[c] += delta
		after := have[c] == need[c]
		if before && !after {
			matches--
		} else if !before && after {
			matches++
		}
	}

	// 初始窗口
	for i := 0; i < m; i++ {
		update(s2[i], +1)
	}
	if matches == 26 {
		return true
	}

	// 滑动
	for i := m; i < n; i++ {
		update(s2[i], +1)
		update(s2[i-m], -1)
		if matches == 26 {
			return true
		}
	}
	return false
}

func main() {
	cases := []struct {
		s1, s2 string
		want   bool
	}{
		{"ab", "eidbaooo", true},
		{"ab", "eidboaoo", false},
		{"adc", "dcda", true},
		{"abc", "abc", true},
		{"abc", "ab", false},
		{"a", "a", true},
		{"hello", "ooolleoooleh", false},
	}
	for _, c := range cases {
		got1 := checkInclusion(c.s1, c.s2)
		got2 := checkInclusionOpt(c.s1, c.s2)
		ok := "OK"
		if got1 != c.want || got2 != c.want {
			ok = "FAIL"
		}
		fmt.Printf("[%s] s1=%q s2=%q  naive=%v opt=%v  want=%v\n",
			ok, c.s1, c.s2, got1, got2, c.want)
	}
}
