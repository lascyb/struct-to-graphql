package test_error

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== 常见错误用法 ==========

// ---------- union 分支必须是「匿名嵌入的命名结构体」 ----------

// 具名字段 + 匿名结构体（未嵌入）
type UnionNamedFieldAnonymousStruct struct {
	Typename string `graphql:"__typename,union"`
	Item     struct {
		Name string `graphql:"name"`
	}
}

// 即使设置了 type flag，具名字段 + 匿名结构体也不能作为分支
type UnionAnonymousStructWithType struct {
	Typename string `graphql:"__typename,union"`
	TextData struct {
		Text string `graphql:"text"`
	} `graphql:"textData,type=TextData"`
}

// 具名字段 + 命名结构体（未嵌入）
type UnionBranchText struct {
	Text string `graphql:"text"`
}
type UnionNamedFieldNamedStruct struct {
	Typename string          `graphql:"__typename,union"`
	Text     UnionBranchText `graphql:"text"`
}

// 嵌入的类型不是结构体
type UnionBranchScalar string
type UnionEmbeddedScalar struct {
	Typename string `graphql:"__typename,union"`
	UnionBranchScalar
}

// 嵌入的结构体没有可查询的字段
type UnionBranchEmpty struct {
	hidden string //nolint:unused
}
type UnionEmbeddedEmptyStruct struct {
	Typename string `graphql:"__typename,union"`
	UnionBranchEmpty
}

func wrapContent[T any]() any {
	return struct {
		Content T `graphql:"content"`
	}{}
}

// ---------- 其他错误 ----------

// 同名变量类型冲突
type VariableTypeConflict struct {
	Products []struct {
		ID string `graphql:"id"`
	} `graphql:"products(author:$author:Int!)"`
	Contents []struct {
		ID string `graphql:"id"`
	} `graphql:"contents(author:$author:String!)"`
}

// 嵌套字段 / 匿名嵌入 / union 分支内部出错时，错误信息带上外层字段路径
type NestedConflictInner struct {
	X string `graphql:"x(a:$v:String)"`
}
type NestedConflictEmbedded struct {
	Y string `graphql:"y(a:$v:String)"`
}
type NestedConflictBranch struct {
	Z string `graphql:"z(a:$v:String)"`
}
type NestedConflictUnion struct {
	Typename string `graphql:"__typename,union"`
	NestedConflictBranch
}
type NestedVariableConflict struct {
	Top   string              `graphql:"top(a:$v:Int)"`
	Outer NestedConflictInner `graphql:"outer"`
}
type EmbeddedVariableConflict struct {
	Top string `graphql:"top(a:$v:Int)"`
	NestedConflictEmbedded
}
type UnionBranchVariableConflict struct {
	Top     string              `graphql:"top(a:$v:Int)"`
	Content NestedConflictUnion `graphql:"content"`
}
type UnionDirectiveConflictBranch struct {
	ID string `graphql:"id"`
}
type UnionDirectiveConflictContent struct {
	Typename                     string `graphql:"__typename,union"`
	UnionDirectiveConflictBranch `graphql:",directive='include(if:$v)'"`
}
type UnionDirectiveVariableConflict struct {
	Top     string                        `graphql:"top(a:$v:Int)"`
	Content UnionDirectiveConflictContent `graphql:"content"`
}

// 字符串字面量为空（全空白）
type EmptyLiteralArgument struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(query:'  ')"`
}

// tag 语法错误（参数值缺失），由 tagkit 报告
type InvalidTagSyntax struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(query::String!)"`
}

// 以反斜杠结尾的未闭合字符串（tagkit 内部会 panic），应返回错误而不是 panic
type UnterminatedString struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(query:'\\"`
}

// 同一参数重复声明
type DuplicateArgument struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(first:1,first:2)"`
}

// 循环引用
type CycleNode struct {
	ID       string       `graphql:"id"`
	Children []*CycleNode `graphql:"children"`
}

type CycleA struct {
	B *CycleB `graphql:"b"`
}
type CycleB struct {
	A []CycleA `graphql:"a"`
}

// 没有可查询的字段
type NoExportedFields struct {
	id string //nolint:unused
}
type AllFieldsSkipped struct {
	ID   string `graphql:"-"`
	Name string `json:"-"`
}

func TestMarshalErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		v    any
		want []string
	}{
		{"UnionNamedFieldAnonymousStruct", wrapContent[UnionNamedFieldAnonymousStruct](), []string{"field [Item]", "should be an embedded struct field"}},
		{"UnionAnonymousStructWithType", wrapContent[UnionAnonymousStructWithType](), []string{"field [textData]", "should be an embedded struct field"}},
		{"UnionNamedFieldNamedStruct", wrapContent[UnionNamedFieldNamedStruct](), []string{"field [text]", "should be an embedded struct field"}},
		{"UnionEmbeddedScalar", wrapContent[UnionEmbeddedScalar](), []string{"field [UnionBranchScalar]", "should be a named struct type"}},
		{"UnionEmbeddedEmptyStruct", wrapContent[UnionEmbeddedEmptyStruct](), []string{"field [UnionBranchEmpty]", "should be a named struct type"}},
		{"UnionAtRoot", UnionNamedFieldAnonymousStruct{}, []string{"should be an embedded struct field"}},
		{"VariableTypeConflict", VariableTypeConflict{}, []string{"变量 author 类型不统一", "[Int!]<==>[String!]"}},
		{"NestedVariableConflict", NestedVariableConflict{}, []string{"failed to build type for field [outer]", "变量 v 类型不统一：[Int]<==>[String]"}},
		{"EmbeddedVariableConflict", EmbeddedVariableConflict{}, []string{"failed to build type for field [NestedConflictEmbedded]", "类型不统一"}},
		{"UnionBranchVariableConflict", UnionBranchVariableConflict{}, []string{"failed to build type for field [content]", "failed to build type for field [NestedConflictBranch]", "类型不统一"}},
		{"UnionDirectiveVariableConflict", UnionDirectiveVariableConflict{}, []string{"directive @include on field [UnionDirectiveConflictBranch]", "[Int]<==>[Boolean!]"}},
		{"EmptyLiteralArgument", EmptyLiteralArgument{}, []string{"field [Items]", "参数值不能定义为空"}},
		{"InvalidTagSyntax", InvalidTagSyntax{}, []string{"field [Items]", "unexpected token"}},
		{"UnterminatedString", UnterminatedString{}, []string{"field [Items]", "invalid tag"}},
		{"DuplicateArgument", DuplicateArgument{}, []string{"参数 first 重复声明"}},
		{"SelfReference", CycleNode{}, []string{"circular reference detected for type test_error.CycleNode"}},
		{"MutualReference", CycleA{}, []string{"circular reference detected for type test_error.CycleA"}},
		{"EmptyStruct", struct{}{}, []string{"struct {} has no queryable fields"}},
		{"NoExportedFields", NoExportedFields{}, []string{"test_error.NoExportedFields has no queryable fields"}},
		{"AllFieldsSkipped", &AllFieldsSkipped{}, []string{"test_error.AllFieldsSkipped has no queryable fields"}},
		{"NotStruct", 42, []string{"must be a struct type to convert, got int"}},
		{"MapNotStruct", map[string]int{}, []string{"got map[string]int"}},
		{"SliceOfNonStruct", []string{}, []string{"got string"}},
		{"Nil", nil, []string{"cannot be nil"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertMarshalError(t, c.v, c.want...)
		})
	}
}

// 变量未声明类型：Marshal 成功，组装完整操作时报错
type MissingVariableType struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(id:$id)"`
}

func TestOperationErrors(t *testing.T) {
	t.Parallel()
	exec, err := graphql.Marshal(MissingVariableType{})
	if err != nil {
		t.Fatalf("marshal should succeed, got: %v", err)
	}
	for name, build := range map[string]func(string) (string, error){
		"query": exec.Query, "mutation": exec.Mutation, "subscription": exec.Subscription,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := build("MissingType")
			testutil.AssertError(t, err, "变量 $id 缺少类型定义")
		})
	}

	var nilExec *graphql.Graphql
	for _, build := range []func(string) (string, error){nilExec.Query, nilExec.Mutation, nilExec.Subscription} {
		_, err := build("X")
		testutil.AssertError(t, err, "graphql cannot be nil")
	}
}
