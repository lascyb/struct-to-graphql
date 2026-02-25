package graphql

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/lascyb/struct-to-graphql/core"
)

// Graphql GraphQL 查询结构
type Graphql struct {
	Body      string           // GraphQL 查询主体内容
	Variables []*core.Variable // 层次化变量统计数组（按路径组织）
	Fragments []*core.Fragment // 复用结构模块数组
}

func Marshal(v any) (*Graphql, error) {
	if v == nil {
		return nil, errors.New("struct to parse cannot be nil")
	}
	parser, err := core.NewParser().ParseType(reflect.TypeOf(v))
	if err != nil {
		return nil, err
	}

	builder := core.NewBuilder()
	body, err := builder.Build(parser)
	if err != nil {
		return nil, err
	}

	return &Graphql{
		Body:      body,
		Variables: slices.Collect(maps.Values(builder.VariableMap)),
		Fragments: slices.Collect(maps.Values(builder.FragmentMap)),
	}, nil
}
func (g *Graphql) build(operation, name string) (string, error) {
	if g == nil {
		return "", errors.New("graphql cannot be nil")
	}

	var parts []string

	// 添加所有 Fragments
	for _, fragment := range g.Fragments {
		parts = append(parts, fragment.Body)
	}

	// 构建变量定义部分
	varDefs := make([]string, 0, len(g.Variables))
	for _, v := range g.Variables {
		if v.Type == "" {
			return "", fmt.Errorf("变量 %s 缺少类型定义", v.Name)
		}
		def := fmt.Sprintf("%s:%s", v.Name, v.Type)
		if v.HasDefault {
			def += "=" + formatVariableDefault(v.DefaultValue)
		}
		varDefs = append(varDefs, def)
	}

	// 构建操作声明
	if name != "" {
		operation += " " + name
	}
	if len(varDefs) > 0 {
		operation += "(" + strings.Join(varDefs, ",") + ")"
	}

	// 组合查询体
	queryBody := fmt.Sprintf("%s %s", operation, g.Body)

	parts = append(parts, queryBody)

	return strings.Join(parts, "\n"), nil
}

// Query 组装完整的 GraphQL 查询字符串
// name: 查询名称，如 "GetUser"、"ListItems" 等
// 返回: 完整的 GraphQL 查询字符串，包含操作声明、变量定义、查询体和 Fragments
func (g *Graphql) Query(name string) (string, error) {
	return g.build("query", name)
}

// Mutation 组装完整的 GraphQL 突变字符串
// name: 突变名称，如 "productUpdate"、"productVariantsBulkUpdate" 等
// 返回: 完整的 GraphQL 突变字符串，包含操作声明、变量定义、查询体和 Fragments
func (g *Graphql) Mutation(name string) (string, error) {
	return g.build("mutation", name)
}

// formatVariableDefault 将变量默认值格式化为 GraphQL 变量定义中的写法（如 "value"、123、[1,2,3]）
func formatVariableDefault(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, e := range val {
			parts = append(parts, formatVariableDefault(e))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprint(v)
	}
}
func SetIndent(val string) {
	core.SetIndent(val)
}
