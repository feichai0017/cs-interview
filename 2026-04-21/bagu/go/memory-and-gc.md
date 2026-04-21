# Go 八股：内存模型 + 垃圾回收（结合链表场景）

今天写 LC 25 链表反转，每一行 `&ListNode{}` 都涉及到「这玩意儿分配在哪？谁回收？什么时候回收？」——这就是 Go 内存管理 + GC 的全部话题。

---

## 1. Go 的内存层次（先建立全局观）

```
进程虚拟内存
├── 代码段 (text)         编译后的指令
├── 数据段 (data/bss)     全局变量
├── 堆 (heap)             ← runtime 管理, GC 扫描
├── ...
└── 栈 (stack)            ← 每个 goroutine 一个, 函数返回自动释放
```

**核心问题**：一个对象创建在堆还是栈，**决定了它的生命周期、分配速度、GC 压力**。

| | 栈 | 堆 |
|---|---|---|
| 分配速度 | **极快**（移动 SP 指针） | 慢（要找空闲块、可能触发扩容） |
| 回收 | 函数返回**自动**释放 | **GC** 异步回收 |
| 大小限制 | goroutine 栈 8KB 起步，最大 1GB | 受系统内存限制 |
| 并发安全 | 天然安全（goroutine 私有） | 需要程序员处理 |
| GC 压力 | 0 | 大对象多 = GC 慢 |

**结论**：**能在栈上的就别上堆**。这就是 Go 编译器**逃逸分析**要做的事。

---

## 2. 逃逸分析（Escape Analysis）

**Go 编译器在编译期决定**：一个变量到底放栈还是堆。

### 决策原则
> **如果一个变量的生命周期超出当前函数**（被外部持有、被指针引用、被存进堆），就**逃逸到堆**。

### 6 个常见的逃逸场景

#### ① 返回局部变量的指针 → **逃逸**
```go
func newNode() *ListNode {
    n := ListNode{Val: 1}    // 想放栈, 但...
    return &n                // 指针被外部持有, 必须逃逸到堆
}
```

#### ② 闭包引用外部变量 → **逃逸**
```go
func counter() func() int {
    count := 0           // count 被闭包持有, 逃逸
    return func() int {
        count++
        return count
    }
}
```

#### ③ 大对象（栈装不下） → **逃逸**
```go
func bigArray() {
    arr := [1 << 20]int{}  // 8MB 数组, 直接逃逸到堆
    _ = arr
}
```

#### ④ interface 装箱 → **逃逸**
```go
func print(x interface{}) {
    fmt.Println(x)        // x 被转成 interface, 通常逃逸
}
print(42)                  // 42 装箱成 interface{}, 逃逸
```

#### ⑤ slice / map 容量编译期不确定 → **逃逸**
```go
func mkSlice(n int) []int {
    return make([]int, n)  // n 是变量, 逃逸
}

func mkSliceFixed() []int {
    return make([]int, 100) // 编译期确定, 但因为返回, 还是逃逸
}
```

#### ⑥ goroutine 引用外部变量 → **逃逸**
```go
func bg() {
    x := 42
    go func() { println(x) }()  // x 被 goroutine 持有, 逃逸
}
```

### 怎么看一个变量是否逃逸？

```bash
go build -gcflags="-m" main.go
```

输出示例：
```
./main.go:5:9: &ListNode{...} escapes to heap     ← 逃逸了
./main.go:8:6: moved to heap: x                    ← 移到堆
./main.go:12:13: ... does not escape               ← 没逃逸, 在栈上
```

**实战调优**：性能敏感代码用 `-gcflags="-m -m"` 看详细原因，针对性消除逃逸。

---

## 3. 链表场景的逃逸分析（联动今天的题）

```go
type ListNode struct {
    Val  int
    Next *ListNode
}

func main() {
    dummy := &ListNode{Next: head}       // ← 逃逸到堆?
    prevGroupTail := dummy               // ← 是同一个对象
    // ...
}
```

**`&ListNode{}` 一定逃逸到堆**，原因：
1. 用了 `&` 取地址 → 编译器看到指针
2. 指针被赋给 `prevGroupTail` 这种长生命周期变量
3. 链表节点要互相引用，**所有 `*ListNode` 都得在堆上**（栈帧内的对象不能被另一个栈帧的指针指向，否则函数返回会野指针）

**结论**：**链表是必然分配在堆上的数据结构**。每个节点 = 一次堆分配 + GC 跟踪一个对象。

### 链表对 GC 的压力

假设你有 100 万节点的链表：
- **100 万次堆分配**（虽然 Go 用 mcache 优化得很好）
- **GC mark 阶段要扫描全部 100 万个 `Next` 指针**
- 链表 cache 不友好（节点散落在堆各处）

### 链表场景的优化技巧

**① 用 sync.Pool 复用节点**
```go
var nodePool = sync.Pool{
    New: func() interface{} { return &ListNode{} },
}

func getNode(val int) *ListNode {
    n := nodePool.Get().(*ListNode)
    n.Val = val
    n.Next = nil
    return n
}

func putNode(n *ListNode) {
    nodePool.Put(n)
}
```
减少分配 + 减少 GC 压力。但注意**手动管理生命周期容易出 bug**，不到性能瓶颈不要用。

**② 数组模拟链表（Cache Friendly）**
```go
type Node struct {
    Val  int
    Next int  // 用下标代替指针
}
nodes := make([]Node, 1000000)
```
所有节点在连续内存里，cache 命中率高，GC 只看到**一个大对象**而不是 100 万个小对象，**GC 压力骤降**。

实战中，**LRU、跳表、B+ 树**这种「链表 + 高并发」场景，常用「数组池 + 下标链」的思路。

---

## 4. Go 的堆分配器：mcache / mcentral / mheap

```
goroutine A → P (Processor) → mcache → 拿小对象
                                ↓ 缓存空了
                              mcentral → 中央列表
                                ↓ 也空了
                               mheap   → 向 OS 要内存
```

**3 级缓存**：
- **mcache**：每个 P 一份，**无锁**分配，最快
- **mcentral**：全局，按 size class 分组，分配时要加锁
- **mheap**：管理所有内存页，向 OS 申请

**67 种 size class**：8B / 16B / 32B / 48B / ... / 32KB，按规格预分配，避免内存碎片。

> **注意：和昨天 Redis SDS embstr 44 字节边界是同一个思路**——按内存分配器的固定档位倒推数据结构大小。

**大对象（> 32KB）**：直接走 mheap，不用 mcache。

---

## 5. Go GC 算法：三色标记 + 写屏障

### 三色标记（Tri-color Marking）

```
白色 (white) - 未被访问的, 可能是垃圾
灰色 (gray)  - 已访问, 但子节点还没扫
黑色 (black) - 已访问, 子节点全扫完
```

**算法步骤**：

```
1. 初始: 所有对象白色
2. 从 GC roots (栈上的指针、全局变量) 出发, 把直接可达的对象涂灰
3. 取一个灰色对象:
     扫描它指向的所有对象 → 涂灰
     自己 → 涂黑
4. 重复 3 直到没有灰色
5. 剩下的白色对象 = 垃圾, 可回收
```

### 链表场景的三色标记

```
栈上的 head 指针 (root)
  ↓
节点1 (灰) → 节点2 (灰) → 节点3 (灰) → ...
  ↓
节点1 (黑), 节点2 还在灰
```

**100 万节点的链表 → mark 阶段要遍历全部 100 万指针**，这就是为什么大链表 / 大 map / 大对象池**对 GC 不友好**。

### 写屏障（Write Barrier）

并发标记时的关键：**用户程序还在跑**，可能改对象引用：

```
B 黑 → C 白 (B 已经被扫完, C 是垃圾候选)
用户突然: B.Next = C
没写屏障的话: GC 不知道 B 又指向 C 了, C 被错误回收
```

**Go 1.8+ 用混合写屏障 (Hybrid Write Barrier)**：
- **删除写屏障**：A.Next = nil 时把原来 A.Next 的对象涂灰（防止它被错误回收）
- **插入写屏障**：A.Next = newObj 时把 newObj 涂灰（防止新引用的对象被错误回收）

混合 = 两种结合，避免了 stack rescan 阶段，**STW 时间从几十 ms 降到 < 1ms**。

### Go GC 的发展历程

| 版本 | 核心改进 | STW 时间 |
|---|---|---|
| Go 1.0 | 标记-清除，全程 STW | 数百 ms |
| Go 1.3 | 并行清除（mark 还是 STW） | ~100 ms |
| Go 1.5 | 三色并发标记，引入写屏障 | ~10 ms |
| Go 1.8 | **混合写屏障**，去掉 stack rescan | **< 1 ms** |
| Go 1.12+ | 标记辅助、Pacer 优化 | 持续优化 |
| Go 1.20+ | Soft Memory Limit (`GOMEMLIMIT`) | 同上 |

**当前 Go GC 的 STW 时间 < 1ms**，绝大多数业务感知不到。

---

## 6. GC 触发时机

Go GC 是 **「Pacer 自适应触发」** + **「定时兜底」**：

| 触发方式 | 条件 |
|---|---|
| **Pacer 触发** | 堆增长到 `prev_heap × (1 + GOGC/100)`，默认 `GOGC=100` 即翻倍触发 |
| **定时触发** | 距上次 GC 超过 2 分钟 |
| **手动触发** | `runtime.GC()`（生产慎用） |
| **达到 GOMEMLIMIT** | Go 1.20+ 软内存上限触发 |

**调优技巧**：
- 内存富裕：**调高 `GOGC=200`**（GC 触发更晚，CPU 占用低）
- 内存吃紧：**调低 `GOGC=50`**（更频繁 GC，内存占用低）
- 大堆（几十 GB）：**用 `GOMEMLIMIT` 设置上限**，避免 OOM

---

## 7. 常见面试题速答

### Q1：Go 的 GC 是什么算法？为什么选这个？
**三色标记 + 混合写屏障 + 并发**。选这个是因为：
- **并发**：业务线程和 GC 线程并行，**STW < 1ms**，对 RT 敏感的服务友好
- **三色标记**：天然支持并发（用颜色状态机替代 STW）
- **混合写屏障**：解决了纯插入 / 纯删除写屏障的 stack rescan 问题

### Q2：什么是逃逸分析？怎么看？
编译器在编译期判断变量是否逃出函数作用域。逃出去就分配堆，否则栈。`go build -gcflags="-m"` 能看。

### Q3：栈和堆的区别？
栈：goroutine 私有，函数返回自动释放，分配快。堆：GC 管理，分配慢，需要全局协调。

### Q4：什么会导致逃逸？
- 返回局部变量指针
- 闭包引用
- 太大装不下栈
- interface 装箱
- goroutine 引用
- make 容量编译期不确定

### Q5：sync.Pool 的作用？什么时候用？
**对象复用池**，减少 GC 压力。适合：**高频创建销毁的临时对象**（如 buffer、解析器）。注意：
- Pool 的对象**随时可能被 GC 回收**（每次 GC 时清理一半）
- **不能放有状态的资源**（连接池要用 channel）
- 对小对象效果不大，对大对象（> KB）效果显著

### Q6：goroutine 的栈是怎么管理的？
- 初始 **2KB**（Go 1.4+），按需增长
- 最大 **1GB**（默认）
- 用 **分段栈** → **连续栈**（Go 1.4+）：栈不够时分配双倍新栈，复制旧栈过去
- 收缩：GC 时如果发现栈太大没用，会**收缩**

### Q7：Go GC 怎么和应用代码协调？
**Pacer 算法**：根据上次 GC 后的堆增长速率，预测下次什么时候触发能保证 GC 完成时堆增量 ≤ GOGC 设定。
**辅助 GC（mark assist）**：如果用户协程分配太快，会被强制帮忙做一点 mark 工作，减缓分配速度。

### Q8：GOMAXPROCS 和 GC 的关系？
- GOMAXPROCS = 同时跑用户代码的 P 数量
- GC 默认用 **GOMAXPROCS / 4** 个核心做并发 mark
- 大堆服务可以**调高 GOMAXPROCS** 让 GC 跑得更快

---

## 8. 链表场景的实战优化清单

回到今天的 LC 25：

| 优化 | 收益 | 适用 |
|---|---|---|
| `sync.Pool` 复用 ListNode | 减少分配 + GC 扫描 | 高频创建销毁 |
| 数组模拟链表（下标代替指针） | GC 只看一个大对象，cache 友好 | 节点数量已知 / 高性能场景 |
| 减少 `Next *ListNode` 嵌套结构 | GC mark 阶段更快 | 通用 |
| 避免在循环中创建临时 slice/map | 减少逃逸 | 通用 |
| 用 `[]byte` 而不是 `string` 构建 | 避免拷贝逃逸 | 字符串拼接 |

---

## 9. 一句话总结

> **Go 的内存哲学：能栈则栈（逃逸分析决定），堆上靠三色标记 + 混合写屏障并发 GC，STW < 1ms。链表这种「指针密集」结构必然全在堆上，每个节点都是一次分配 + 一个 GC 扫描对象——大链表场景考虑用数组模拟或 sync.Pool 优化。**


## 参考阅读
- 用 `go build -gcflags="-m -m"` 实测看你的代码哪里逃逸
- 用 `GODEBUG=gctrace=1` 跑你的程序看 GC 日志（每次 GC 的 STW 时间、堆大小）
- 用 `runtime.ReadMemStats` 程序内监控内存
