# 2026-04-20 复盘

## 今日题目

1. **CtCI 1.2 判定字符串是否互为字符重排** —— [question.md](./question.md) / [question.go](./question.go)
2. **LeetCode 567 字符串的排列** —— [leetcode567/question.md](./leetcode567/question.md) / [leetcode567/solution.go](./leetcode567/solution.go)

两题本质一样：**判断字符多重集是否相等**。区别只在于比较对象——整串 vs 滑动窗口。

## 今日八股

按域分类在 `bagu/` 下（Go / Redis / MySQL / OS / Network / Kafka）：

### Go
- **Slice 底层原理与扩容** —— [bagu/go/slice.md](./bagu/go/slice.md) / [bagu/go/slice_demo.go](./bagu/go/slice_demo.go)
  - SliceHeader（ptr/len/cap, 24B）
  - Go 1.18+ 新扩容规则（阈值 256，< 2x，> 渐进 1.25x，+ size class 对齐）
  - append 共享底层数组的陷阱 + `s[low:high:max]` 修复
  - nil vs empty slice、传参行为、内存泄漏场景

### Redis
- **五大数据结构 + 底层编码** —— [bagu/redis/data-structures.md](./bagu/redis/data-structures.md)
  - SDS（对比 Go `[]byte`）、String 三种编码（int / embstr / raw）、embstr 为什么 44
  - listpack 如何替代 ziplist（消除级联更新）
  - quicklist 链表 + listpack 节点的设计权衡
  - Hash 的渐进式 rehash、扩缩容阈值、负载因子
  - ZSet = 跳表 + 字典，跳表 vs 红黑树 vs B+ 树
  - 过期 key 清理、8 种淘汰策略、近似 LRU/LFU
  - redisObject 通用结构（type/encoding/lru/refcount/ptr）+ 共享对象池
- **8 大经典使用场景 + 面试陷阱** —— [bagu/redis/use-cases.md](./bagu/redis/use-cases.md)
  - 缓存（含三座大山：穿透 / 击穿 / 雪崩）
  - 分布式锁（SET NX PX + Lua 释放 + 看门狗 + RedLock）
  - 限流（固定窗口 / 滑动窗口 ZSet / 令牌桶 Lua）
  - 排行榜（ZSet 王牌场景 + 同分排序技巧）
  - 计数器（INCR + 批量落库）
  - 分布式 Session（String vs Hash 选型）
  - 消息队列（List / ZSet 延时 / Stream）
  - 去重（Set / HyperLogLog / Bitmap）


## 解法对比速查

| 方法 | 时间 | 空间 | 适用场景 | 备注 |
|---|---|---|---|---|
| 排序 `[]byte` 比较 | `O(n log n)` | `O(n)` | **仅 ASCII 正确**；UTF-8 下同样有假阳性 | 和字节计数犯同样的错 |
| 排序 `[]rune` 比较 | `O(n log n)` | `O(n)` | 任意 Unicode | 正确但慢 |
| `[128]int` 计数 | `O(n)` | `O(1)` | 纯 ASCII | byte > 127 会越界 |
| `[256]int` 计数 | `O(n)` | `O(1)` | **仅 ASCII 正确**；UTF-8 下有假阳性 | 见下文反例 |
| **`map[rune]int` 计数** | **`O(n)`** | **`O(k)`** | **任意 Unicode，推荐默认** | 常数大、有 GC |
| 滑动窗口 + `[26]int` | `O(n)` | `O(1)` | LC567 / LC438 子串匹配 | 数组可直接 `==` |
| 滑动窗口 + matches 计数器 | `O(n)` 严格 | `O(1)` | 同上，进阶版 | 避免每次 26 次比较 |


## 核心面试考点

### 1. 剪枝先行
长度不等直接 `return false`，别等到计数阶段才发现。`O(1)` 的判断，省大量无效工作。

### 2. 「一正一负」单次循环
长度相等时，用同一个计数数组，`s1[i]++, s2[i]--`，最后判断是否全 0。比「两个 map 分别计数再比」更快、更省。

### 3. 数组 vs map 的真实差距
- `[256]int` = 2KB，栈上分配，cache 友好，索引 `O(1)` 一次寻址。
- `map[byte]int` = hmap + bmap + 溢出桶，单 kv 实际占几十字节，操作要 hash + 比较 + 可能 rehash，**慢一个数量级**，还给 GC 加压。
- **只有字符集稀疏且巨大时（比如 Unicode 全集 `0~0x10FFFF`）才该用 map。**

### 4. 定长数组可以 `==` 比较
```go
var a, b [26]int
if a == b { ... }   // ✅ 值类型, 逐元素比较
```
切片不行（切片不可比较，必须手写循环或 `reflect.DeepEqual`）。

### 5. ⚠️ 字节计数在 UTF-8 下有假阳性（重要修正）
我一开始声称「byte 多重集相等 ⇔ rune 多重集相等」，**这是错的**。UTF-8 的前缀码性质只保证**有序字节流**能唯一解码回 rune 流，但打乱成**多重集**后这个唯一性就丢了。

**反例**（见 [counterexample/main.go](./counterexample/main.go)）：
```
一 U+4E00 = E4 B8 80
你 U+4F60 = E4 BD A0
丠 U+4E20 = E4 B8 A0     <- 交换"一""你"的末字节得到
佀 U+4F40 = E4 BD 80     <-

"一你" bytes = E4 B8 80 · E4 BD A0
"丠佀" bytes = E4 B8 A0 · E4 BD 80
```
字节多重集完全相等，但 4 个都是合法 CJK 汉字，rune 多重集明显不同。`[256]int` 算法会**误判为 true**。

**结论**：
- **纯 ASCII** → `[128]/[256]int` 按 byte 正确。
- **UTF-8 / 任意 Unicode** → 必须按 **rune** 计数，不能偷懒用 byte。

正确的 Unicode 版：
```go
func isAnagramRune(s1, s2 string) bool {
    r1, r2 := []rune(s1), []rune(s2)
    if len(r1) != len(r2) {
        return false
    }
    cnt := make(map[rune]int, len(r1))
    for i := 0; i < len(r1); i++ {
        cnt[r1[i]]++
        cnt[r2[i]]--
    }
    for _, v := range cnt {
        if v != 0 {
            return false
        }
    }
    return true
}
```

---

## Go 字符串详解

### 1. 底层结构

```go
type stringHeader struct {
    Data unsafe.Pointer  // 指向底层字节数组(只读)
    Len  int             // 字节长度
}
```

- string 本质就是**只读的 `[]byte`** + 长度，没有 cap 字段。
- **不可变**：任何「修改」都是分配新 string。`s[0] = 'x'` 编译不过。
- 字符串常量在**只读段**（rodata），多个同值常量共享地址——但 Go 不做运行时 intern。

### 2. 三种「长度」和「索引」方式

| 操作 | 返回 | 粒度 |
|---|---|---|
| `len(s)` | `int` | **字节数** |
| `s[i]` | `byte` (`uint8`) | 第 i 个字节 |
| `for i, r := range s` | `(int, rune)` | UTF-8 解码，`i` 是字节下标，`r` 是 rune |
| `[]rune(s)` 后索引 | `rune` (`int32`) | Unicode 码点 |
| `utf8.RuneCountInString(s)` | `int` | 字符数（rune 数） |

**例子**：

```go
s := "你好a"           // UTF-8: e4 bd a0  e5 a5 bd  61
len(s)                  // 7  (不是 3!)
s[0]                    // 228 (0xE4), "你" 的第一个字节
for i, r := range s {
    fmt.Println(i, string(r))
    // 0 你
    // 3 好
    // 6 a
}
utf8.RuneCountInString(s) // 3
```

### 3. 常见陷阱

**陷阱 A：反转字符串**

```go
// ❌ 错误: 中文会乱码
func reverseWrong(s string) string {
    b := []byte(s)
    for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
        b[i], b[j] = b[j], b[i]
    }
    return string(b)
}

// ✅ 正确: 按 rune 反转
func reverse(s string) string {
    r := []rune(s)
    for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
        r[i], r[j] = r[j], r[i]
    }
    return string(r)
}
```

**陷阱 B：取第 k 个字符**

```go
s := "你好世界"
s[1]              // ❌ 返回一个字节, 不是"好"
[]rune(s)[1]      // ✅ '好'
```

**陷阱 C：统计字符数**

```go
len("你好")                       // 6 字节
utf8.RuneCountInString("你好")    // 2 字符
len([]rune("你好"))               // 2, 但会分配新 slice
```

### 4. 类型转换的真实开销

| 转换 | 开销 | 说明 |
|---|---|---|
| `string(b []byte)` | **拷贝** | 分配新底层数组 |
| `[]byte(s string)` | **拷贝** | 同上，string 只读不能直接共享 |
| `string(r []rune)` | 拷贝 + UTF-8 编码 | |
| `[]rune(s string)` | 拷贝 + UTF-8 解码 | 最贵 |
| `string(b)` 仅比较 `m[string(b)]` | **零拷贝**（编译器优化） | Go 对 map key 的特殊优化 |

**性能要点**：在热路径上反复 `[]byte(s)` / `string(b)` 会是性能杀手。如果你需要频繁字节级操作，一开始就用 `[]byte`，只在必须的时候转 string。

### 5. 拼接的正确姿势

```go
// ❌ O(n^2) —— 每次 + 都分配新 string, 复制所有已有内容
s := ""
for _, x := range parts {
    s += x
}

// ✅ O(n) —— 用 strings.Builder, 内部是 []byte 扩容
var b strings.Builder
for _, x := range parts {
    b.WriteString(x)
}
s := b.String()   // String() 是零拷贝(Builder 结束不再写)

// ✅ 已知所有片段: strings.Join 最清爽
s := strings.Join(parts, ",")

// ✅ bytes.Buffer 类似 strings.Builder, 但 String() 会拷贝一次
```

`strings.Builder` 和 `bytes.Buffer` 的核心区别：**Builder 的 `String()` 零拷贝**（因为承诺之后不再写入），`Buffer.String()` 每次拷贝。写 string 场景优先 Builder。

### 6. 字符串比较

- `==` / `<` / `>`：按**字典序**比较字节，`O(min(len(a), len(b)))`。
- `strings.EqualFold(a, b)`：大小写不敏感相等。
- 想做 Unicode-aware 排序（比如德语 ß、中文笔顺）？标准库 `golang.org/x/text/collate`。

### 7. 常用标准库清单

| 包 | 典型用途 |
|---|---|
| `strings` | `Contains`, `Index`, `Split`, `Join`, `Replace`, `ToLower`, `Builder` |
| `bytes` | `[]byte` 版本的 strings，API 几乎同名 |
| `strconv` | `Itoa`, `Atoi`, `ParseInt`, `FormatFloat` —— 数字 ↔ string |
| `unicode` | `IsLetter`, `IsDigit`, `IsSpace`, `ToLower`（rune 级） |
| `unicode/utf8` | `RuneCountInString`, `DecodeRuneInString`, `ValidString` |
| `fmt` | `Sprintf` —— 方便但比 Builder 慢 ~5-10x |
| `regexp` | 正则, RE2 引擎, 无回溯 |

### 8. 面试一句话总结

> **Go 的 string 是只读 `[]byte` 头。`len` 返字节，`s[i]` 取字节，`range` 迭代 rune。做 ASCII 题按 byte 算最快；做 Unicode 题明确你要的是字节还是码点，再选索引方式。性能敏感场景避免无谓的 `string(b)` / `[]byte(s)` 转换，拼接用 `strings.Builder`。**


## 字符集 vs 编码：ASCII / Unicode / UTF-8

一个经常被混淆的问题：**ASCII、Unicode、UTF-8 到底是什么关系？**

一句话：**ASCII 是字符集，Unicode 是字符集，UTF-8 是 Unicode 的一种编码方式。**

- **字符集 (character set)**：给每个「字符」分配一个整数编号（叫 *code point* / 码点）。它只管「字符 ↔ 数字」的映射，不管怎么存。
- **编码 (encoding)**：把码点序列化成字节序列。它只管「数字 ↔ 字节」。

所以：**Unicode 是「数字表」，UTF-8 是「怎么把数字写成字节」。**

### 1. ASCII（1963）

- 7-bit 编码，共 128 个字符，编号 `0x00 ~ 0x7F`。
- 只覆盖英文字母、数字、基本标点、控制字符。
- 每个字符 **固定 1 字节**（最高位始终是 0）。
- 在 ASCII 世界里，「字符集」和「编码」合二为一——数字就是字节。

```
'A' = 0x41 = 0100 0001
'0' = 0x30
' ' = 0x20
```

### 2. Extended ASCII / ISO-8859 系列（~1980s）

- 8-bit，128 ~ 255 用来加一些「本地语言」字符（西欧重音、希腊字母、希伯来字母…）。
- 问题：**同一个字节值在不同地区代表不同字符**。`0xA3` 在 ISO-8859-1 是 `£`，在 ISO-8859-5 是西里尔字母。跨区域乱码的根源。
- 中文也有自己的一套：GB2312、GBK、GB18030；日文 Shift-JIS；港台 Big5……互相不兼容。

### 3. Unicode（1991 起，持续更新）

- 目标：给**全世界所有字符**一个**统一的**编号。
- 码点范围：`U+0000 ~ U+10FFFF`（理论上 1,114,112 个位置，约 110 万）。
- 已分配约 15 万字符，覆盖 150+ 种文字 + emoji + 数学符号 + 历史文字。
- **Unicode 本身只是「码点表」**，它不告诉你「一个码点用几个字节存」。
- 码点记法：`U+4F60` = 「你」，`U+1F600` = 😀。

```
U+0041 = 'A'           (65)
U+4F60 = '你'          (20320)
U+1F600 = '😀'         (128512, 超出基本平面 BMP)
```

### 4. UTF-8 / UTF-16 / UTF-32（Unicode 的三种编码）

| 编码 | 单位 | 每字符字节数 | 兼容 ASCII | 主要使用场景 |
|---|---|---|---|---|
| UTF-32 | 4 字节 | **固定 4** | ❌ | 内存里需要 O(1) 按下标访问码点时 |
| UTF-16 | 2 字节 | 2 或 4（代理对） | ❌ | Windows API, Java `char`, JavaScript string |
| **UTF-8** | 1 字节 | **1~4 变长** | ✅ | **Web / Linux / Go / 几乎所有新系统** |

**UTF-8 的编码规则**（变长，1~4 字节）：

| 码点范围 | 字节模式 |
|---|---|
| `U+0000 ~ U+007F`   (ASCII) | `0xxxxxxx` |
| `U+0080 ~ U+07FF`            | `110xxxxx 10xxxxxx` |
| `U+0800 ~ U+FFFF`   (中文常用区) | `1110xxxx 10xxxxxx 10xxxxxx` |
| `U+10000 ~ U+10FFFF` (emoji) | `11110xxx 10xxxxxx 10xxxxxx 10xxxxxx` |

几个重要性质：
1. **ASCII 完全兼容**：ASCII 字符在 UTF-8 里还是 1 字节，值完全一样。纯英文文本 ASCII 和 UTF-8 字节级别完全相同。
2. **自同步 (self-synchronizing)**：看任意一个字节的高位，就能判断它是「首字节」(`0xxx` 或 `11xx`) 还是「续字节」(`10xx`)。从文件中间开始读也能快速对齐到下一个字符边界。
3. **前缀码 (prefix code)**：没有任何字符的编码是另一个字符编码的前缀。这保证了**字节多重集相等 ⇔ 码点多重集相等**（我们之前用 `[256]int` 算 anagram 就靠这个性质）。
4. **字典序兼容**：按字节比较 UTF-8 字符串，结果和按码点比较一致。所以 Go 的 `s1 < s2` 虽然比的是字节，但在 Unicode 码点序上也是对的。

**举例：「你」`U+4F60`**

```
码点: 0x4F60 = 0100 1111 0110 0000  (二进制)

套 U+0800~U+FFFF 模板: 1110xxxx 10xxxxxx 10xxxxxx
取 4 + 6 + 6 = 16 位码点填进去:
  1110 0100  10 111101  10 100000
= E4         BD          A0
```

所以 `"你"` 在 UTF-8 下是 3 字节：`E4 BD A0`。这也解释了为什么 `len("你")` 返回 3。

### 5. 对应到 Go

- **Go 源码文件默认 UTF-8 编码**，规范要求。
- `string` 的底层字节是 UTF-8（只是「惯例上」，Go 运行时并不强制校验，你塞非法字节也能存）。
- `byte` = `uint8`，对应 UTF-8 的一个字节。
- `rune` = `int32`，对应一个 Unicode 码点。用 `int32` 是因为 `U+10FFFF` 需要 21 位，装不进 `int16`。
- `unicode/utf8` 包提供 UTF-8 编解码工具：`utf8.RuneCountInString`, `utf8.DecodeRuneInString`, `utf8.ValidString`。

### 6. 一张图记住全部

```
┌──────────────────────────────────────────────────────────┐
│                    字符 (character)                       │
│                        ↓                                  │
│         字符集给编号  (character set)                      │
│                        ↓                                  │
│            码点 code point (U+XXXX)                       │
│                        ↓                                  │
│        编码给字节表示  (encoding)                          │
│                        ↓                                  │
│              字节序列 (bytes on disk / wire)              │
└──────────────────────────────────────────────────────────┘

ASCII       : 字符集 + 编码合二为一 (128 个字符, 1 字节)
Unicode     : 只是字符集  (给 110 万个位置编号)
UTF-8       : Unicode 的一种编码 (变长 1~4 字节, 兼容 ASCII)
UTF-16      : Unicode 的一种编码 (2 或 4 字节)
UTF-32      : Unicode 的一种编码 (固定 4 字节)
GBK / Big5  : 中文专用字符集+编码, 不兼容 Unicode
```

### 7. 面试常见追问

**Q：为什么现代系统都选 UTF-8 而不是 UTF-16/UTF-32？**
- ASCII 兼容，老系统和协议无缝过渡。
- 空间效率：英文文本和 UTF-16 比省一半空间，和 UTF-32 比省 75%。
- 自同步、无字节序 (BOM) 问题：UTF-16/32 有大端小端之分，UTF-8 没有。
- Web / Linux / JSON / HTTP 全部默认 UTF-8，事实标准。

**Q：Java 的 `String` / `char` 是什么编码？**
- Java 内部用 **UTF-16**。`char` = 2 字节。
- 问题：一个 emoji（如 😀，`U+1F600`）在 Java 里是 **2 个 `char`**（代理对 surrogate pair），`"😀".length() == 2`。这就是为什么 `charAt` 对 emoji 经常返回乱码。

**Q：为什么中文在 UTF-8 里要 3 字节，在 GBK 里只要 2 字节？**
- GBK 只覆盖中文 + ASCII，编码空间小，能挤进 2 字节。
- UTF-8 要容纳 110 万码点，中文排在 `U+4E00~U+9FFF` 区段，套进三字节模板。
- **空间 vs 通用性的权衡**。现在带宽和存储不是瓶颈，通用性赢了。

**Q：「字符」、「码点」、「字形 (grapheme)」一样吗？**
- 不一样。比如 `é` 可以是：
  - 1 个码点 `U+00E9`（预组合），或
  - 2 个码点 `U+0065 + U+0301`（`e` + 组合重音），**视觉上看起来一样**。
- 「用户感知的一个字符」叫 grapheme cluster，可能对应多个 rune。做文本长度显示、光标移动要按 grapheme 而不是 rune。Go 标准库没有，需要 `golang.org/x/text/unicode/norm` 或第三方库。
- emoji 组合尤其坑：👨‍👩‍👧‍👦 = 7 个码点 + 3 个零宽连接符 (ZWJ) = 10 rune，但用户看到是 1 个字符。

### 8. 一句话总结

> **ASCII 是「古代英语小字典」；Unicode 是「全人类大字典」；UTF-8 是「把 Unicode 写成字节流的一种高效方式」。Go 全程用 UTF-8，`byte` 是字节，`rune` 是 Unicode 码点，搞清楚你在哪一层操作，就不会乱码。**


## Follow-up 回顾

1. **流式大文件 anagram**：两文件同步流式读，维护 `[256]int` 差分，`s1` 字节 `+1`，`s2` 字节 `-1`，读完判全 0。空间 `O(1)`。记得先比两个文件大小，不等秒拒。
2. **LC567 字符串排列子串**：滑动窗口 + 定长计数数组，进阶用 matches 计数器做到严格 `O(n)`。
3. **LC438 找到所有异位词**：LC567 的全量收集版，模板同上。


## 下次预告

- CtCI 1.3 URL 化（字符替换 + 原地操作）
- 或者切入并发方向：生产者消费者、限流器、`sync.Pool`
