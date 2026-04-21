// Go slice 底层行为实测
package main

import "fmt"

func main() {
	demoGrowth()
	demoAppendShare()
	demoFullSliceExpr()
	demoNilVsEmpty()
	demoPassByValue()
}

// ---------- Demo 1: 实测扩容曲线 ----------
func demoGrowth() {
	fmt.Println("=== Demo 1: append 扩容曲线 ===")
	var s []int
	prevCap := cap(s)
	for i := 0; i < 2000; i++ {
		s = append(s, i)
		if cap(s) != prevCap {
			fmt.Printf("  len=%4d  cap=%4d  (grew from %d)\n", len(s), cap(s), prevCap)
			prevCap = cap(s)
		}
	}
	fmt.Println()
}

// ---------- Demo 2: append 共享底层数组的坑 ----------
func demoAppendShare() {
	fmt.Println("=== Demo 2: append 共享底层 ===")
	s := []int{1, 2, 3, 4, 5}
	a := s[:3]               // len=3, cap=5
	fmt.Printf("  原始 s = %v\n", s)
	fmt.Printf("  a = s[:3] = %v  (len=%d cap=%d)\n", a, len(a), cap(a))

	b := append(a, 99)       // cap 够, 原地写, 覆盖 s[3]
	fmt.Printf("  append(a, 99) 后:\n")
	fmt.Printf("    b = %v\n", b)
	fmt.Printf("    s = %v   <- 被意外修改了!\n", s)
	fmt.Println()
}

// ---------- Demo 3: full slice expression 修复 ----------
func demoFullSliceExpr() {
	fmt.Println("=== Demo 3: s[low:high:max] 修复共享问题 ===")
	s := []int{1, 2, 3, 4, 5}
	a := s[:3:3]             // cap 被限制为 3
	fmt.Printf("  s = %v\n", s)
	fmt.Printf("  a = s[:3:3] = %v  (len=%d cap=%d)\n", a, len(a), cap(a))

	b := append(a, 99)       // cap 不够, 必须扩容到新数组
	fmt.Printf("  append(a, 99) 后:\n")
	fmt.Printf("    b = %v  (cap=%d)\n", b, cap(b))
	fmt.Printf("    s = %v  <- 保持不变\n", s)
	fmt.Println()
}

// ---------- Demo 4: nil slice vs empty slice ----------
func demoNilVsEmpty() {
	fmt.Println("=== Demo 4: nil vs empty slice ===")
	var d []int
	e := []int{}

	fmt.Printf("  d (var):   len=%d cap=%d  d==nil: %v\n", len(d), cap(d), d == nil)
	fmt.Printf("  e ([]{}):  len=%d cap=%d  e==nil: %v\n", len(e), cap(e), e == nil)

	// 都可以 append
	d = append(d, 1)
	e = append(e, 1)
	fmt.Printf("  append 1 后: d=%v e=%v\n", d, e)
	fmt.Println()
}

// ---------- Demo 5: slice 传参的「半共享」行为 ----------
func demoPassByValue() {
	fmt.Println("=== Demo 5: slice 传参 ===")

	modify := func(s []int) {
		s[0] = 999              // 改底层, 对外可见
		s = append(s, 100)      // 局部扩容, 对外不可见
	}

	s := make([]int, 3, 10)    // cap=10, 这样 append 不会重新分配
	s[0], s[1], s[2] = 1, 2, 3
	fmt.Printf("  传参前: s = %v (len=%d cap=%d)\n", s, len(s), cap(s))
	modify(s)
	fmt.Printf("  传参后: s = %v (len=%d cap=%d)\n", s, len(s), cap(s))
	fmt.Printf("  注意: s[0] 改了 (共享底层), 但 len 没变 (header 是拷贝)\n")
}
