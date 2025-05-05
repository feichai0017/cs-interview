# Go 内存模型与内存逃逸

## 1. Go 内存模型（Memory Model）

* **Happens‑Before 关系**：如果操作 A "happens-before" 操作 B，则 A 的所有写入对 B 可见。
* **同步原语**：`sync.Mutex`、`sync.RWMutex`、`sync/atomic`、`channel` 等都会建立 happens-before 关系。
* **数据竞争**：两个 goroutine 对同一内存位置同时至少有一个写，且无 happens‑before 关系，属于未定义行为。

**示例**：

```go
var mu sync.Mutex
var x int

go func() {
    mu.Lock()
    x = 42 // A
    mu.Unlock()
}()

mu.Lock()
fmt.Println(x) // B
mu.Unlock()
```

在上例中，A happens-before B，因此 B 一定能看到 x=42。

---

## 2. 内存逃逸（Escape Analysis）

* **栈分配 vs 堆分配**：

  * 栈上分配：快速，随函数调用进出栈自动回收。
  * 堆上分配：由 GC 管理，可跨函数返回，但开销更高。

* **触发逃逸的典型场景**：

  1. 返回局部变量地址：

     ```go
     func Foo() *int { x := 5; return &x }
     ```
  2. 闭包捕获外层变量：

     ```go
     func Bar() func() int { x := 0; return func() int { x++; return x } }
     ```
  3. 接口赋值：

     ```go
     var w io.Writer = bytes.NewBuffer(nil)
     ```
  4. 写入更长生命周期的数据结构（slice、map）中的指针。

* **查看逃逸**：

  ```bash
  go build -gcflags="-m" .
  ```

  输出中 `moved to heap` 即表示逃逸。

* **优化策略**：避免返回局部指针，显式复用对象（`sync.Pool`），拆分大结构体，减少接口装箱。

---

## 3. Channel 底层实现

### 3.1 环形缓冲区（ring buffer）

带缓冲通道使用 `hchan`：

```go
// 省略部分字段
type hchan struct {
    qcount   uint      // 当前元素数量
    dataqsiz uint      // 缓冲区容量
    buf      unsafe.Pointer // 元素存储区
    elemsize uint16    // 单元素大小
    sendx    uint      // 写索引
    recvx    uint      // 读索引
    sendq    waitq     // 等待发送的 goroutine 队列
    recvq    waitq     // 等待接收的 goroutine 队列
    lock     mutex     // 保护所有字段
    closed   uint32    // 是否关闭
}
```

* 写时：`buf[sendx] = value; sendx = (sendx+1)%dataqsiz; qcount++`。
* 读时：`value = buf[recvx]; recvx = (recvx+1)%dataqsiz; qcount--`。
* 无缓冲通道（cap=0）则不分配 `buf`，所有数据经 `sendq`/`recvq` 直接配对。

### 3.2 sendq 与 recvq（waitq）

* **waitq**：双向链表，节点 `sudog` 保存被阻塞的 goroutine 和待传数据。
* **sendq**：缓冲区满或无接收时，发送者加入此队列。
* **recvq**：缓冲区空或无发送时，接收者加入此队列。
* **配对唤醒**：

  1. 发送者到来且 `recvq` 非空 → dequeue 一个接收者，直接唤醒并 hand-off 数据。
  2. 接收者到来且 `sendq` 非空 → 同理从 `sendq` dequeue。

### 3.3 并发安全机制

* **互斥锁**：所有对 `hchan` 字段的修改都由内部 `lock` 保护。
* **原子检查**：`len()`/`cap()` 等快速路径使用 `atomic.Load`，避免不必要的加锁。
* **Goroutine 挂起/唤醒**：使用 `gopark`/`goready` 在 `sendq`/`recvq` 上高效阻塞和恢复。

---

## 4. 多 goroutine 并发消费

* 一个 channel 可被任意多个 goroutine 同时发送和接收。
* 缓冲区和 `sendq`/`recvq` 共同支持多生产者/多消费者场景，不需显式区分角色。
* 对端唤醒逻辑保证了所有等待的 goroutine 都能有序完成一次一对一的数据交换。
