# Redis 八股：8 大经典使用场景 + 面试陷阱

每个场景讲清楚：**用什么类型、为什么、示例命令、生产陷阱**。

---

## 1. 缓存（最常见，占 80% 用法）

### 场景
DB 扛不住读压力，Redis 挡在前面缓存热点数据。典型：商品详情、用户信息、配置项。

### 实现

```
GET  goods:12345            ← 先查缓存
  未命中
  ↓
SELECT * FROM goods WHERE id=12345
  ↓
SET  goods:12345 {json} EX 600   ← 回填缓存, 10 分钟过期
```

Go 伪代码：

```go
func GetGoods(id int64) (*Goods, error) {
    key := fmt.Sprintf("goods:%d", id)
    if data, err := rdb.Get(ctx, key).Bytes(); err == nil {
        var g Goods
        _ = json.Unmarshal(data, &g)
        return &g, nil
    }
    // 缓存未命中
    g, err := db.QueryGoods(id)
    if err != nil { return nil, err }
    data, _ := json.Marshal(g)
    rdb.Set(ctx, key, data, 10*time.Minute)
    return g, nil
}
```

### ⚠️ 三大经典难题

#### 1.1 缓存穿透（查不存在的数据）
**问题**：黑客构造大量不存在的 id 请求（比如 `goods:-1`），缓存永远不命中，请求全部打到 DB。

**解法**：
- **缓存空值**：查不到也写 `SET goods:-1 "" EX 60`，下次直接挡住
- **布隆过滤器**：启动时把所有合法 id 加布隆过滤器，不存在直接拒绝
- **参数校验**：id < 0 之类的直接前置拦截

#### 1.2 缓存击穿（单 key 失效的瞬间）
**问题**：**单个**热点 key 刚好过期，瞬间所有请求涌向 DB（一般叫「热点 key 雪崩」）。

**解法**：
- **互斥锁**：查 DB 前先抢分布式锁，抢到的人查 DB 回填，其他人等结果
- **逻辑过期**：永不设 TTL，把「逻辑过期时间」写在 value 里，异步刷新
- **热点探测 + 本地缓存**：超级热 key 进 Go 进程的本地 cache（比如 groupcache）

#### 1.3 缓存雪崩（大批 key 同时失效）
**问题**：大量 key 在同一时刻集中过期（比如整点批量刷新），或 Redis 直接挂了。

**解法**：
- **TTL 加随机值**：`rand.Intn(300)` 让过期时间散开
- **多级缓存**：Redis + 本地 LRU + DB 三层，Redis 挂了本地还能扛
- **熔断 + 限流**：打到 DB 的流量做限流，保护 DB
- **Redis 集群**：单点故障变概率故障

### 📝 面试速答

> 「缓存三大问题：**穿透查不存在 → 缓存空值 / 布隆过滤器**；**击穿单热点 → 互斥锁 / 逻辑过期**；**雪崩批量失效 → 随机 TTL / 多级缓存 / 熔断限流**。」

---

## 2. 分布式锁

### 场景
多实例竞争同一资源：扣库存、防重复提交、定时任务防多机器重复执行。

### 实现

**错误姿势**（面试扣分）：
```
SETNX lock:order:123 1
EXPIRE lock:order:123 10    ← 两步不原子! 如果 SETNX 后进程挂了, 锁永不释放
```

**正确姿势**：用 `SET` 的 NX + PX 参数一步到位
```
SET lock:order:123 <random_uuid> NX PX 10000
  NX   不存在才设置 (防止覆盖别人的锁)
  PX   毫秒级过期 (防止死锁)
```

释放锁：**必须校验持有者**（防误删别人的锁），要用 Lua 脚本保证原子性：

```lua
-- 释放锁的标准 Lua
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
```

### ⚠️ 深坑

| 坑 | 现象 | 解法 |
|---|---|---|
| 锁提前过期 | 业务耗时 > TTL，锁到期自动释放，另一个人也拿到锁 | **看门狗**（watchdog）：持锁期间后台线程定时续约，Redisson 自带 |
| 主从切换丢锁 | 主节点刚设完锁就挂，从节点还没同步，failover 后新主没有这个锁 | **RedLock 算法**（向 N/2+1 个独立 Redis 写锁，争议较大）或直接上 etcd/ZooKeeper |
| 误删别人的锁 | A 的锁过期自动释放了，A 还没意识到继续 DEL，把 B 的锁删了 | **value 存 UUID + Lua 校验**（如上） |

### 📝 面试速答

> 「分布式锁用 `SET key uuid NX PX ttl` 一步原子加锁，释放用 Lua 校验 UUID。深坑是**业务超时**（用 Redisson 看门狗续约）和**主从切换丢锁**（RedLock 或换 etcd）。」

---

## 3. 限流

### 场景
接口防刷、API QPS 限制、防爬虫。

### 三种常见算法

#### 3.1 固定窗口计数（最简单）
```
INCR ratelimit:user:1001:{当前分钟}
EXPIRE ratelimit:user:1001:{当前分钟} 60
  如果 > 100 就拒绝
```
**缺点**：窗口边界问题——59 秒和 61 秒之间可能放行 2 倍流量。

#### 3.2 滑动窗口（ZSet 巧用）
```
ZADD ratelimit:user:1001 {时间戳} {时间戳}    ← score 和 member 都用时间戳
ZREMRANGEBYSCORE ratelimit:user:1001 0 {当前时间 - 60s}
ZCARD ratelimit:user:1001
  如果 >= 100 拒绝
```
**思路**：ZSet 当时间序列用，每次请求插入，清理 60 秒前的旧请求，计数当前窗口内的数量。

#### 3.3 令牌桶（Lua 脚本）
最灵活，支持突发流量。标准写法是把「拿令牌 + 更新时间」封装成 Lua 保证原子：

```lua
local tokens = tonumber(redis.call('GET', KEYS[1]) or capacity)
local last = tonumber(redis.call('GET', KEYS[2]) or now)
local refill = (now - last) * rate
tokens = math.min(capacity, tokens + refill)
if tokens >= 1 then
    tokens = tokens - 1
    redis.call('SET', KEYS[1], tokens)
    redis.call('SET', KEYS[2], now)
    return 1
else
    return 0
end
```

### 📝 面试速答

> 「三种算法：**固定窗口 INCR**（有边界问题）、**滑动窗口 ZSet**（精确但占空间）、**令牌桶 Lua**（支持突发，最常用）。生产环境一般用令牌桶 + Lua 脚本保证原子。」

---

## 4. 排行榜（ZSet 的王牌场景）

### 场景
游戏分数榜、热门商品榜、文章点赞榜。

### 实现

```
ZADD rank:game 1500 "player:1001"      ← 加分
ZINCRBY rank:game 10 "player:1001"     ← 加 10 分
ZREVRANGE rank:game 0 9 WITHSCORES     ← Top 10
ZREVRANK rank:game "player:1001"       ← 查某人排名
ZSCORE rank:game "player:1001"         ← 查某人分数
```

**复杂度**：所有操作 O(log n)，完美匹配跳表。

### 进阶技巧

**同分如何排名？** `score` 相同时 ZSet 按 member 字典序排。如果要「同分按时间先后」，把 score 编码成 `分数 * 10^10 + (INT_MAX - 时间戳)`，这样后来的人 score 更小，排在相同分数的前面。

**千万级排行榜怎么办？** 
- 分片：按 `score` 范围分到多个 ZSet（`rank:1000-2000`、`rank:2000-3000`）
- 只维护 Top N：插入时判断 `ZCOUNT >= N` 就 `ZPOPMIN` 淘汰最低分，只保留前 N
- 冷热分离：前 100 名走 Redis，100 名后查 DB（用户也不关心第 500 名）

### 📝 面试速答

> 「排行榜天然匹配 ZSet，所有操作 O(log n)。同分排序用 score 编码技巧；千万级用分片或只维护 Top N。」

---

## 5. 计数器

### 场景
文章阅读数、视频点赞数、API 调用次数。

### 实现

```
INCR article:123:views       ← 单线程保证原子
INCRBY article:123:views 10  ← 批量加
HINCRBY article:123 likes 1  ← Hash 里计数 (一个 key 多个计数字段)
```

**为什么不用 MySQL？** MySQL 的 `UPDATE ... SET count = count + 1` 在高并发下行锁冲突，QPS 撑不住；Redis 单线程无锁，轻松 10 万 QPS。

### 典型架构

```
用户点赞 → Redis INCR (瞬时写)
              ↓
          定时任务每分钟
              ↓
          批量落库 MySQL (持久化 + 可查询)
```

### ⚠️ 坑

- **Redis 挂了丢数据**：计数值是内存数据，没持久化前挂了就没了。办法：AOF `appendfsync everysec`（最多丢 1 秒）
- **重复计数**：同一个用户 1 秒点赞 100 次，业务要去重。用 Set 存「已点赞用户」，`SADD article:123:likers user_id`，`SCARD` 得到去重计数

---

## 6. 分布式 Session

### 场景
多台 Web 服务器共享登录态。

### 实现

```
SET session:<sessionId> <userJson> EX 1800
```

或用 Hash 存分散字段（修改单字段不用反序列化整个对象）：

```
HSET session:abc userId 1001 name "张三" role admin
HGETALL session:abc
EXPIRE session:abc 1800
```

### 选 String 还是 Hash？

| | String + JSON | Hash |
|---|---|---|
| 读整个对象 | ✅ 一次 GET | ❌ HGETALL |
| 改单个字段 | ❌ 反序列化 → 改 → 序列化 → SET | ✅ 一次 HSET |
| 内存占用 | JSON 有字段名开销 | listpack 下更紧凑 |
| TTL 粒度 | 整个对象一个 TTL | Hash 只能整个设 TTL，不能单字段 |

**经验**：字段多、频繁改单字段 → Hash；整体读写为主、字段少 → String + JSON。

---

## 7. 消息队列 / 延时队列

### 轻量场景：List 做队列

```
LPUSH  mq:email "msg1"        ← 生产者从左 push
BRPOP  mq:email 0             ← 消费者从右 pop (阻塞)
```

**缺点**：
- 消费确认机制弱，消费者挂了消息可能丢
- 只有一对一，没有广播/消费组

### 延时队列：ZSet score = 执行时间戳

```
ZADD  delay_queue <执行时间戳> "task_data"
  ↓ 每秒轮询一次
ZRANGEBYSCORE delay_queue 0 <now> LIMIT 0 10
  → 拿到到期任务, 处理完 ZREM
```

**坑**：轮询有延迟，可以改成事件驱动（keyspace notification）。

### 重量场景：Stream（Redis 5.0+）

Stream 是 Redis 自带的 Kafka-like 消息队列：

```
XADD  stream:orders * userId 1001 amount 99.9
XREADGROUP GROUP g1 consumer1 COUNT 10 STREAMS stream:orders >
XACK  stream:orders g1 <message_id>
```

特性：
- **消息持久化** + 消费者组 + 消费确认
- **Pending List**：消费了没 ACK 的消息记着，消费者重启后能重新消费
- 仍然是单机 Redis 的能力，**不能替代 Kafka**（吞吐、分区、副本都差一截）

### 📝 面试速答

> 「轻量用 List（BRPOP）；延时任务用 ZSet score=时间戳；要消费确认和消费组用 Stream。但要真高可用消息队列还是上 Kafka/RocketMQ，Redis 的消息能力是辅助。」

---

## 8. 去重 / 共同好友 / 标签

### 8.1 Set 做去重

```
SADD  article:123:readers user1001
SCARD article:123:readers         ← 独立阅读数 (UV)
```

### 8.2 Set 集合运算：共同好友 / 共同关注

```
SINTER  user:A:friends user:B:friends    ← 共同好友
SUNION  user:A:friends user:B:friends    ← 并集
SDIFF   user:A:friends user:B:friends    ← A 有 B 没有
```

### 8.3 超大去重 → HyperLogLog

Set 存 1 亿 UV 要几 GB，如果只要**近似值**（误差 0.81%），用 HLL 只要 **12KB**：

```
PFADD  uv:20260420 user1001
PFADD  uv:20260420 user1002
PFCOUNT uv:20260420                ← 12KB 存下任意基数
PFMERGE uv:week uv:20260414 uv:20260415 ... ← 合并一周的 UV
```

**权衡**：精确去重用 Set / Bitmap；近似大基数用 HLL。

### 8.4 Bitmap：签到、活跃用户

```
SETBIT  user:1001:signin:202604 20 1    ← 4月20号签到
GETBIT  user:1001:signin:202604 20      ← 查是否签到
BITCOUNT user:1001:signin:202604        ← 本月签到天数
BITOP AND dst user:1001:signin user:1002:signin   ← 两个用户的共同签到日
```

**为什么用 Bitmap？** 1 个用户 1 个月签到状态只要 30 bit ≈ 4 字节，1 亿用户也才 400MB。

### 📝 面试速答

> 「**精确去重**用 Set；**集合运算**用 SINTER/SUNION/SDIFF；**超大近似去重**用 HyperLogLog（12KB 搞定任意基数）；**布尔状态统计**（签到、在线）用 Bitmap。根据「精度需求 × 数据规模」选。」

---

## 9. 场景选型总速查

| 场景 | 首选类型 | 备选 |
|---|---|---|
| 对象缓存 | String + JSON | Hash（字段多） |
| 分布式锁 | String + SET NX | Redisson（带看门狗） |
| 限流 | String INCR / ZSet / Lua 令牌桶 | — |
| 排行榜 | **ZSet** | — |
| 计数器 | String INCR / Hash HINCRBY | — |
| 分布式 Session | String + JSON / Hash | — |
| 消息队列 | List / Stream | Kafka（真消息中间件） |
| 延时队列 | **ZSet** (score=时间戳) | Stream + Pending |
| 去重 | Set | HyperLogLog（大基数近似） |
| 集合运算 | Set + SINTER/SUNION | — |
| 签到/在线 | Bitmap (SETBIT) | — |
| 超大 UV | **HyperLogLog** | — |
| 地理位置 | GEO (底层 ZSet) | — |

---

## 10. 面试官最爱追问的 5 个点

1. **「你用 Redis 做过什么？」** → 别只说「缓存」，至少讲 2~3 个场景（缓存 + 锁 + 计数器）
2. **「缓存一致性怎么保证？」** → Cache-Aside 模式、延迟双删、Canal 订阅 binlog
3. **「怎么防止缓存穿透/击穿/雪崩？」** → 上面第 1 节
4. **「分布式锁的坑？」** → 锁过期业务没完 + 主从切换丢锁
5. **「为什么选 Redis 不选 Memcached？」** → 数据结构丰富 + 持久化 + 主从复制 + Lua 脚本 + 集群

---

## 11. 一句话终极总结

> **Redis 是「高性能内存数据结构服务器」，不只是缓存。选对类型 + 理解底层编码 + 避开坑（大 key、热 key、锁过期、缓存三座大山），它能抗下绝大多数场景。**
