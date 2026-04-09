# 🧠 Coding 题目回顾
设计一个基于内存的限流器（Rate Limiter）

背景：

每个用户，每分钟最多允许访问 100 次。

超过的话，需要拒绝（返回 429 Too Many Requests）。

要求：支持高并发，并且未来可以扩展到分布式版本。

Bonus（如果有时间）：设计成滑动窗口限流，而不是简单的固定窗口。

方向 | 讨论点
内存优化 | timestamps数组很大时怎么办？怎么裁剪？
并发优化 | 单锁变细粒度锁？分桶？
持久化问题 | 单机挂了怎么办？需要 Redis等外部存储？
分布式限流 | 多机限流，如何一致性？需要用 token bucket？
高并发场景 | 有没有批量处理？减少锁竞争？
TTL清理冷用户 | 长时间没活动的 user 怎么回收资源？

1. 内存优化（Memory Usage）
✅ 你可以说：

问题：每个请求记录一个 time.Time，如果有很多用户、每秒上万次访问，内存很快爆掉。

优化方法：

近似滑动窗口：不存每个请求的时间戳，而是将时间分成小段（如1秒一个 bucket），只记录每秒的请求数，保留最近60个 bucket。

时间复杂度从 O(N) -> O(1)

内存占用从 O(N 请求数) -> O(60)

示例结构：

```go
type bucket struct {
    timestamp time.Time
    count     int
}
```

2. 并发优化（Concurrency）
✅ 你可以说：

问题：当前 RateLimiter 有一把大锁 (sync.Mutex)，所有用户共享，容易锁竞争严重。

优化方法：

按用户分桶（Sharding）：把 userID hash 成不同的 bucket，每个 bucket 有自己的锁。

示例结构：

```go
type ShardedRateLimiter struct {
    shards []RateLimiterShard
}

type RateLimiterShard struct {
    mu    sync.Mutex
    users map[string]*userData
}
```
这样锁的粒度细，减少竞争，大幅提高并发性能。

3. 持久化和高可用（Persistence and HA）
✅ 你可以说：

问题：现在的限流器是内存版，服务挂了，限流状态就丢了，容易被攻击。

优化方法：

用 Redis 做持久化的限流。

每个请求到来时，通过 Redis 的 INCR + EXPIRE 命令进行计数。

可以用 Redis Lua 脚本实现原子滑动窗口限流。

例子（伪代码）：

```lua
if redis.call('EXISTS', key) == 0 then
    redis.call('SET', key, 1, 'EX', 60)
else
    redis.call('INCR', key)
end
```

4. 分布式限流（Distributed Rate Limiting）
✅ 你可以说：

问题：单机限流在多台机器部署下不生效（每台机器单独计算）。

优化方法：

统一用 Redis、Etcd、或者基于 Kafka/消息队列来集中限流控制。

可以采用 Token Bucket 算法，统一发放 Token，每台机器从 Token Server 拿 token 处理请求。

需要保证一定的一致性（弱一致通常足够）。

5. 冷用户清理（TTL for inactive users）
✅ 你可以说：

问题：一些用户访问完一次后长时间不活跃，浪费内存。

优化方法：

给每个用户加个 TTL（比如最后访问时间超过5分钟），定时清理冷用户。

# 👨‍💻 System Design: Global Rate Limiting System

## 🔹 题目背景

> 设计一个全球范围，支持多Region，多机器的综合限流系统。

需求：
- 每个用户每分钟最多允许100次请求
- 用户请求可能打到任意机器，不同 Region
- 限流需要正确，不允许超量
- 系统需要高可用，低延时


## 📈 初步系统设计思路

- 设置一个 **中央调度中心 (Dispatcher)** ，综合调度资源，进行限流判定
- 每台访问节点都应该有本地限流算法
- 使用 **分布式 KV 系统（etcd）** ，做服务发现和分布式数据综合
- 所有计算依据 **全局统一时间戳（同步 NTP或统一 Clock服务）**


## 🔹 调度过程

1. 请求进入 API Gateway
2. Gateway 转发到随机选择的 Limiter Node
3. 本地限流进行初步判定
4. 如果接近 Threshold，向中央 Dispatcher Push 本地统计数据
5. Dispatcher 统合判定过去60秒内总请求次数
6. 返回 Allow / Reject 结果


## 🔹 性能优化

- 每台机器 **自动 Push 统计信息**，减少 Dispatcher 请求压力
- 将总请求量换算成每机 local threshold，大部分请求本地判定
- 使用滑动窗口分段统计，减少内存压力


## 🔹 VIP 用户流量倾斜处理

- 对 VIP 用户，分配更高权重（weight功能）
- 为 VIP 用户单独设置阈值 Threshold
- 调度器根据负载，有意识将 VIP 请求分散到不同节点
- 支持动态调整


## 🔹 节点故障处理

- 相信 etcd 通过 **Raft 共识算法**，保证服务发现和限流统计数据的强一致性
- 限流节点可以定期 Push 自己状态到 etcd
- 节点重启后，从 etcd 拉取最新状态，快速恢复运行
- 多Region处理，可考虑 etcd 跨地区副本处理，增强异地耐性


---

# 🔹 总结

用户提出的观点和方案，系统性，展示出对分布式系统、限流算法、高可用性、性能优化等方面的熟悉程度，足以在大型科技公司中突显出强大的系统设计能力。


