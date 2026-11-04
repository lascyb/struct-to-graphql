package core

import (
	"fmt"
	"strings"

	"github.com/lascyb/tagkit"
)

// 本文件负责把 tagkit 解析出的字面量（tagkit.Literal，区分字符串 / 枚举 / 数字 / 布尔 / null / 列表）
// 转换为 GraphQL 写法。

// quoteGraphQLString 将任意字符串编码为合法的 GraphQL 字符串字面量
func quoteGraphQLString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// renderLiteral 输出 tagkit 字面量的 GraphQL 写法：带引号的字符串转义为 GraphQL 字符串，裸标识符原样输出（枚举值），
// 数字 / 布尔 / null 使用 tagkit 保留的原文，列表逐元素按相同规则输出。
// quoteBare 为 true 时裸标识符按字符串输出（用于 String/ID 类型变量的默认值，兼容旧写法）。
func renderLiteral(l *tagkit.Literal, quoteBare bool) string {
	switch l.Kind {
	case tagkit.KindString:
		s, _ := l.Value.(string)
		return quoteGraphQLString(s)
	case tagkit.KindEnum:
		s, _ := l.Value.(string)
		if quoteBare {
			return quoteGraphQLString(s)
		}
		return s
	case tagkit.KindList:
		parts := make([]string, 0, len(l.Elems))
		for _, e := range l.Elems {
			parts = append(parts, renderLiteral(e, quoteBare))
		}
		return "[" + strings.Join(parts, ",") + "]"
	default: // number / bool / null
		return l.Text
	}
}

// isStringLikeType 判断变量类型的基础类型是否为 String 或 ID（忽略列表与非空修饰）
func isStringLikeType(t string) bool {
	base := strings.Trim(t, "[]!")
	return base == "String" || base == "ID"
}

// FormatValue 将 Go 值格式化为 GraphQL 字面量：字符串输出为转义后的 GraphQL 字符串，
// nil 输出 null，[]interface{} 逐元素格式化为列表，其余（数字、布尔）按 fmt 默认格式输出。
// 用于手动构造 Variable（未设置 Default）时生成默认值写法。
func FormatValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case string:
		return quoteGraphQLString(val)
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, e := range val {
			parts = append(parts, FormatValue(e))
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return fmt.Sprint(v)
	}
}
