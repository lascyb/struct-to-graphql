package graphql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/lascyb/struct-to-graphql/core"
)

// Variable 生成结果中的变量定义（core.Variable 的别名，便于直接引用）
type Variable = core.Variable

// Fragment 生成结果中的 Fragment 定义（core.Fragment 的别名，便于直接引用）
type Fragment = core.Fragment

// Graphql GraphQL 查询结构
type Graphql struct {
	Body      string           // GraphQL 查询主体内容
	Variables []*core.Variable // 变量数组，按首次出现的顺序排列
	Fragments []*core.Fragment // 复用结构模块数组，按首次使用的顺序排列
}

// Option Marshal 的可选参数（core.Option 的别名），由 WithFragment、WithIndent 创建
type Option = core.Option

// WithFragment 控制是否把被多次引用的命名结构体抽取为顶层 fragment，默认 true。
// 传 false 时这些结构体在每个引用处原地展开，Graphql.Fragments 为空；union 分支仍输出 "... on Type { ... }"。
func WithFragment(enable bool) Option {
	return core.WithFragment(enable)
}

// WithIndent 设置本次 Marshal 的单级缩进，优先于全局 SetIndent，且不修改全局设置
func WithIndent(val string) Option {
	return core.WithIndent(val)
}

// Marshal 把带 graphql tag 的结构体解析为 GraphQL 选择集、变量与 Fragment。
// opts 为可选参数（WithFragment、WithIndent），nil 会被忽略；同一选项传多次时以最后一次为准。
func Marshal(v any, opts ...Option) (*Graphql, error) {
	if v == nil {
		return nil, errors.New("struct to parse cannot be nil")
	}
	typ := reflect.TypeOf(v)
	parser, err := core.NewParser().ParseType(typ)
	if err != nil {
		return nil, err
	}
	if parser == nil {
		return nil, fmt.Errorf("graphql: %s has no queryable fields (no exported fields, or all fields are skipped)", typ)
	}

	builder := core.NewBuilder(opts...)
	body, err := builder.Build(parser)
	if err != nil {
		return nil, err
	}

	return &Graphql{
		Body:      body,
		Variables: builder.Variables(),
		Fragments: builder.Fragments(),
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
			if v.Default != "" {
				def += "=" + v.Default
			} else {
				def += "=" + core.FormatValue(v.DefaultValue)
			}
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

// Subscription 组装完整的 GraphQL 订阅字符串
// name: 订阅名称，如 "OnCommentAdded"、"OrderStatusChanged" 等
// 返回: 完整的 GraphQL 订阅字符串，包含操作声明、变量定义、查询体和 Fragments
func (g *Graphql) Subscription(name string) (string, error) {
	return g.build("subscription", name)
}

// SetIndent 设置生成结果的全局单级缩进（默认两个空格），并发安全；对之后开始、且未传 WithIndent 的 Marshal 调用生效
func SetIndent(val string) {
	core.SetIndent(val)
}
