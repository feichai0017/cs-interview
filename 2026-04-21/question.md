## K 个一组翻转链表（LeetCode 25）

**问题描述**

给你链表的头节点 `head`，每 `k` 个节点一组进行翻转，请你返回修改后的链表。

`k` 是一个正整数，它的值小于或等于链表的长度。如果节点总数不是 `k` 的整数倍，那么请将**最后剩余的节点保持原有顺序**。

你**不能只是单纯的改变节点内部的值**，而是需要**实际进行节点交换**。


**示例 1**

```
输入: head = [1,2,3,4,5], k = 2
输出: [2,1,4,3,5]

原链表:  1 -> 2 -> 3 -> 4 -> 5
分组:   [1,2] [3,4] [5]
翻转:   [2,1] [4,3] [5]   (最后一组不足 k 个, 不翻转)
结果:    2 -> 1 -> 4 -> 3 -> 5
```

**示例 2**

```
输入: head = [1,2,3,4,5], k = 3
输出: [3,2,1,4,5]

原链表:  1 -> 2 -> 3 -> 4 -> 5
分组:   [1,2,3] [4,5]
翻转:   [3,2,1] [4,5]    (最后一组不足 3 个)
结果:    3 -> 2 -> 1 -> 4 -> 5
```

**示例 3**

```
输入: head = [1,2,3,4,5,6], k = 3
输出: [3,2,1,6,5,4]
```


**约束**

- 链表中的节点数目为 `n`，`1 <= k <= n <= 5000`
- `0 <= Node.val <= 1000`

**进阶**

- 你可以设计一个**只用 O(1) 额外空间**的算法吗？（迭代法）


**链表节点定义（Go）**

```go
type ListNode struct {
    Val  int
    Next *ListNode
}
```


**解题思路**

把整道题拆成三个独立的小操作，分别写、分别调：

1. **反转一段链表**（原子操作，不管什么题都通用）
   - 三指针 `prev / curr / next` 走位
   - 每步：先记下 `next = curr.Next`，再翻转 `curr.Next = prev`，最后 `prev = curr; curr = next`
   - 循环结束时 `prev` 是新头

2. **探路**（每组开始前判断够不够 k 个）
   - 从 `prevGroupTail` 出发走 k 步，能走完 → 够；遇到 `nil` → 不够，直接返回

3. **断开 + 反转 + 拼回**
   - 临时把 `groupEnd.Next = nil` 切断当前组
   - 反转得到新头（= 原 `groupEnd`）
   - `prevGroupTail.Next = 新头`，`groupStart.Next = nextGroupStart`
   - 更新 `prevGroupTail = groupStart`（反转后 groupStart 变成尾巴）

**关键技巧**：用**哑节点 (dummy)** 简化「头节点也可能被反转」的边界处理，省去对第一组的特殊判断。


**源码参考**

迭代法（O(n) 时间，O(1) 空间，**推荐**）：

```go
func reverseKGroup(head *ListNode, k int) *ListNode {
    dummy := &ListNode{Next: head}
    prevGroupTail := dummy

    for {
        // ① 探路: 走 k 步到 groupEnd
        groupStart := prevGroupTail.Next
        groupEnd := prevGroupTail
        for i := 0; i < k; i++ {
            groupEnd = groupEnd.Next
            if groupEnd == nil {
                return dummy.Next
            }
        }

        // ② 断开 + 反转
        nextGroupStart := groupEnd.Next
        groupEnd.Next = nil
        reversedHead := reverse(groupStart)

        // ③ 拼回
        prevGroupTail.Next = reversedHead
        groupStart.Next = nextGroupStart

        // ④ 更新
        prevGroupTail = groupStart
    }
}

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
```

递归法（O(n) 时间，O(n/k) 栈空间）：

```go
func reverseKGroupRecursive(head *ListNode, k int) *ListNode {
    tail := head
    for i := 0; i < k; i++ {
        if tail == nil {
            return head
        }
        tail = tail.Next
    }
    newHead := reverseRange(head, tail)
    head.Next = reverseKGroupRecursive(tail, k)
    return newHead
}

// 反转 [head, end), 左闭右开
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
```


**复盘**

### 复杂度

| | 时间 | 空间 | 适用 |
|---|---|---|---|
| 迭代 | O(n) | **O(1)** | 面试默认推荐 |
| 递归 | O(n) | O(n/k) 栈 | 简洁，但链表巨长会栈溢出 |

### 4 个易错点

1. **探路步数**：从 `prevGroupTail` 走 k 步 vs 从 `groupStart` 走 k-1 步。一定先在纸上画一遍。
2. **必须先切断再反转**：否则 `reverse()` 会反转到链表末尾。
3. **拼回时新尾巴是 `groupStart`**：反转后 start 和 end 角色互换，别写错。
4. **nil 检查**：探路循环里 `if groupEnd == nil { return }` 必须有，否则 panic。

### Follow-up

| 问题 | 答案 |
|---|---|
| 递归 vs 迭代选哪个 | n ≤ 10^4 都行；n ≥ 10^6 必须迭代（栈溢出） |
| 不足 k 个也要翻转 | 探路允许 nil，最后一组单独反转 |
| 只翻转奇数组 | 加 `groupIdx` 计数，偶数组只移 `prevGroupTail` |
| Go 链表的指针陷阱 | 任何 `.Next` 之前都要确认非 nil |

### 链表题套路总结

这道题打开了「**链表反转家族**」整个套路：

| 题号 | 题目 | 核心 |
|---|---|---|
| LC 206 | 反转链表 | 三指针走位（最基础） |
| LC 92 | 反转链表 II（区间反转） | 哑节点 + 头插法 |
| **LC 25** | **K 个一组反转** | 探路 + 调用基础反转 |
| LC 24 | 两两交换节点 | LC 25 的 k=2 特例 |
| LC 234 | 回文链表 | 找中点 + 反转后半段 |

掌握 LC 25 后，前 4 题都能 5 分钟搞定。
