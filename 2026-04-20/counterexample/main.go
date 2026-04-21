// 验证 [256]int 按 byte 计数在 UTF-8 上会不会误判
package main

import "fmt"

func isAnagramByByte(s1, s2 string) bool {
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

func isAnagramByRune(s1, s2 string) bool {
	r1, r2 := []rune(s1), []rune(s2)
	if len(r1) != len(r2) {
		return false
	}
	m := make(map[rune]int)
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

func dumpBytes(s string) {
	fmt.Printf("  %q bytes: ", s)
	for i := 0; i < len(s); i++ {
		fmt.Printf("%02X ", s[i])
	}
	fmt.Printf("  runes: ")
	for _, r := range s {
		fmt.Printf("U+%04X ", r)
	}
	fmt.Println()
}

func main() {
	// 构造: 同样 6 个字节, 但 rune 组合完全不同
	// 一 U+4E00 = E4 B8 80
	// 你 U+4F60 = E4 BD A0
	// 丠 U+4E20 = E4 B8 A0   <- 交换低字节
	// 佀 U+4F40 = E4 BD 80   <- 交换低字节
	s1 := "一你"
	s2 := "丠佀"

	dumpBytes(s1)
	dumpBytes(s2)

	fmt.Printf("\nisAnagramByByte(s1,s2) = %v\n", isAnagramByByte(s1, s2))
	fmt.Printf("isAnagramByRune(s1,s2) = %v   <- 真相\n", isAnagramByRune(s1, s2))
}
