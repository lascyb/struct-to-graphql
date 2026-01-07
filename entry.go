package graphql

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// Graphql GraphQL 查询结构
type Graphql struct {
	Body      string      // GraphQL 查询主体内容
	Variables []*Variable // 层次化变量统计数组（按路径组织）
	Fragments []*Fragment // 复用结构模块数组
}

// Variable GraphQL 变量
type Variable struct {
	Name  string   // 变量名（如 "$nodes_fieldName_first"）
	Paths []string // 变量路径（如 "nodes.edges.node.first"）
	Type  string   // 变量类型（如 Int、Int!、String、String!）
}

// Fragment GraphQL Fragment
type Fragment struct {
	Name string // Fragment 名称（如 "UserInfo"）
	Type string // Fragment 类型（如 "User"）
	Body string // Fragment 完整定义（如 "fragment UserInfo on User { ... }"）
}

func Marshal(v any) (*Graphql, error) {
	if v == nil {
		return nil, errors.New("struct to parse cannot be nil")
	}
	parser, err := NewParser().ParseType(reflect.TypeOf(v))
	if err != nil {
		return nil, err
	}

	builder := NewBuilder()
	body, err := builder.Build(parser)
	if err != nil {
		return nil, err
	}

	return &Graphql{
		Body:      body,
		Variables: slices.Collect(maps.Values(builder.variableMap)),
		Fragments: slices.Collect(maps.Values(builder.fragmentMap)),
	}, nil
}
func (g *Graphql) build(operation, name string) (string, error) {
	if g == nil {
		return "", errors.New("Graphql cannot be nil")
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
		varDefs = append(varDefs, fmt.Sprintf("%s: %s", v.Name, v.Type))
	}

	// 构建操作声明
	if name != "" {
		operation += " " + name
	}
	if len(varDefs) > 0 {
		operation += "(" + strings.Join(varDefs, ", ") + ")"
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
