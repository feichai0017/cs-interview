// Go 逃逸分析 demo
//
// 验证方式: go build -gcflags="-m" escape_demo.go
//          能看到每个变量的逃逸决定
//
// 也可以 go run escape_demo.go 看运行结果 (但逃逸信息只在编译期能看)
package main

import (
	"fmt"
	"runtime"
)

type ListNode struct {
	Val  int
	Next *ListNode
}

// 案例 1: 返回局部变量指针 → 逃逸
func newNode(val int) *ListNode {
	n := ListNode{Val: val} // 编译器会报: moved to heap: n
	return &n
}

// 案例 2: 不返回指针 → 不逃逸 (栈分配)
func sumOnStack() int {
	n := ListNode{Val: 42} // 不逃逸, 栈上
	return n.Val
}

// 案例 3: 大对象直接逃逸
func bigArray() {
	arr := [1 << 16]int{} // 256KB 数组
	_ = arr
}

// 案例 4: 闭包捕获 → 逃逸
func makeCounter() func() int {
	count := 0 // 被闭包持有, 逃逸到堆
	return func() int {
		count++
		return count
	}
}

// 案例 5: interface 装箱 → 逃逸
func printAny(x interface{}) {
	fmt.Println(x)
}

// 案例 6: 链表构造 → 全部节点逃逸
func buildList(n int) *ListNode {
	dummy := &ListNode{}
	curr := dummy
	for i := 0; i < n; i++ {
		curr.Next = &ListNode{Val: i} // 每个节点都逃逸
		curr = curr.Next
	}
	return dummy.Next
}

// 演示: 大链表对 GC 的影响
func gcStressTest() {
	var stats runtime.MemStats

	runtime.ReadMemStats(&stats)
	fmt.Printf("起始: HeapAlloc=%d KB, NumGC=%d\n",
		stats.HeapAlloc/1024, stats.NumGC)

	// 创建 100 万节点链表
	head := buildList(1_000_000)

	runtime.ReadMemStats(&stats)
	fmt.Printf("100万节点后: HeapAlloc=%d KB, HeapObjects=%d\n",
		stats.HeapAlloc/1024, stats.HeapObjects)

	// 触发 GC
	runtime.GC()
	runtime.ReadMemStats(&stats)
	fmt.Printf("GC 后: HeapAlloc=%d KB, NumGC=%d, GCPauseTotal=%d µs\n",
		stats.HeapAlloc/1024, stats.NumGC, stats.PauseTotalNs/1000)

	// 释放引用
	head = nil
	_ = head
	runtime.GC()
	runtime.ReadMemStats(&stats)
	fmt.Printf("释放后 GC: HeapAlloc=%d KB, HeapObjects=%d\n",
		stats.HeapAlloc/1024, stats.HeapObjects)
}

func main() {
	// 调用一下避免优化掉
	_ = newNode(1)
	_ = sumOnStack()
	bigArray()
	c := makeCounter()
	c()
	printAny(42)

	fmt.Println("=== 大链表 GC 压力测试 ===")
	gcStressTest()

	fmt.Println("\n=== 看逃逸分析请运行 ===")
	fmt.Println("go build -gcflags=\"-m\" escape_demo.go")
}
