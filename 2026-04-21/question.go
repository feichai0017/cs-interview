// LeetCode 25 - K 个一组翻转链表
package main

import "fmt"

type ListNode struct {
	Val  int
	Next *ListNode
}

// ===== 解法 1: 迭代 + 哑节点 (O(n) 时间, O(1) 空间, 推荐) =====
func reverseKGroup(head *ListNode, k int) *ListNode {
	dummy := &ListNode{Next: head}
	prevGroupTail := dummy

	for {
		// ① 探路: 从 prevGroupTail.Next 开始, 走 k-1 步看够不够
		groupStart := prevGroupTail.Next
		groupEnd := prevGroupTail
		for range k {
			groupEnd = groupEnd.Next
			if groupEnd == nil {
				return dummy.Next // 不足 k 个, 保持原序
			}
		}

		// ② 断开当前组, 单独反转
		nextGroupStart := groupEnd.Next
		groupEnd.Next = nil
		reversedHead := reverse(groupStart) // = groupEnd

		// ③ 拼回主链
		prevGroupTail.Next = reversedHead
		groupStart.Next = nextGroupStart

		// ④ 更新 prevGroupTail (反转后, groupStart 变成尾巴)
		prevGroupTail = groupStart
	}
}

// 反转一整个链表, 返回新头
func reverse(head *ListNode) *ListNode {
	var prev *ListNode
	curr := head
	for curr != nil {
		next := curr.Next
		curr.Next = prev
		prev = curr
		curr = next
	}
	return prev
}

// ===== 解法 2: 递归 (O(n) 时间, O(n/k) 栈空间) =====
// 思想: 反转前 k 个, 后面递归处理, 拼起来
func reverseKGroupRecursive(head *ListNode, k int) *ListNode {
	// 探路: 看够不够 k 个
	tail := head
	for range k {
		if tail == nil {
			return head // 不够, 保持原序返回
		}
		tail = tail.Next
	}
	// 此时 [head..tail前一个] 是 k 个节点, tail 是下一组的起点

	// 反转 [head..tail前一个]
	newHead := reverseRange(head, tail) // = 原本的第 k 个节点

	// 递归处理后续, 把结果接到 head (反转后变成尾) 后面
	head.Next = reverseKGroupRecursive(tail, k)

	return newHead
}

// 反转 [head, end) 这段 (左闭右开), 返回新头
func reverseRange(head, end *ListNode) *ListNode {
	var prev *ListNode = end
	curr := head
	for curr != end {
		next := curr.Next
		curr.Next = prev
		prev = curr
		curr = next
	}
	return prev
}

// ===== 测试辅助 =====
func build(vals []int) *ListNode {
	dummy := &ListNode{}
	curr := dummy
	for _, v := range vals {
		curr.Next = &ListNode{Val: v}
		curr = curr.Next
	}
	return dummy.Next
}

func dump(head *ListNode) string {
	if head == nil {
		return "nil"
	}
	s := ""
	for head != nil {
		s += fmt.Sprintf("%d", head.Val)
		if head.Next != nil {
			s += "->"
		}
		head = head.Next
	}
	return s
}

func main() {
	cases := []struct {
		vals []int
		k    int
		want string
	}{
		{[]int{1, 2, 3, 4, 5}, 2, "2->1->4->3->5"},
		{[]int{1, 2, 3, 4, 5}, 3, "3->2->1->4->5"},
		{[]int{1, 2, 3, 4, 5, 6}, 3, "3->2->1->6->5->4"},
		{[]int{1, 2, 3, 4, 5}, 1, "1->2->3->4->5"}, // k=1 等于不变
		{[]int{1, 2, 3, 4, 5}, 5, "5->4->3->2->1"}, // 一组反转整个
		{[]int{1}, 1, "1"},
		{[]int{1, 2}, 3, "1->2"}, // 不足 k
	}

	for _, c := range cases {
		// 测试两个版本
		head1 := build(c.vals)
		got1 := dump(reverseKGroup(head1, c.k))

		head2 := build(c.vals)
		got2 := dump(reverseKGroupRecursive(head2, c.k))

		ok := "OK"
		if got1 != c.want || got2 != c.want {
			ok = "FAIL"
		}
		fmt.Printf("[%s] vals=%v k=%d  iter=%-25s  recur=%-25s  want=%s\n",
			ok, c.vals, c.k, got1, got2, c.want)
	}
}
