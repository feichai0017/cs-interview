# Redis 八股：五大基础数据结构 + 底层编码

Redis 面试的「第一道坎」就是这个——**你知道 String、List、Hash、Set、ZSet 这五种类型，但你知道它们底层用的是什么数据结构吗？为什么要用这些结构？**

答得上来是「会用 Redis」，答得透彻是「懂 Redis」。

---

## 1. 整体图谱

Redis 的用户看到的是 5 种类型（`redis-cli type <key>` 返回值），但每种类型**内部可能根据数据规模切换不同的底层编码**。这是 Redis 性能的核心秘密：**小数据用紧凑编码省内存，大数据用 O(1)/O(log n) 编码保性能**。

```
┌─────────────────────────────────────────────────────────────┐
│  用户可见类型  │        底层编码 (encoding)                  │
├─────────────────────────────────────────────────────────────┤
│  String       │  int  /  embstr  /  raw (SDS)              │
│  List         │  listpack  →  quicklist                    │
│  Hash         │  listpack  →  hashtable                    │
│  Set          │  intset  /  listpack  →  hashtable         │
│  ZSet         │  listpack  →  skiplist + hashtable         │
└─────────────────────────────────────────────────────────────┘
```

用 `OBJECT ENCODING <key>` 可以看到具体编码。

---

## 2. String：SDS（Simple Dynamic String）

Redis 不直接用 C 字符串，自己实现了 **SDS**。

### C 字符串 vs SDS

| 特性 | C 字符串 (`char *`) | Redis SDS |
|---|---|---|
| 长度 | `strlen()` O(n) 扫描 | `len` 字段 O(1) 获取 |
| 二进制安全 | ❌ 遇 `\0` 截断 | ✅ 按 len 处理 |
| 缓冲区溢出 | ❌ 要手动保证 | ✅ API 自动扩容 |
| 修改次数 | 每次 `realloc` | 预分配 + 惰性释放 |
| 兼容 C 字符串 | — | ✅ 末尾仍有 `\0` |

### SDS 结构（简化）

```c
struct sdshdr {
    uint32_t len;      // 字符串当前长度
    uint32_t alloc;    // 分配的空间总长 (不含 header)
    uint8_t  flags;    // 类型标记
    char     buf[];    // 真实内容, 柔性数组
};
```

**对比 Go 的 string**：

| | Go `string` | Go `[]byte` | Redis SDS |
|---|---|---|---|
| 长度 | `len` 字段 (O(1)) | `len` 字段 | `len` 字段 |
| 容量 | 无（只读） | `cap` 字段 | `alloc` 字段 |
| 可变 | 不可变 | 可变 | 可变 |
| 二进制安全 | ✅ | ✅ | ✅ |

你看，**SDS 其实非常像 Go 的 `[]byte`**——都是「长度 + 容量 + 可变底层数组」，和我们今天早上讲的 slice 一脉相承。

### String 的 3 种编码

```
int     — 值能存成 long (8 字节整数) 时, 直接把整数塞进 redisObject.ptr
embstr  — 字符串长度 ≤ 44 字节, 一次 malloc 同时分配 redisObject + SDS
raw     — 字符串长度 > 44 字节, 两次 malloc 分别分配
```

**为什么是 44？** 因为 Redis 的 jemalloc 分配 64 字节时最划算，`redisObject (16) + SDS header (3) + 字符串 + '\0' (1) = 64`，倒推字符串长度 ≤ 44。

**面试高频追问：为什么 embstr 是只读的？**
因为 redisObject 和 SDS 在一次 malloc 出来的**连续内存**里，修改字符串要扩容就得重新分配，没办法原地改。所以 Redis 5.0 起 embstr 一律当只读处理，修改操作会先转成 raw。


---

## 3. List：listpack → quicklist

### 演化史
- Redis 3.2 前：`ziplist`（元素少）+ `linkedlist`（元素多）
- Redis 3.2 ~ 6.x：`ziplist` + `quicklist`
- **Redis 7.0+：`listpack` + `quicklist`**（`ziplist` 下线）

### listpack（替代 ziplist）

一种**紧凑连续内存**格式。所有元素挤在一块连续空间里，每个元素自描述自己的长度。

```
[total_bytes][num_elements][elem1][elem2]...[elemN][end_marker]
```

**优点**：省内存，cache 友好。
**缺点**：增删中间元素是 O(n)（要 memmove），查第 k 个也是 O(n)。

**为什么 ziplist 被淘汰？**
ziplist 有个经典 bug：**级联更新 (cascade update)**。每个节点存「前一个节点的长度」，如果某个节点变长，后面所有节点可能连锁更新，最坏 O(n²)。listpack 去掉了「前节点长度」字段，改成每个节点存自己的长度前缀 + 长度后缀，彻底消除级联更新。

### quicklist

**listpack 作为节点的双向链表**。

```
head  ◄──►  [listpack节点]  ◄──►  [listpack节点]  ◄──►  ...  ◄──►  tail
              (最多 N 元素或 M 字节)
```

配置项：
- `list-max-listpack-size -2`（每个节点最多 8KB）
- `list-compress-depth 0`（首尾不压缩的节点数）

**设计权衡**：纯链表指针多（每节点 24B 额外开销），纯 listpack 插入慢。quicklist 折中——**整体是链表（改中间 O(1) 到节点，节点内 O(n)）+ 每节点紧凑存储（省内存）**。

---

## 4. Hash：listpack → hashtable

两种编码切换点（默认配置）：
- `hash-max-listpack-entries 128`（元素数 ≤ 128）
- `hash-max-listpack-value 64`（每个 value 长度 ≤ 64 字节）

超过任一阈值 → 升级为 hashtable，**不会降回去**（Redis 不做降级）。

### hashtable（dict）

Redis 的 dict 是**数组 + 链表**的经典哈希表，但有两个亮点：

**① 渐进式 rehash**

扩容时不会一次性搬完所有 kv（大 hash 一次搬会卡住主线程），而是维护**两个 table**：

```c
typedef struct dict {
    dictht ht[2];          // 两个哈希表
    int rehashidx;         // -1 表示不在 rehash, 否则是当前进度
    // ...
};
```

- 每次增删改查操作**顺带搬一个 bucket** 从 ht[0] 到 ht[1]。
- 定时任务也会做一些搬迁。
- 搬完后 ht[0] = ht[1]，ht[1] 清空。

**好处**：均摊 O(1)，避免一次性搬迁阻塞。
**代价**：rehash 期间查找要查两个表。

**② 触发扩容的条件**

- 负载因子 `ratio = used / size > 1` 且**没有 bgsave/bgrewriteaof 子进程**时触发扩容。
- 有子进程时阈值放宽到 `ratio > 5`（为了避免 copy-on-write 触发大量页复制）。
- 负载因子 `< 0.1` 触发缩容。

**面试陷阱**：「Redis hash 冲突怎么解决？」—— 链地址法（拉链）。「为什么不开放寻址？」—— 渐进式 rehash 对链表友好，元素可以按 bucket 批量搬。

---

## 5. Set：intset / listpack → hashtable

三种编码：
- **intset**：所有元素都是整数时用。底层是有序 int 数组 + 二分查找（O(log n) 成员判定）。
- **listpack** (Redis 7.2+)：元素少且不全是整数时用。
- **hashtable**：超出阈值后用。key = 元素，value 为空。

切换阈值：`set-max-intset-entries 512`、`set-max-listpack-entries 128`。

---

## 6. ZSet：listpack → skiplist + hashtable

这个是面试**王者题**。ZSet 的底层是 **跳表 (skiplist) + 字典 (hashtable)** 双结构。

### 为什么双结构？

- **hashtable**：`{member: score}`，支持 O(1) 按 member 取 score（`ZSCORE`）。
- **skiplist**：按 score 排序，支持 O(log n) 范围查询（`ZRANGEBYSCORE`、`ZRANK`）。
- **两者共享 member 对象**，只是多存一份指针，不是复制。

### 跳表原理（面试必考）

```
Level 3:  head ──────────────────────► 30 ─────────────► NIL
Level 2:  head ─────► 10 ────────────► 30 ─────► 50 ──► NIL
Level 1:  head ─► 5 ► 10 ────► 20 ──► 30 ► 40 ► 50 ──► NIL
Level 0:  head ─► 5 ► 10 ► 15 ► 20 ► 25 ► 30 ► 35 ► 40 ► 45 ► 50 ► NIL
```

**本质**：带多层索引的有序链表。每个节点以概率 p（Redis 用 0.25）「晋升」到上一层。

**查找** O(log n)：从最高层开始，能往右就往右，不能就往下。
**插入** O(log n)：找到位置后随机决定新节点的层数。
**范围查询** O(log n + k)：定位起点后沿最底层链表走。

**跳表 vs 红黑树**：
- 跳表实现更简单（红黑树的旋转逻辑复杂）
- 跳表范围查询更快（底层就是有序链表）
- 跳表并发友好（局部改动不需要全局重平衡）
- 红黑树常数更小，空间更紧凑

**为什么 Redis 选跳表？** 作者 antirez 亲自回答：**范围查询是 ZSet 的核心需求，跳表对范围查询最友好；实现复杂度低；并发扩展性好**。

### listpack 小对象优化

ZSet 元素少（≤ 128）且每个 value 短（≤ 64B）时用 listpack，元素按 `[member, score, member, score, ...]` 顺序排列，线性扫描。

切换阈值：`zset-max-listpack-entries 128`、`zset-max-listpack-value 64`。

---

## 7. 常见面试题速答

### Q1：Redis 为什么快？
1. **纯内存**，无磁盘 IO（除非 AOF fsync）
2. **单线程**处理命令，无锁开销（注意：IO 多线程是 6.0 引入的，但执行还是单线程）
3. **多路复用 IO** (epoll)
4. **高效数据结构**（SDS、listpack、skiplist、渐进 rehash 等）
5. **优化的编码**（小对象用紧凑格式省内存 + cache 友好）

### Q2：为什么 Redis 选单线程？
- 内存操作本身是 ns 级，CPU 不是瓶颈，瓶颈在 IO
- 避免多线程的锁、上下文切换开销
- 代码简单、容易实现事务和原子操作

但 Redis 6.0 加了**多线程 IO**：网络读/写用多线程，命令执行仍单线程。大 value 场景能显著提升吞吐。

### Q3：String 能存多大？
**512 MB**，硬编码上限。实战建议单 value 不超过 **1 MB**，否则大 key 会阻塞。

### Q4：为什么 ZSet 用跳表不用 B+ 树？
B+ 树为磁盘设计（多叉减少 IO 次数），Redis 全内存用不上多叉的优势；跳表更简单、范围查询更快、并发友好。

### Q5：Redis 过期 key 怎么清理？
- **惰性删除**：访问 key 时才检查过期
- **定期删除**：每秒 10 次随机抽样 20 个带过期时间的 key，过期的删除，若过期比例 > 25% 重复抽样

这两种配合：惰性删不抢 CPU 但可能留内存，定期删兜底。**极端情况仍可能堆积**，最后靠 maxmemory 淘汰策略兜底。

### Q6：有哪些淘汰策略？（8 种）
```
noeviction         (默认) 内存满直接拒绝写
allkeys-lru        所有 key 中 LRU 淘汰
allkeys-lfu        所有 key 中 LFU 淘汰 (4.0+)
allkeys-random     所有 key 中随机淘汰
volatile-lru       只在带 TTL 的 key 中 LRU 淘汰
volatile-lfu       只在带 TTL 的 key 中 LFU 淘汰
volatile-random    只在带 TTL 的 key 中随机淘汰
volatile-ttl       只在带 TTL 的 key 中淘汰最快过期的
```

**LRU vs LFU**：LRU 看「多久没用」，LFU 看「用的频率」。热点数据偏向 LFU，访问模式均匀偏向 LRU。

### Q7：Redis 的 LRU 是怎么实现的？
**不是标准 LRU 双链表**（那个实现对每次访问都要 O(1) 移动节点，内存开销大）。Redis 是**近似 LRU**：每个 key 存一个 `lru` 时间戳，淘汰时随机抽样 N 个 key，淘汰时间戳最旧的。`maxmemory-samples` 默认 5，调大精度越高但越慢。

---

## 8. 一张速查表

| 类型 | 小规模编码 | 大规模编码 | 典型用途 |
|---|---|---|---|
| String | int / embstr | raw (SDS) | 缓存对象、计数器、分布式锁 |
| List | listpack | quicklist | 消息队列、时间线、最近记录 |
| Hash | listpack | hashtable | 对象属性（比 String + JSON 省空间） |
| Set | intset / listpack | hashtable | 去重、标签、共同好友 |
| ZSet | listpack | skiplist + hashtable | 排行榜、延时队列、评分 |

---

## 9. 一句话总结

> **Redis 每种类型都有「小对象紧凑编码（listpack/intset/embstr/int）」和「大对象性能编码（hashtable/skiplist/quicklist/raw）」两套实现，根据规模阈值自动切换。这是 Redis 在「内存效率」和「操作性能」之间做的精妙权衡，也是面试官最喜欢考的点。**
