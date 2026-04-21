# 2026-04-21 复盘

## 今日题目

### 算法
- **LeetCode 25 K 个一组翻转链表** —— [question.md](./question.md) / [question.go](./question.go)
  - 链表反转的「九九乘法表」：三指针 `prev/curr/next` 走位
  - 哑节点 (dummy) 简化头节点变化的边界
  - 探路 + 断开 + 反转 + 拼回 四步走
  - 迭代 O(1) 空间 vs 递归 O(n/k) 栈空间
  - 链表反转家族（LC 206 / 92 / 25 / 24 / 234）的统一套路

## 今日八股

### MySQL
- **索引原理（B+ 树）** —— [bagu/mysql/index.md](./bagu/mysql/index.md)
  - 为什么选 B+ 树（vs 哈希 / 红黑树 / 跳表 / B 树）
  - 聚簇索引 vs 非聚簇索引、回表与覆盖索引
  - 联合索引的最左前缀原则、索引下推 (ICP)
  - 7 大索引失效场景（函数、隐式转换、`%xxx`、OR、范围后列…）
  - 10 个高频面试题速答（B+ vs B / 自增 ID / 红黑树 / 跳表对比…）
  - EXPLAIN 实战诊断流程
- **MySQL 实战拷打 Q&A** —— [bagu/mysql/qa.md](./bagu/mysql/qa.md)
  - **Q1**: MySQL 为什么用 B+ 树（含数字估算：3 层索引 10 亿行的由来）
  - **Q2**: 订单表主键选 UUID / 自增 / Snowflake（页分裂 + 索引膨胀 + 分布式唯一）
  - **Q3**: 分库分表 4 种类型 + 4 种分片策略；为什么不用主从/MGR；为什么不直接上 TiDB/OceanBase

### Go
- **内存模型 + 垃圾回收（结合链表场景）** —— [bagu/go/memory-and-gc.md](./bagu/go/memory-and-gc.md) / [bagu/go/escape_demo.go](./bagu/go/escape_demo.go)
  - 栈 vs 堆，逃逸分析的 6 个常见场景
  - 链表 `*ListNode` 必然逃逸，对 GC 的压力分析
  - mcache / mcentral / mheap 三级分配器
  - 三色标记 + 混合写屏障，Go 1.0 → 1.20 GC 演进（STW 从数百 ms → < 1ms）
  - GC 触发时机（Pacer + 定时 + GOGC + GOMEMLIMIT）
  - 链表场景优化：sync.Pool / 数组模拟链表
  - 实战：`go build -gcflags="-m"` 看每行的逃逸决定

## 知识打通：今天和昨天的连接

- **链表的指针操作 ↔ B+ 树叶子节点的双向链表**：B+ 树叶子节点之间用双向指针连，范围查询沿链表扫——和今天 LC 25 的链表操作思想一致。
- **哑节点 ↔ 数据库的「头页」**：都是为了简化「第一个元素特殊处理」的边界。
- **MySQL 选 B+ 树不选跳表，Redis ZSet 选跳表不选 B+ 树**：本质是「磁盘 vs 内存」的权衡。这是昨天 Redis 八股 + 今天 MySQL 八股的联合考点。

## 待提交文件

```
2026-04-21/
├── question.md                   # LC25 完整题面 + 解法 + 复盘
├── question.go                   # 迭代 + 递归两版, 7 用例全绿
├── notes.md                      # ← 本文件
└── bagu/
    ├── mysql/
    │   ├── index.md              # MySQL 索引深度八股
    │   └── qa.md                 # Q&A 实战拷打 (索引/主键/分库分表)
    └── go/
        ├── memory-and-gc.md      # Go 内存模型 + 三色标记 GC
        └── escape_demo.go        # 逃逸分析可运行 demo
```
