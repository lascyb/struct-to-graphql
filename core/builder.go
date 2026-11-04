package core

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

type Builder struct {
	FragmentMap   map[reflect.Type]*Fragment // Fragment 映射，用于去重
	VariableMap   map[string]*Variable       // 变量映射，用于去重
	currentPaths  []string
	fragmentOrder []reflect.Type // Fragment 首次使用的顺序，保证输出稳定
	variableOrder []string       // 变量首次出现的顺序，保证输出稳定
}

// Fragment GraphQL Fragment
type Fragment struct {
	Name string // Fragment 名称（如 "UserInfo"）
	Type string // Fragment 类型（如 "User"）
	Body string // Fragment 完整定义（如 "fragment UserInfo on User { ... }"）
}

// Variable GraphQL 变量
type Variable struct {
	Name         string      // 变量名（如 "$nodes_fieldName_first"）
	Paths        []string    // 变量路径（如 "nodes.edges.node.first"）
	Type         string      // 变量类型（如 Int、Int!、String、String!）
	HasDefault   bool        // 是否有默认值
	DefaultValue interface{} // 默认值，用于变量定义中的 " = value"
}

func NewBuilder() *Builder {
	return &Builder{
		FragmentMap:  make(map[reflect.Type]*Fragment),
		VariableMap:  make(map[string]*Variable),
		currentPaths: []string{},
	}
}

// Variables 按首次出现的顺序返回所有变量
func (g *Builder) Variables() []*Variable {
	result := make([]*Variable, 0, len(g.variableOrder))
	for _, name := range g.variableOrder {
		if v, ok := g.VariableMap[name]; ok {
			result = append(result, v)
		}
	}
	return result
}

// Fragments 按首次使用的顺序返回所有 Fragment
func (g *Builder) Fragments() []*Fragment {
	result := make([]*Fragment, 0, len(g.fragmentOrder))
	for _, typ := range g.fragmentOrder {
		if f, ok := g.FragmentMap[typ]; ok {
			result = append(result, f)
		}
	}
	return result
}

func (g *Builder) Build(typeParser *TypeParser) (string, error) {
	if typeParser != nil {
		return g.buildSelectionSet(typeParser, false, typeParser.Union, 0)
	}
	return "", fmt.Errorf("struct to parse cannot be nil")
}

// buildSelectionSet 递归生成 GraphQL 类型定义字符串
// 根据类型解析器构建 GraphQL 查询语法，支持联合类型、内联字段和嵌套结构
// typeParser: 类型解析器，包含字段列表、联合类型标识和重用次数等信息
// inlineType: 是否为内联类型，true 表示该类型是匿名字段或标记为 inline 的字段，字段名会被省略
// isUnionSubType: 是否为联合类型的子类型，true 表示当前正在处理联合类型的某个具体类型分支
// level: 缩进层级，用于格式化输出，0 表示顶级，每递归一层递增
// path: 当前字段路径，用于参数变量名生成
// 返回: GraphQL 类型定义字符串，格式如 "{ field1 { nestedField } field2 }" 或 "... on TypeName { field }"
func (g *Builder) buildSelectionSet(typeParser *TypeParser, inlineType, isUnionSubType bool, level uint) (string, error) {
	if typeParser == nil {
		return "", nil
	}
	if typeParser.Reused > 1 && typeParser.source.Name() != "" {
		if fragment, ok := g.FragmentMap[typeParser.source]; ok {
			if inlineType && !isUnionSubType {
				return fmt.Sprintf("\n%s...%s", indentWithLevel(level+1), fragment.Name), nil
			}
			return fmt.Sprintf("{ ...%s }", fragment.Name), nil
		}
		if !inlineType {
			level = 0
		}
		if !inlineType || isUnionSubType {
			// 在开始构建时登记顺序，使外层 Fragment 排在其内部引用的 Fragment 之前（按首次使用顺序）
			g.fragmentOrder = append(g.fragmentOrder, typeParser.source)
		}
	}
	buf := new(strings.Builder)

	// 非内联类型或联合子类型需要添加花括号包裹字段
	if !inlineType || isUnionSubType {
		buf.WriteString("{")
	}
	// 遍历所有字段，递归构建 GraphQL 查询字符串
	currentPathsCount := len(g.currentPaths)
	defer func() {
		// 使用 defer 确保路径栈始终被恢复，即使在异常情况下也不会导致状态污染
		g.currentPaths = g.currentPaths[:currentPathsCount]
	}()

	for _, field := range typeParser.Fields {
		g.currentPaths = append(g.currentPaths[:currentPathsCount], field.FieldName)
		// 处理联合类型：使用 GraphQL 的 inline fragment 语法 "... on TypeName"
		if typeParser.Union {
			buf.WriteString("\n")
			buf.WriteString(indentWithLevel(level + 1))
			// __typename 字段直接输出，用于类型判断
			if field.FieldName == "__typename" {
				buf.WriteString(field.FieldName)
			} else if field.TypeParser != nil {
				// 联合类型的其他字段使用 "... on TypeName" 语法
				buf.WriteString("... on ")
				if field.TypeName == "" {
					return "", fmt.Errorf("anonymous struct types are not supported for field [%s] in union types", field.FieldName)
				}
				buf.WriteString(field.TypeName)
				// inline fragment 上的指令：... on Type @include(if:$x) { ... }
				directives, err := g.buildDirectives(field)
				if err != nil {
					return "", err
				}
				buf.WriteString(directives)
				buf.WriteString(" ")
				// 递归构建子类型，标记为联合子类型以保持花括号
				set, err := g.buildSelectionSet(field.TypeParser, field.Inline, true, level+1)
				if err != nil {
					return "", fmt.Errorf("failed to build type for field [%s]: %w", field.FieldName, err)
				}
				buf.WriteString(set)
			} else {
				return "", fmt.Errorf("field [%s] in union type [%s] should be a struct type", field.FieldName, typeParser.source.String())
			}
			continue
		}
		// 处理匿名嵌入字段：直接展开字段内容，不添加字段名
		if field.Inline {
			if len(field.Directives) > 0 {
				return "", fmt.Errorf("directives are not supported on embedded (flattened) field [%s]; put them on a named field or a union branch", field.FieldName)
			}
			set, err := g.buildSelectionSet(field.TypeParser, true, false, level)
			if err != nil {
				return "", fmt.Errorf("failed to build type for field [%s]: %w", field.FieldName, err)
			}
			buf.WriteString(set)
		} else {
			// 处理普通字段：添加字段名和适当缩进
			buf.WriteString("\n")
			buf.WriteString(indentWithLevel(level + 1))
			buf.WriteString(field.FieldName)
			// 构建字段参数
			args, err := g.buildFieldArgs(field)
			if err != nil {
				return "", err
			}
			buf.WriteString(args)
			// 构建字段指令（位于参数之后、子选择集之前）
			directives, err := g.buildDirectives(field)
			if err != nil {
				return "", err
			}
			buf.WriteString(directives)
			// 递归构建嵌套类型，层级递增
			set, err := g.buildSelectionSet(field.TypeParser, false, false, level+1)
			if err != nil {
				return "", fmt.Errorf("failed to build type for field [%s]: %w", field.FieldName, err)
			}
			buf.WriteString(set)
		}
	}
	// 闭合花括号，与开头的花括号对应
	if !inlineType || isUnionSubType {
		buf.WriteString("\n")
		buf.WriteString(indentWithLevel(level))
		buf.WriteString("}")

		// 处理重用类型：当类型被多次引用且不是顶级类型时，应封装为 Fragment（匿名结构体无法生成 Fragment，跳过）
		if typeParser.Reused > 1 && typeParser.source.Name() != "" {
			// 生成 Fragment 名称时，使用单独的 result 切片来避免修改原始 split 导致的索引混乱
			split := strings.Split(typeParser.source.String(), ".")
			var result []string
			for _, s := range split {
				if s == "" {
					continue
				}
				// 首字母大写
				runes := []rune(s)
				if len(runes) > 0 {
					runes[0] = unicode.ToUpper(runes[0])
				}
				result = append(result, string(runes))
			}
			fragmentName := strings.Join(result, "")
			fragmentType := typeParser.source.Name()
			fragment := fmt.Sprintf("fragment %s on %s%s", fragmentName, fragmentType, buf.String())
			g.FragmentMap[typeParser.source] = &Fragment{
				Name: strings.ReplaceAll(fragmentName, ".", "_"),
				Type: fragmentType,
				Body: fragment,
			}
			return fmt.Sprintf("{ ...%s }", fragmentName), nil
		}
	}
	return buf.String(), nil
}

// buildFieldArgs 构建字段参数字符串，返回形如 "(a: 1, b: $x)" 的片段
func (g *Builder) buildFieldArgs(field *FieldParser) (string, error) {
	if field == nil || field.TagValue == nil || len(field.TagValue.Args) == 0 {
		return "", nil
	}
	return g.buildArgs(field.TagValue.Args, field.TagValue.ArgKeys, "")
}

// buildArgs 按 keys 顺序构建参数列表字符串，返回形如 "(a:1,b:$x)" 的片段
// varPrefix: 匿名变量名的参数前缀（指令参数使用 "指令名_"，避免与字段参数重名）
func (g *Builder) buildArgs(args map[string]*Arg, keys []string, varPrefix string) (string, error) {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, err := g.buildArgumentValue(varPrefix+key, args[key])
		if err != nil {
			return "", err
		}
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%s", key, value))
	}

	if len(parts) == 0 {
		return "", nil
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ",")), nil
}

// buildDirectives 构建字段指令字符串，返回形如 " @include(if:$show) @skip(if:false)" 的片段
func (g *Builder) buildDirectives(field *FieldParser) (string, error) {
	if field == nil || len(field.Directives) == 0 {
		return "", nil
	}
	buf := new(strings.Builder)
	for _, d := range field.Directives {
		args, err := g.buildArgs(d.Args, d.ArgKeys, d.Name+"_")
		if err != nil {
			return "", fmt.Errorf("directive @%s on field [%s]: %w", d.Name, field.FieldName, err)
		}
		buf.WriteString(" @")
		buf.WriteString(d.Name)
		buf.WriteString(args)
	}
	return buf.String(), nil
}

func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// buildArgumentValue 根据占位符与自定义名生成参数值字符串
func (g *Builder) buildArgumentValue(key string, arg *Arg) (string, error) {
	if arg == nil {
		return "", nil
	}

	if arg.ArgValue.Type == "variable" {
		varName := arg.VarName
		if varName == "" {
			varName = CamelToSnake(strings.ReplaceAll(strings.Join(g.currentPaths, "_")+"_"+key, ":", "_"))
		}
		if variable, ok := g.VariableMap[varName]; ok {
			if variable.Type != arg.GraphQLType {
				return "", fmt.Errorf("变量 %s 类型不统一：[%s]<==>[%s]", varName, variable.Type, arg.GraphQLType)
			}
			g.VariableMap[varName].Paths = append(g.VariableMap[varName].Paths, strings.Join(g.currentPaths, "/"))
		} else {
			g.variableOrder = append(g.variableOrder, varName)
			g.VariableMap[varName] = &Variable{
				Name:         "$" + varName,
				Paths:        []string{strings.Join(g.currentPaths, "/")},
				Type:         arg.GraphQLType,
				HasDefault:   arg.HasDefault,
				DefaultValue: arg.DefaultVal,
			}
		}
		return g.VariableMap[varName].Name, nil
	}
	return formatLiteralArgValue(arg.Value), nil
}

// formatLiteralArgValue 将字面量参数格式化为 GraphQL 查询中的写法（字符串加引号等）
func formatLiteralArgValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}
