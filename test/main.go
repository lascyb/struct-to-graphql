package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	graphql "github.com/lascyb/struct-to-graphql"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

type Union1 struct {
	Union1Field1 string `json:"union1Field1" graphql:"union1Field1"`
}
type Union2 struct {
	Union2Field1 string `json:"union2Field1" graphql:"union1Field1"`
}
type Union struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	Union1
	Union2
	Union3 Fragment
}
type Fragment struct {
	FragmentField1 string `json:"fragmentField1" graphql:"fragmentField1"`
	FragmentField2 string `json:"fragmentField2_alias1" graphql:"fragmentField1,alias=fragmentField2_alias1"`
}
type LineItem struct {
	LineItemField1 string `json:"lineItemField1" graphql:"lineItemField1"`
}
type LineItemConnect struct {
	Nodes []LineItem `json:"nodes" graphql:"nodes"`
}
type Inline struct {
	InlineField1 string `json:"inlineField1" graphql:"inlineField1"`
	InlineField2 string `json:"inline_alias2" graphql:"inlineField2,alias=inline_alias2"`
}
type Inline2 struct {
	Inline2Field2 string `json:"inlineField2" graphql:"inlineField2"`
}
type Tree struct {
	Tree1 Tree1 `json:"tree1" graphql:"tree1"`
}
type Tree1 struct {
	Union1Field1 Union  `json:"union1Field1" graphql:"union1Field1"`
	Tree1Field1  string `json:"tree1Field1" graphql:"tree1Field1"`
	Tree1Field2  Tree2  `json:"tree1Field2" graphql:"tree1Field2"`
	Inline1      Inline `json:"inline2" graphql:"inline1,alias=inline2"`
	//Inline2 Inline `json:"inline2" graphql:"inline2"`
	Inline
}
type Tree2 struct {
	Tree2Field1 string          `json:"tree2Field1" graphql:"tree2Field1"`
	LineItem    LineItemConnect `json:"lineItem" graphql:"lineItem(first:10,query:$:String!)"`
}
type TestStruct struct {
	Field1     string          `json:"field1" graphql:"field1"`
	Fragment1  Fragment        `json:"fragment1" graphql:"fragment1"`
	Fragment2  Fragment        `json:"fragment2" graphql:"fragment2"`
	UnionField Union           `json:"unionField" graphql:"unionField"`
	LineItem   LineItemConnect `json:"lineItem_alias1" graphql:"lineItem(first:10,query:$:String!,id:$id:Int!),alias=lineItem_alias1"`
	Inline
	Inline2         Inline2 `json:"inline2" graphql:"inline2,inline"`
	anonymityField1 string
	TreeField       Tree `json:"tree" graphql:"tree"`
	Test
	Anonymous1 struct {
		Field1 string `json:"alias2" graphql:"field1,alias=alias2"`
	} `json:"anonymity1" graphql:"anonymity1"`
	Fragment `json:"alias1" graphql:"anonymity2,inline,alias=alias1"`
}
type Test struct {
	anonymityField1 string
}
type Query struct {
	Field1 string `graphql:"field1"`
	// 支持参数占位符，$ 表示匿名占位符，自动生成变量名
	List struct {
		Nodes []struct {
			Name string `graphql:"name"`
		} `graphql:"nodes"`
	} `graphql:"list(first:10,query:$,id:$id:Int!)"`
}

func main() {
	//exec, err := graphql.Marshal(Query{})
	exec, err := graphql.Marshal(TestStruct{})
	if err != nil {
		panic(err)
	}
	fmt.Println(exec.Body)
	for _, fragment := range exec.Fragments {
		fmt.Println(fragment.Body)
	}
	for _, variable := range exec.Variables {
		fmt.Println(variable.Name, "|", variable.Type, "|", strings.Join(variable.Paths, ","))
	}
}
