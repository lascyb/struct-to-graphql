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
		{Name: "$lineItem_query", Path: []string{"tree"}},
		{Name: "$tree_tree1_tree1Field2_lineItem_query", Path: []string{"tree", "tree1", "tree1Field2", "lineItem"}},
	}
	if len(exec.Variables) != len(expectedVars) {
		t.Fatalf("变量数量不符，实际 %d 预期 %d", len(exec.Variables), len(expectedVars))
	}
	for i, v := range exec.Variables {
		if v == nil {
			t.Fatalf("变量 %d 为 nil", i)
		}
		if v.Name != expectedVars[i].Name || !reflect.DeepEqual(v.Path, expectedVars[i].Path) {
			t.Fatalf("变量 %d 不符合预期，实际 %+v 预期 %+v", i, v, expectedVars[i])
		}
	}
}

const expectedBody = `{
  field1
  fragment1{ ...MainFragment }
  fragment2{ ...MainFragment }
  unionField{ ...MainUnion }
  lineItem(first: 10, query: $lineItem_query){ ...MainLineItemConnect }
  inlineField1
  inlineField2
  tree{
    tree1{
      union1Field1{ ...MainUnion }
      tree1Field1
      tree1Field2{
        tree2Field1
        lineItem(first: 10, query: $tree_tree1_tree1Field2_lineItem_query){ ...MainLineItemConnect }
      }
    }
  }
}`

var expectedFragments = map[string]string{
	"MainFragment": `fragment MainFragment on Fragment{
  fragmentField1
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
