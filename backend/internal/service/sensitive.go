package service

import (
	"strings"
	"unicode"
)

// sensitiveNode trie 节点（spec §5.5：手写 trie/DFA，避免正则回溯）。
type sensitiveNode struct {
	children map[rune]*sensitiveNode
	end      bool
}

// SensitiveFilter 敏感词过滤器。词表与待检文本使用同一套归一化，
// 对抗「全角、大小写、中间插空格/符号」类变形（如「傻 X」「傻.逼」）。
type SensitiveFilter struct {
	root *sensitiveNode
}

// NewSensitiveFilter 从换行分隔的词表构建 trie；坏行自动跳过。
func NewSensitiveFilter(wordsText string) *SensitiveFilter {
	f := &SensitiveFilter{root: &sensitiveNode{children: map[rune]*sensitiveNode{}}}
	for _, w := range strings.Split(wordsText, "\n") {
		w = normalizeSensitive(w)
		if w == "" {
			continue
		}
		f.add(w)
	}
	return f
}

func (f *SensitiveFilter) add(word string) {
	node := f.root
	for _, r := range word {
		child, ok := node.children[r]
		if !ok {
			child = &sensitiveNode{children: map[rune]*sensitiveNode{}}
			node.children[r] = child
		}
		node = child
	}
	node.end = true
}

// Hit 归一化后逐位做 trie 前缀匹配，命中任一词即返回 true。
func (f *SensitiveFilter) Hit(text string) bool {
	runes := []rune(normalizeSensitive(text))
	for i := range runes {
		node := f.root
		for j := i; j < len(runes); j++ {
			child, ok := node.children[runes[j]]
			if !ok {
				break
			}
			node = child
			if node.end {
				return true
			}
		}
	}
	return false
}

// normalizeSensitive：全角→半角、大写→小写、剔除空白/标点/符号/Cf 格式字符
// （匹配维度上等效于去变形插入；Cf 如软连字符/词连接符渲染不可见但会拆散相邻字符）。
func normalizeSensitive(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0x3000: // 全角空格
			r = ' '
		case r >= 0xFF01 && r <= 0xFF5E: // 全角 ASCII 区
			r -= 0xFEE0
		}
		r = unicode.ToLower(r)
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
