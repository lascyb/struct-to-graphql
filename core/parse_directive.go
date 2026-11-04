package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lascyb/tagkit"
)

// directiveFlag 字段 tag 中声明指令的标记位名称，可重复出现，按书写顺序输出：
//
//	graphql:"field,directive='include(if:$show:Boolean!)',directive='skip(if:$hide)'"
const directiveFlag = "directive"

var directiveNameRe = regexp.MustCompile(`^[_A-Za-z][_0-9A-Za-z]*$`)

// Directive GraphQL 指令（如 @include(if:$show)）
type Directive struct {
	Name    string          // 指令名（不含 @），如 "include"
	Args    map[string]*Arg // 指令参数，与字段参数使用相同的占位符语法
	ArgKeys []string        // 参数名列表，按书写顺序排列
}

// parseDirectives 从 tag 标记位中解析所有 directive='...' 声明
// 指令值使用与字段调用相同的 tagkit 语法：name(arg:value, ...)，可选前导 @；
// 参数值支持字面量以及 $、$name、$:Type、$name:Type（可带默认值）等变量占位符。
func parseDirectives(tv *tagkit.TagValue) ([]*Directive, error) {
	var result []*Directive
	seen := make(map[string]bool)
	for _, flag := range tv.FlagsNamed(directiveFlag) {
		raw, ok := flag.Value.(string)
		if !flag.HasValue() || !ok {
			return nil, fmt.Errorf("directive 需要以字符串形式声明，如 directive='include(if:$show:Boolean!)'，实际为 %v", flag.Value)
		}
		directive, err := parseDirective(raw)
		if err != nil {
			return nil, err
		}
		// @include / @skip 在 GraphQL 规范中不可重复
		if (directive.Name == "include" || directive.Name == "skip") && seen[directive.Name] {
			return nil, fmt.Errorf("directive @%s 不能在同一字段上重复声明", directive.Name)
		}
		seen[directive.Name] = true
		result = append(result, directive)
	}
	return result, nil
}

// parseDirective 解析单个指令声明，如 "@include(if:$show:Boolean!)"、"cached(ttl:60)"、"deprecated"
func parseDirective(raw string) (*Directive, error) {
	src := strings.TrimSpace(raw)
	src = strings.TrimPrefix(src, "@")
	if src == "" {
		return nil, fmt.Errorf("directive 不能为空")
	}
	tv, err := tagkit.ParseTagValue(src)
	if err != nil {
		return nil, fmt.Errorf("directive [%s] 解析失败: %w", raw, err)
	}
	if len(tv.Flags) > 0 {
		return nil, fmt.Errorf("directive [%s] 只能包含一个指令调用，多个指令请分别使用 directive='...' 声明", raw)
	}
	if !directiveNameRe.MatchString(tv.Name) || !strings.HasPrefix(src, tv.Name) {
		return nil, fmt.Errorf("directive [%s] 缺少合法的指令名", raw)
	}
	args, keys, err := newArgs(tv)
	if err != nil {
		return nil, fmt.Errorf("directive [%s]: %w", raw, err)
	}
	d := &Directive{
		Name:    tv.Name,
		Args:    args,
		ArgKeys: keys,
	}
	if d.Name == "include" || d.Name == "skip" {
		if err := validateConditionalDirective(d, raw); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// validateConditionalDirective 校验 @include / @skip：有且仅有一个 if 参数，取值为布尔字面量或变量；
// 变量未声明类型时按规范推断为 Boolean!
func validateConditionalDirective(d *Directive, raw string) error {
	arg, ok := d.Args["if"]
	if !ok || len(d.Args) != 1 {
		return fmt.Errorf("directive [%s]: @%s 必须且只能包含参数 if", raw, d.Name)
	}
	switch arg.Type {
	case "variable":
		if arg.GraphQLType == "" {
			arg.GraphQLType = "Boolean!"
		}
	case "literal":
		if _, isBool := arg.Value.(bool); !isBool {
			return fmt.Errorf("directive [%s]: @%s 的 if 参数必须是布尔值或变量，实际为 %v", raw, d.Name, arg.Value)
		}
	}
	return nil
}
