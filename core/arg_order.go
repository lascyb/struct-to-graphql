package core

import (
	"sort"

	tkcore "github.com/lascyb/tagkit/core"
)

// argKeyOrder 按 tag 原文中出现的顺序返回字段调用的参数名列表。
// tagkit 将参数保存在 map 中，遍历顺序随机；这里借助 tagkit 的词法分析器重新扫描原文，
// 仅收集第一层圆括号内、位于 "(" 或 "," 之后且紧跟 ":" 的标识符作为参数名，
// 字符串字面量（含引号内的括号、逗号）由词法分析器整体识别，不会被误判。
func argKeyOrder(raw string) []string {
	lexer := tkcore.NewLexer(raw)
	var (
		keys       []string
		parenDepth int
		brackDepth int
		prev       tkcore.Token
	)
	cur := lexer.NextToken()
	for cur.Type != tkcore.TokenEOF {
		next := lexer.NextToken()
		switch cur.Type {
		case tkcore.TokenLParen:
			parenDepth++
		case tkcore.TokenRParen:
			parenDepth--
		case tkcore.TokenLBracket:
			brackDepth++
		case tkcore.TokenRBracket:
			brackDepth--
		case tkcore.TokenIdent:
			if parenDepth == 1 && brackDepth == 0 && next.Type == tkcore.TokenColon &&
				(prev.Type == tkcore.TokenLParen || prev.Type == tkcore.TokenComma) {
				keys = append(keys, cur.Value)
			}
		}
		prev = cur
		cur = next
	}
	return keys
}

// orderedArgKeys 返回与 args 完全一致的参数名列表：优先按原文出现顺序，
// 未能从原文中定位的参数（理论上不会出现）按字典序追加，保证输出稳定。
func orderedArgKeys[V any](raw string, args map[string]V) []string {
	result := make([]string, 0, len(args))
	seen := make(map[string]bool, len(args))
	for _, k := range argKeyOrder(raw) {
		if _, ok := args[k]; ok && !seen[k] {
			seen[k] = true
			result = append(result, k)
		}
	}
	var rest []string
	for k := range args {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(result, rest...)
}
