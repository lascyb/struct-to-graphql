package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lascyb/tagkit"
	tkcore "github.com/lascyb/tagkit/core"
)

// tagkit 会把参数保存在 map 中（遍历顺序随机），并且不区分「带引号的字符串」与「裸标识符」，
// 还会丢弃列表字面量参数。本文件借助 tagkit 的词法分析器重新扫描 tag 原文，按与 tagkit 一致的语法
// 解析字段调用的参数，得到：参数书写顺序、每个字面量的 GraphQL 写法，以及变量默认值的 GraphQL 写法。

// parseTagValue 调用 tagkit.ParseTagValue，并把 tagkit 内部的 panic 转换为错误
func parseTagValue(raw string) (tv *tagkit.TagValue, err error) {
	defer func() {
		if r := recover(); r != nil {
			tv, err = nil, fmt.Errorf("invalid tag %q: %v", raw, r)
		}
	}()
	return tagkit.ParseTagValue(raw)
}

// valueKind 字面量的种类
type valueKind int

const (
	kindString valueKind = iota // 带引号的字符串：'abc' 或 "abc"
	kindEnum                    // 裸标识符：CREATED_AT、PUBLIC（GraphQL 枚举值）
	kindNumber                  // 数字：10、-1、1.5
	kindBool                    // true / false
	kindNull                    // null
	kindList                    // 列表：[1,2]、[A,B]
)

// scannedValue 扫描得到的字面量
type scannedValue struct {
	Kind  valueKind
	Text  string         // 原始文本；字符串为去除引号并处理转义后的内容
	Elems []scannedValue // Kind 为 kindList 时的元素
}

// scannedArg 扫描得到的单个参数
type scannedArg struct {
	Key        string
	IsVariable bool
	Value      *scannedValue // 字面量参数的值，变量参数为 nil
	VarType    string        // 变量声明的类型（如 Int!、[ID!]!）
	Default    *scannedValue // 变量默认值，未声明时为 nil
}

// argScanner 基于 tagkit 词法单元的递归下降扫描器，语法与 tagkit 的 parseFieldCall 保持一致
type argScanner struct {
	raw  string
	toks []tkcore.Token
	pos  int
}

func newArgScanner(raw string) (s *argScanner) {
	s = &argScanner{raw: raw}
	defer func() {
		// tagkit v1.0.6 的词法分析器在「以反斜杠结尾的未闭合字符串」上会 panic，此时按空输入处理，
		// 真正的语法错误由 parseTagValue 报告
		if recover() != nil {
			s.toks = []tkcore.Token{{Type: tkcore.TokenEOF}}
		}
	}()
	lexer := tkcore.NewLexer(raw)
	for {
		tok := lexer.NextToken()
		s.toks = append(s.toks, tok)
		if tok.Type == tkcore.TokenEOF {
			return s
		}
	}
}

func (s *argScanner) cur() tkcore.Token { return s.toks[s.pos] }

func (s *argScanner) next() {
	if s.pos < len(s.toks)-1 {
		s.pos++
	}
}

// scanCallArgs 按书写顺序返回 tag 原文中字段调用（第一个圆括号）内的参数。
// 遇到 tagkit 也会拒绝的非法写法时停止扫描并返回已识别的部分（此时 tagkit 会报告具体错误）。
func scanCallArgs(raw string) []scannedArg {
	s := newArgScanner(raw)
	for s.cur().Type != tkcore.TokenLParen {
		if s.cur().Type == tkcore.TokenEOF {
			return nil
		}
		s.next()
	}
	s.next()
	var result []scannedArg
	for {
		switch s.cur().Type {
		case tkcore.TokenRParen, tkcore.TokenEOF:
			return result
		case tkcore.TokenComma:
			s.next()
			continue
		case tkcore.TokenIdent:
		default:
			return result
		}
		key := s.cur().Value
		s.next()
		if s.cur().Type != tkcore.TokenColon {
			return result
		}
		s.next()
		arg, ok := s.scanArgValue()
		if !ok {
			return result
		}
		arg.Key = key
		result = append(result, arg)
	}
}

// scanArgValue 解析参数值：变量 $name:Type=default、字面量或列表
func (s *argScanner) scanArgValue() (scannedArg, bool) {
	if s.cur().Type != tkcore.TokenDollar {
		v, ok := s.scanLiteralOrList()
		return scannedArg{Value: v}, ok
	}
	s.next()
	arg := scannedArg{IsVariable: true}
	if s.cur().Type == tkcore.TokenIdent {
		s.next()
	}
	if s.cur().Type == tkcore.TokenColon {
		s.next()
		arg.VarType = s.scanType()
	}
	if s.cur().Type == tkcore.TokenEqual {
		s.next()
		v, ok := s.scanLiteralOrList()
		if !ok {
			return arg, false
		}
		arg.Default = v
	}
	return arg, true
}

// scanType 解析变量类型，规则与 tagkit 的 parseType 一致：Name、Name!、[Inner]、[Inner]!
func (s *argScanner) scanType() string {
	if s.cur().Type == tkcore.TokenLBracket {
		s.next()
		inner := s.scanType()
		// 内层类型已消费自身的 "!"，此处只可能是重复的 "!"（如 [Int!!]），与 tagkit 一致地忽略
		if s.cur().Type == tkcore.TokenBang {
			s.next()
		}
		if s.cur().Type != tkcore.TokenRBracket {
			return "Unknown"
		}
		s.next()
		t := "[" + inner + "]"
		if s.cur().Type == tkcore.TokenBang {
			t += "!"
			s.next()
		}
		return t
	}
	if s.cur().Type != tkcore.TokenIdent {
		return "Unknown"
	}
	t := s.cur().Value
	s.next()
	if s.cur().Type == tkcore.TokenBang {
		t += "!"
		s.next()
	}
	return t
}

// scanLiteralOrList 解析单个字面量或列表字面量
func (s *argScanner) scanLiteralOrList() (*scannedValue, bool) {
	tok := s.cur()
	switch tok.Type {
	case tkcore.TokenLBracket:
		s.next()
		list := &scannedValue{Kind: kindList, Elems: []scannedValue{}}
		for {
			switch s.cur().Type {
			case tkcore.TokenRBracket:
				s.next()
				return list, true
			case tkcore.TokenComma:
				s.next()
				continue
			case tkcore.TokenEOF:
				return nil, false
			}
			elem, ok := s.scanLiteralOrList()
			if !ok {
				return nil, false
			}
			list.Elems = append(list.Elems, *elem)
		}
	case tkcore.TokenString:
		s.next()
		return &scannedValue{Kind: kindString, Text: unescapeTagString(tok.Value)}, true
	case tkcore.TokenNumber:
		s.next()
		text := tok.Value
		// tagkit 的词法分析器会跳过无法识别的字符（包括负号），这里根据原文补回负号
		if tok.Pos > 0 && s.raw[tok.Pos-1] == '-' {
			text = "-" + text
		}
		return &scannedValue{Kind: kindNumber, Text: text}, true
	case tkcore.TokenBool:
		s.next()
		return &scannedValue{Kind: kindBool, Text: tok.Value}, true
	case tkcore.TokenIdent:
		s.next()
		if tok.Value == "null" {
			return &scannedValue{Kind: kindNull, Text: "null"}, true
		}
		return &scannedValue{Kind: kindEnum, Text: tok.Value}, true
	}
	return nil, false
}

// unescapeTagString 处理 tag 字符串字面量中的转义：\\ \' \" 还原为对应字符，其余反斜杠原样保留
func unescapeTagString(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\', '\'', '"':
				b.WriteByte(s[i+1])
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

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

// render 输出字面量的 GraphQL 写法。
// quoteBare 为 true 时裸标识符按字符串输出（用于 String/ID 类型变量的默认值，兼容旧写法）。
func (v *scannedValue) render(quoteBare bool) string {
	switch v.Kind {
	case kindString:
		return quoteGraphQLString(v.Text)
	case kindEnum:
		if quoteBare {
			return quoteGraphQLString(v.Text)
		}
		return v.Text
	case kindList:
		parts := make([]string, 0, len(v.Elems))
		for i := range v.Elems {
			parts = append(parts, v.Elems[i].render(quoteBare))
		}
		return "[" + strings.Join(parts, ",") + "]"
	default: // number / bool / null
		return v.Text
	}
}

// isStringLikeType 判断变量类型的基础类型是否为 String 或 ID（忽略列表与非空修饰）
func isStringLikeType(t string) bool {
	base := strings.Trim(t, "[]!")
	return base == "String" || base == "ID"
}

// argKeyOrder 按 tag 原文中出现的顺序返回字段调用的参数名列表
func argKeyOrder(raw string) []string {
	scanned := scanCallArgs(raw)
	keys := make([]string, 0, len(scanned))
	for _, a := range scanned {
		keys = append(keys, a.Key)
	}
	return keys
}

// orderedArgKeys 返回与 args 完全一致的参数名列表：优先按 order 中的顺序，
// 未能在 order 中定位的参数（理论上不会出现）按字典序追加，保证输出稳定。
func orderedArgKeys[V any](order []string, args map[string]V) []string {
	result := make([]string, 0, len(args))
	seen := make(map[string]bool, len(args))
	for _, k := range order {
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
