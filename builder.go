package graphql

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/lascyb/tagkit"
)

type Builder struct {
	fragmentMap  map[reflect.Type]*Fragment // Fragment 映射，用于去重
	variableMap  map[string]uint            // 变量映射，用于去重
	Variables    []*Variable
	currentPaths []string
}

func NewBuilder() *Builder {
	return &Builder{
		fragmentMap:  make(map[reflect.Type]*Fragment),
		variableMap:  make(map[string]uint),
		currentPaths: []string{},
	}
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
	if typeParser.Reused > 1 {
		if fragment, ok := g.fragmentMap[typeParser.source]; ok {
			if inlineType {
				return fmt.Sprintf("\n%s...%s", indentWithLevel(level+1), fragment.Name), nil
			}
			return fmt.Sprintf("{ ...%s }", fragment.Name), nil
		}
		if !inlineType {
			level = 0
		}
	}
	buf := new(strings.Builder)

	// 非内联类型或联合子类型需要添加花括号包裹字段
	if !inlineType || isUnionSubType {
		buf.WriteString("{")
	}
	// 遍历所有字段，递归构建 GraphQL 查询字符串
	currentPathsCount := len(g.currentPaths)
	for _, field := range typeParser.Fields {
		if len(g.currentPaths) > currentPathsCount {
			g.currentPaths = append(g.currentPaths[:len(g.currentPaths)-1], field.FieldName)
		} else {
			g.currentPaths = append(g.currentPaths, field.FieldName)
		}
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
				if field.TypeParser.source.Name() == "" {
					return "", fmt.Errorf("anonymous struct types are not supported for field [%s] in union types", field.FieldName)
				}
				buf.WriteString(field.TypeParser.source.Name())
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
		// 处理内联字段：直接展开字段内容，不添加字段名
		if field.Inline {
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
			buf.WriteString(g.buildFieldArgs(field))
			// 递归构建嵌套类型，层级递增
			set, err := g.buildSelectionSet(field.TypeParser, false, false, level+1)
			if err != nil {
				return "", fmt.Errorf("failed to build type for field [%s]: %w", field.FieldName, err)
			}
			buf.WriteString(set)
		}
	}
	if len(g.currentPaths) > currentPathsCount {
		g.currentPaths = g.currentPaths[:len(g.currentPaths)-1]
	}
	// 闭合花括号，与开头的花括号对应
	if !inlineType || isUnionSubType {
		buf.WriteString("\n")
		buf.WriteString(indentWithLevel(level))
		buf.WriteString("}")

		// 处理重用类型：当类型被多次引用且不是顶级类型时，应封装为 Fragment
		if typeParser.Reused > 1 {
			split := strings.Split(typeParser.source.String(), ".")
			for i, s := range split {
				runes := []rune(s)
				runes = append([]rune{unicode.ToUpper(rune(s[0]))}, runes[1:]...)
				split[i] = string(runes)
			}
			fragmentName := strings.Join(split, "")
			fragmentType := typeParser.source.Name()
			fragment := fmt.Sprintf("fragment %s on %s%s", fragmentName, fragmentType, buf.String())
			g.fragmentMap[typeParser.source] = &Fragment{
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
func (g *Builder) buildFieldArgs(field *FieldParser) string {
	if field == nil || field.TagValue == nil || len(field.TagValue.Args) == 0 {
		return ""
	}

	parts := make([]string, 0, len(field.TagValue.Args))
	for name, arg := range field.TagValue.Args {
		value := g.buildArgumentValue(arg)
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", name, value))
	}

	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
}

// buildArgumentValue 根据占位符与自定义名生成参数值字符串
func (g *Builder) buildArgumentValue(arg *tagkit.Arg) string {
	if arg == nil {
		return ""
	}

	if arg.Placeholder {
		var varName string

		if arg.CustomName != "" {
			// 自定义名称
			varName = arg.CustomName
			if _, ok := g.variableMap[varName]; !ok {
				g.Variables = append(g.Variables, &Variable{
					Name: "$" + varName,
					Path: g.currentPaths[:],
				})
			}
		} else {
			// 匿名占位符，生成变量名（会在内部标记到 variableMap）
			varName = strings.Join(g.currentPaths, "_") + "_" + arg.Name
			if index, ok := g.variableMap[varName]; ok {
				varName = fmt.Sprintf("%s_%d", varName, index+1)
				g.variableMap[varName]++
			} else {
				g.variableMap[varName] = 0
			}
			// 收集变量到 Variables 数组（generateAnonymousArgName 已标记到 variableMap，这里直接添加）
			g.Variables = append(g.Variables, &Variable{
				Name: "$" + varName,
				Path: g.currentPaths[:],
			})
		}

		return "$" + varName
	}
	return arg.Value
}
