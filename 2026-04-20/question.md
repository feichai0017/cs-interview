## 判定字符串是否互为字符重排

**问题描述**

给定两个字符串 `s1` 和 `s2`，请编写一个程序，确定其中一个字符串的字符重新排列后，能否变成另一个字符串。

例如：

```
s1 = "abc"   s2 = "bca"   => true
s1 = "abc"   s2 = "bad"   => false
s1 = "abc"   s2 = "abcd"  => false
```

约束：字符串的字符为 `ASCII` 字符，长度不做硬性限制。需要注意 `大小写敏感` 以及 `空白字符` 是否参与比较（此处按「全部字符都参与比较，大小写敏感」处理）。


**解题思路**

首先有一个最朴素的剪枝：如果两个字符串长度不同，直接返回 `false`。

然后有三种主流做法：

1. **排序后比较**：把两个字符串各自排序，再比较是否相等。时间复杂度 `O(n log n)`，实现最短。
2. **计数数组**：利用 `ASCII` 只有 128/256 个字符这个特性，开一个长度为 128 的计数数组，遍历 `s1` 时 `+1`，遍历 `s2` 时 `-1`，最后判断是否全为 0。时间复杂度 `O(n)`，空间复杂度 `O(1)`（常数）。
3. **map 计数**：通用性最好，可以处理 Unicode。时间 `O(n)`，空间 `O(k)`（k 为不同字符数）。

注意一个坑：Go 中 `range string` 按 `rune` 遍历（UTF-8 解码），`s[i]` 才是按 `byte` 遍历。对于纯 ASCII 场景两者结果一致，但如果题目扩展到 Unicode，必须用 `rune`，否则多字节字符会被当成多个 `byte` 计数导致出错。


**源码参考**

排序法：

```go
func isAnagramSort(s1, s2 string) bool {
    if len(s1) != len(s2) {
        return false
    }
    b1, b2 := []byte(s1), []byte(s2)
    sort.Slice(b1, func(i, j int) bool { return b1[i] < b1[j] })
    sort.Slice(b2, func(i, j int) bool { return b2[i] < b2[j] })
    return string(b1) == string(b2)
}
```

计数数组法（仅适用于 ASCII）：

```go
// 按 byte 统计。仅对 ASCII 正确。
// UTF-8 下会有假阳性, 见 counterexample/main.go
func isAnagramCount(s1, s2 string) bool {
    if len(s1) != len(s2) {
        return false
    }
    var cnt [256]int
    for i := 0; i < len(s1); i++ {
        cnt[s1[i]]++
        cnt[s2[i]]--
    }
    for _, v := range cnt {
        if v != 0 {
            return false
        }
    }
    return true
}
```

map 计数法（支持 Unicode）：

```go
func isAnagramMap(s1, s2 string) bool {
    r1, r2 := []rune(s1), []rune(s2)
    if len(r1) != len(r2) {
        return false
    }
    m := make(map[rune]int, len(r1))
    for _, r := range r1 {
        m[r]++
    }
    for _, r := range r2 {
        m[r]--
        if m[r] < 0 {
            return false
        }
    }
    return true
}
```


**源码解析**

- 排序法胜在简洁，但 `O(n log n)` 不是最优，且需要分配新的 `[]byte`。注意不能写成 `sort.Strings([]string{s1})`，字符串的 `sort` 要对底层 `[]byte` 排序。
- 计数数组法把两个字符串在同一次遍历中一正一负抵消，避免了两次遍历两个 map。对于 ASCII 限定的场景这是最优解。
- map 计数法在遇到 Unicode（如中文、emoji）时才需要，`len(string)` 返回的是 `byte` 长度，所以长度比较一定要用 `[]rune` 转换后的长度，否则 `"好"` 和 `"abc"` 会被误判为长度相等。
- 在 `map` 实现里用了 `if m[r] < 0 return false` 的早退出，比遍历结束后再判断要快一些。


**扩展思考**

1. 如果允许「多出任意数量空格」算互为重排，比如 `"dog cat"` 和 `"cat dog"` 是 true，应该怎么改？
2. 如果题目是「判定 `s2` 是否是 `s1` 的某个排列的子串」（LeetCode 567），又该怎么做？滑动窗口 + 计数数组。
3. 如果字符串巨长且是流式读入，无法一次加载到内存，如何做？可以维护一个增量的 `[128]int` 差分数组，读完两个流后比较。
