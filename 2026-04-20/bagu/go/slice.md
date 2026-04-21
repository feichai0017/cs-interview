# Go 八股：Slice 底层原理与 append 扩容

Slice 是 Go 面试**最高频的八股**，没有之一。下面按「底层结构 → 创建方式 → 扩容机制 → append 陷阱 → 参数传递 → 内存泄漏 → 典型考题」的顺序讲透。


## 1. 底层结构：SliceHeader

```go
type slice struct {
    array unsafe.Pointer  // 指向底层数组的指针
    len   int             // 当前长度 (可用元素数)
    cap   int             // 容量 (底层数组剩余空间)
}
```

**三个字段，24 字节**（64 位机器）。这就是为什么传 slice 进函数是「小对象拷贝」——拷贝的是这 24 字节的 header，**底层数组共享**。

关键图示：

```
s := make([]int, 3, 5)
s[0], s[1], s[2] = 1, 2, 3

    ┌───────────────────────┐
    │ s (SliceHeader, 24B)  │
    │   array ──┐           │
    │   len = 3 │           │
    │   cap = 5 │           │
    └───────────┼───────────┘
                ▼
    ┌─────┬─────┬─────┬─────┬─────┐
    │  1  │  2  │  3  │  0  │  0  │   ← 底层数组, 容量 5
    └─────┴─────┴─────┴─────┴─────┘
     [0]   [1]   [2]   [3]   [4]
     └── len=3 ──┘
     └──── cap=5 ─────────────────┘
```


## 2. Slice vs Array

| 特性 | Array `[5]int` | Slice `[]int` |
|---|---|---|
| 类型 | **值类型** | **引用类型**（头 + 指针） |
| 长度 | 类型一部分，固定 | 运行时可变 |
| 参数传递 | **完整拷贝** | 拷贝 header（24B），共享底层 |
| 可以 `==` 比较 | ✅（元素逐个比较） | ❌（只能和 `nil` 比） |
| 典型用途 | 很少直接用 | **99% 场景用这个** |

**坑**：`[5]int` 和 `[6]int` 是**完全不同的类型**，不能互相赋值。


## 3. 创建 Slice 的 4 种方式

```go
// 1) 字面量
a := []int{1, 2, 3}              // len=3, cap=3

// 2) make 指定长度
b := make([]int, 5)              // len=5, cap=5, 元素全 0

// 3) make 指定长度和容量
c := make([]int, 3, 10)          // len=3, cap=10, 前 3 个是 0, 后 7 个预留

// 4) nil slice
var d []int                      // len=0, cap=0, 底层指针是 nil
```

**nil slice vs empty slice**：

```go
var d []int              // nil slice:   header = {nil, 0, 0}
e := []int{}             // empty slice: header = {某地址, 0, 0}

len(d) == 0  && cap(d) == 0   // ✅
len(e) == 0  && cap(e) == 0   // ✅
d == nil                      // true
e == nil                      // false   <- 这是区别

// 但对大多数操作它们行为一致:
d = append(d, 1)              // ✅ nil slice 可以直接 append
for range d { }               // ✅ 循环 0 次
```

**面试官常问**：「nil slice 能不能 append？」—— 能，因为 append 会自动分配底层数组。


## 4. 扩容机制（Go 1.18+ 新规则，重要！）

这是**面试最高频的单点**，很多人背的是老版本规则（「cap 小于 1024 翻倍，大于 1024 增长 25%」），**Go 1.18 之后规则变了**。

### Go 1.18 之前

```
if oldCap < 1024:
    newCap = oldCap * 2
else:
    newCap = oldCap * 1.25
```

### Go 1.18+ 新规则（`runtime/slice.go` 的 `growslice`）

```
threshold = 256

if oldCap < threshold:
    newCap = oldCap * 2
else:
    // 渐进式增长: 初始 2x，随 cap 增大逐渐趋近 1.25x
    newCap = oldCap + (oldCap + 3*threshold) / 4
```

这个公式的意图：**小 slice 快速翻倍**，**大 slice 平滑过渡到 1.25x** 增长，避免老规则在 1024 附近的「断崖」。

### 还有一层：内存对齐

`runtime/malloc.go` 的 `roundupsize` 会把计算出的 `newCap * sizeof(T)` 向上取整到 **Go 的内存分配器 size class**（67 个预定义规格），减少内存碎片。

**所以你实测 `cap` 经常不是「精确 2 倍」**，而是稍微多一些——这是 size class 对齐的结果，不是公式算错。

### 举例（`[]int`，每个 8 字节）

| 老 cap | 新增 1 个元素后 cap |
|---|---|
| 0 | 1 |
| 1 | 2 |
| 2 | 4 |
| 4 | 8 |
| 8 | 16 |
| 256 | ~512 |
| 512 | ~848（不是精确 1024） |

具体数值见 [slice_demo.go](./slice_demo.go) 里的实测。


## 5. append 的经典陷阱

### 陷阱 A：append 可能**返回新底层数组**，也可能**不返回**

```go
s := make([]int, 3, 5)  // len=3, cap=5
s[0], s[1], s[2] = 1, 2, 3

t := append(s, 4)       // len=4, cap=5   ← 没扩容, 用同一个底层数组
u := append(s, 4, 5, 6) // len=6, cap ≥6  ← 扩容了, 新底层数组

t[0] = 999              // 修改 t, s[0] 也变成 999！
u[0] = 888              // 修改 u, s[0] 不变, 因为底层不同了
```

**这是 Go 里最容易踩的坑之一**。`append` 不承诺返回新底层数组，它只承诺「容量够就原地写，不够就分配新的」。

### 陷阱 B：共享底层的 append 会**互相覆盖**

```go
s := []int{1, 2, 3, 4, 5}
a := s[:3]              // [1 2 3], len=3, cap=5
b := append(a, 99)      // [1 2 3 99], len=4, cap=5   ← 原地写, 覆盖了 s[3]!
fmt.Println(s)          // [1 2 3 99 5]   ← s 被意外修改
fmt.Println(b)          // [1 2 3 99]
```

**这个 bug 在生产里非常常见**。解决方法：

```go
// ① 明确复制
a := make([]int, 3)
copy(a, s[:3])

// ② 用 full slice expression 限制容量
a := s[:3:3]            // 第三个参数 = cap, 现在 append 必扩容
b := append(a, 99)      // 新底层数组, 不动 s
```

**`s[low:high:max]` 这个语法**（full slice expression）就是为了解决这个问题。`max` 设置上限容量，让后续 `append` 必然触发扩容。


## 6. Slice 作为函数参数

**slice 不是「引用」，是「header 按值拷贝 + 共享底层」**。

```go
func modify(s []int) {
    s[0] = 999           // ✅ 生效, 改的是共享底层
    s = append(s, 4)     // ❌ 对外不可见, 修改的是局部 header
}

s := []int{1, 2, 3}
modify(s)
// s[0] 是 999, 但 len(s) 还是 3
```

**要在函数里改 slice 长度**，三种做法：

```go
// ① 返回新 slice (Go 标准做法, 类似 append)
func addOne(s []int) []int {
    return append(s, 1)
}
s = addOne(s)

// ② 传指针
func addOne(s *[]int) {
    *s = append(*s, 1)
}
addOne(&s)

// ③ 预先 make 够大, 在函数里只 index 写
```


## 7. 子切片导致的内存泄漏

```go
func readFirstLine(filename string) []byte {
    data, _ := os.ReadFile(filename)   // 假设读了 1GB
    return data[:100]                  // 只要前 100 字节
}
```

**隐患**：返回的 100 字节 slice **共享** 1GB 的底层数组，GC 只要这个 slice 还存活，就不会回收那 1GB。

**正确做法**：显式拷贝脱离底层。

```go
func readFirstLine(filename string) []byte {
    data, _ := os.ReadFile(filename)
    out := make([]byte, 100)
    copy(out, data[:100])
    return out                         // 新底层, 原 1GB 可被 GC
}
```

这个陷阱在处理大文件、解析大 JSON、切网络包的场景里**特别常见**。


## 8. copy 函数

```go
copy(dst, src)   // 返回实际拷贝元素数 = min(len(dst), len(src))
```

要点：
- **按 len 拷贝，不按 cap**。`dst` 没长度就拷不进去（需要 `make([]T, n)` 预留）。
- dst 和 src 可以重叠，`copy` 会处理正确（内部用 `memmove` 语义）。
- **`copy` 不会扩容 dst**。要扩容用 `append`。

```go
src := []int{1, 2, 3, 4, 5}
dst := make([]int, 3)
n := copy(dst, src)            // n=3, dst=[1 2 3]
```


## 9. 面试高频题

### Q1：slice 和数组区别？
见第 2 节。关键点：**值类型 vs 引用类型、长度固定 vs 可变、参数传递代价**。

### Q2：slice 底层结构？
三字段 header：`ptr, len, cap`，共 24 字节（64 位）。

### Q3：append 扩容机制？
**必须答新规则**（Go 1.18+）：阈值 256，小于 2x 增长，大于后渐进趋近 1.25x，最后还要按 size class 对齐。

### Q4：下面代码输出什么？
```go
s := []int{1, 2, 3}
s1 := append(s, 4)
s2 := append(s, 5)
fmt.Println(s1, s2)
```
**考点**：`s` 的 `cap` 是 3，append 都会扩容，但两次 append 各自分配新数组 → `s1=[1 2 3 4]`, `s2=[1 2 3 5]`。

**变种**：
```go
s := make([]int, 3, 10)
s[0], s[1], s[2] = 1, 2, 3
s1 := append(s, 4)
s2 := append(s, 5)
fmt.Println(s1, s2)
```
这次 `cap=10` 够用，**两次 append 共用底层数组**，第二次覆盖第一次 → `s1=[1 2 3 5]`, `s2=[1 2 3 5]`。

### Q5：如何清空 slice？
```go
s = s[:0]              // 保留底层, len 置 0 (推荐, 可复用内存)
s = nil                // 释放底层引用, 可 GC
s = make([]int, 0)     // 新分配空 slice, 浪费
```

### Q6：`s[low:high:max]` 的 max 有什么用？
**限制 cap**，避免后续 append 原地写污染原 slice。第 5 节陷阱 B 的解法。

### Q7：为什么 slice 不支持 `==`？
官方解释：slice 可能包含自己（通过指针），定义相等语义有歧义；而且按元素比会「意外」O(n)，不符合 `==` 的直觉。想比较用 `reflect.DeepEqual` 或 `slices.Equal`（Go 1.21+）。


## 10. 一句话总结

> **Slice = 指向底层数组的 header（ptr/len/cap）。传参共享底层；append 容量够就原地写、不够就扩容到新地址；小 < 2x、大 → 1.25x、最后按 size class 对齐。共享底层是性能利器，也是 bug 温床。**


## 参考实测

运行 [slice_demo.go](./slice_demo.go) 查看：
1. 实测扩容曲线
2. append 共享底层的坑
3. full slice expression 的修复
4. 内存泄漏场景
