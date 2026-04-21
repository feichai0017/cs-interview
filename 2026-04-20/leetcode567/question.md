## LeetCode 567 - 字符串的排列

**问题描述**

给你两个字符串 `s1` 和 `s2`，写一个函数来判断 `s2` 是否包含 `s1` 的**排列**。换句话说，`s1` 的某个排列是 `s2` 的**子串**。

```
Input:  s1 = "ab", s2 = "eidbaooo"
Output: true     (s2 包含 "ba"，是 "ab" 的排列)

Input:  s1 = "ab", s2 = "eidboaoo"
Output: false

Input:  s1 = "adc", s2 = "dcda"
Output: true     (s2 包含 "dcd"... 等等不对，应该是 "dca" 或 "cda")
                 实际: s2[1:4] = "cda", 是 "adc" 的排列
```

约束：`s1`、`s2` 由小写英文字母组成，`1 <= len(s1), len(s2) <= 10^4`。


**解题思路**

题目的本质是：**在 `s2` 里找一个长度为 `len(s1)` 的子串，使其字符多重集 == `s1` 的字符多重集**。

这就是 anagram 判定的「滑动窗口版」。

1. **剪枝**：如果 `len(s1) > len(s2)`，直接 `false`。
2. 用 `need [26]int` 统计 `s1` 的字符频次。
3. 用 `have [26]int` 维护 `s2` 上一个长度为 `len(s1)` 的滑动窗口：
   - 窗口每向右滑一格：右端新进字符 `have[c]++`，左端移出字符 `have[c]--`。
   - 每次滑动后判断 `have == need`（Go 里数组可以直接用 `==` 比较）。
4. 找到匹配就 `true`；全程没匹配就 `false`。

**朴素版复杂度**

- 时间：`O((n - m + 1) * 26)`，窗口数 × 每次数组比较的 26 次。可以看作 `O(n)`，`n = len(s2)`。
- 空间：`O(1)`，两个 `[26]int` 固定大小。

**优化版：维护 matches 计数器**

朴素版每次窗口滑动都要比较两个数组（26 次循环）。可以进一步优化：
- 维护一个 `matches` 变量，记录**当前有多少种字符 `have[c] == need[c]`**。
- 每次改变 `have[c]` 前后，分别判断它是否「刚好匹配」或「刚好脱离匹配」，`O(1)` 更新 `matches`。
- 当 `matches == 26` 时就全匹配了。

时间变为严格 `O(n)`，常数更小。这是字节/FAANG 面试里想听到的「pro 版」。


**源码参考**

朴素版：

```go
func checkInclusion(s1, s2 string) bool {
    m, n := len(s1), len(s2)
    if m > n {
        return false
    }
    var need, have [26]int
    for i := 0; i < m; i++ {
        need[s1[i]-'a']++
        have[s2[i]-'a']++
    }
    if need == have {
        return true
    }
    for i := m; i < n; i++ {
        have[s2[i]-'a']++
        have[s2[i-m]-'a']--
        if need == have {
            return true
        }
    }
    return false
}
```

优化版（matches 计数器）：

```go
func checkInclusionOpt(s1, s2 string) bool {
    m, n := len(s1), len(s2)
    if m > n {
        return false
    }
    var need, have [26]int
    for i := 0; i < m; i++ {
        need[s1[i]-'a']++
    }

    matches := 0
    // 初始窗口
    for i := 0; i < m; i++ {
        c := s2[i] - 'a'
        have[c]++
        if have[c] == need[c] {
            matches++
        } else if have[c] == need[c]+1 {
            // 之前相等, 现在超了, 脱离匹配
            matches--
        }
    }
    if matches == 26 {
        return true
    }

    for i := m; i < n; i++ {
        // 右进
        r := s2[i] - 'a'
        have[r]++
        if have[r] == need[r] {
            matches++
        } else if have[r] == need[r]+1 {
            matches--
        }
        // 左出
        l := s2[i-m] - 'a'
        have[l]--
        if have[l] == need[l] {
            matches++
        } else if have[l] == need[l]-1 {
            matches--
        }

        if matches == 26 {
            return true
        }
    }
    return false
}
```


**源码解析**

- `need == have`：Go 里**固定长度数组**是值类型，`==` 会逐元素比较，所以可以直接用。切片不行（切片不可比较）。
- `s2[i]-'a'`：因为题目限定小写字母，直接映射到 `[0, 25]`。如果是任意 ASCII，换成 `[256]int` 并用 `s2[i]` 索引即可。
- 滑动窗口的「同时进出」：一次循环里先 `++` 再 `--`（或反过来），保持窗口大小始终 = `m`。注意下标：新进的是 `s2[i]`，移出的是 `s2[i-m]`。
- matches 计数器的核心逻辑：
  - **刚从 `need-1` 变成 `need`** → `matches++`（多一种字符达标）
  - **刚从 `need` 变成 `need+1` 或 `need-1`** → `matches--`（少一种字符达标）
  - 这样每次滑动只做 `O(1)` 更新。


**和 anagram 的关系**

- **Anagram（CtCI 1.2）**：整个串 vs 整个串。长度相等 + 多重集相等。
- **LC567**：固定窗口 vs 子串。`len(s1) == 窗口长度` + 多重集相等。
- **LC438 找所有异位词**：LC567 的「全量版」，不是找一个，而是把所有匹配起点都收集起来，模板完全一样。


**扩展**

- 如果字符集换成 Unicode？滑动窗口仍成立，但不能用 `[26]int`，得用 `map[rune]int`，或者按 byte 搞 `[256]int`（因为 UTF-8 的前缀码性质，byte 多重集 ⇔ rune 多重集，在**定长窗口**里依然等价）。
- 如果是「`s2` 包含 `s1` 的**子序列**」而不是「排列的子串」？那就是完全不同的题（子序列 DP 或贪心双指针），别搞混。
