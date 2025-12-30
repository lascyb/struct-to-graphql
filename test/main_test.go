package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

func TestMarshalGeneratesExpectedGraphql(t *testing.T) {
	exec, err := graphql.Marshal([]TestStruct{})
	if err != nil {
		t.Fatalf("Marshal 返回错误: %v", err)
	}

	if normalizeArgs(exec.Body) != normalizeArgs(expectedBody) {
		t.Fatalf("生成的 Body 不符合预期:\n实际:\n%s\n预期:\n%s", exec.Body, expectedBody)
	}

	gotFragments := make(map[string]string, len(exec.Fragments))
	for _, f := range exec.Fragments {
		gotFragments[f.Name] = f.Body
	}
	if !reflect.DeepEqual(expectedFragments, gotFragments) {
		t.Fatalf("Fragments 不符合预期:\n实际:%v\n预期:%v", gotFragments, expectedFragments)
	}

	expectedVars := []graphql.Variable{
		{Name: "$lineItem_alias1_lineItem_query", Paths: []string{"lineItem_alias1:lineItem"}, Type: "String!"},
		{Name: "$id", Paths: []string{"lineItem_alias1:lineItem"}, Type: "Int!"},
		{Name: "$tree_tree1_tree1Field2_lineItem_query", Paths: []string{"tree/tree1/tree1Field2/lineItem"}, Type: "String!"},
	}
	if len(exec.Variables) != len(expectedVars) {
		t.Fatalf("变量数量不符，实际 %d 预期 %d", len(exec.Variables), len(expectedVars))
	}
	// 按变量名排序以便比较
	gotVarsMap := make(map[string]*graphql.Variable)
	for _, v := range exec.Variables {
		if v == nil {
			t.Fatalf("变量为 nil")
		}
		gotVarsMap[v.Name] = v
	}
	expectedVarsMap := make(map[string]*graphql.Variable)
	for i := range expectedVars {
		expectedVarsMap[expectedVars[i].Name] = &expectedVars[i]
	}
	for name, expectedVar := range expectedVarsMap {
		gotVar, ok := gotVarsMap[name]
		if !ok {
			t.Fatalf("缺少变量 %s", name)
		}
		if gotVar.Name != expectedVar.Name || !reflect.DeepEqual(gotVar.Paths, expectedVar.Paths) || gotVar.Type != expectedVar.Type {
			t.Fatalf("变量 %s 不符合预期，实际 %+v 预期 %+v", name, gotVar, expectedVar)
		}
	}
}

const expectedBody = `{
  field1
  fragment1{ ...MainFragment }
  fragment2{ ...MainFragment }
  unionField{ ...MainUnion }
  lineItem_alias1:lineItem(first: 10, query: $lineItem_alias1_lineItem_query, id: $id){ ...MainLineItemConnect }
  inlineField1
  inline_alias2:inlineField2
  inlineField2
  tree{
    tree1{
      union1Field1{ ...MainUnion }
      tree1Field1
      tree1Field2{
        tree2Field1
        lineItem(query: $tree_tree1_tree1Field2_lineItem_query, first: 10){ ...MainLineItemConnect }
      }
      inline2:inline1{ ...MainInline }
      ...MainInline
    }
  }
  anonymity1{
    alias2:field1
  }
  ...MainFragment
}`

var expectedFragments = map[string]string{
	"MainFragment": `fragment MainFragment on Fragment{
  fragmentField1
  fragmentField2_alias1:fragmentField1
}`,
	"MainUnion": `fragment MainUnion on Union{
  __typename
  ... on Union1 {
    union1Field1
  }
  ... on Union2 {
    union1Field1
  }
  ... on Fragment { ...MainFragment }
}`,
	"MainLineItemConnect": `fragment MainLineItemConnect on LineItemConnect{
  nodes{
    lineItemField1
  }
}`,
	"MainInline": `fragment MainInline on Inline{
  inlineField1
  inline_alias2:inlineField2
}`,
}

// normalizeArgs 将字段参数按名称排序以消除 map 遍历顺序带来的波动
func normalizeArgs(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		start := strings.Index(line, "(")
		end := strings.Index(line, ")")
		if start == -1 || end == -1 || end <= start {
			continue
		}
		args := strings.Split(line[start+1:end], ",")
		for j := range args {
			args[j] = strings.TrimSpace(args[j])
		}
		sort.Strings(args)
		lines[i] = line[:start+1] + strings.Join(args, ", ") + line[end:]
	}
	return strings.Join(lines, "\n")
}
