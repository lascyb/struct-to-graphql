package graphql

import (
	"errors"
	"maps"
	"reflect"
	"slices"
)

// Graphql GraphQL 查询结构
type Graphql struct {
	Body      string      // GraphQL 查询主体内容
	Variables []*Variable // 层次化变量统计数组（按路径组织）
	Fragments []*Fragment // 复用结构模块数组
}

// Variable GraphQL 变量
type Variable struct {
	Name string   // 变量名（如 "$nodes_fieldName_first"）
	Path []string // 变量路径（如 "nodes.edges.node.first"）
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
		Variables: builder.Variables,
		Fragments: slices.Collect(maps.Values(builder.fragmentMap)),
	}, nil
}
