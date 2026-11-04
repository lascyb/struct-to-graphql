package graphql_test

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 手动构造的 Graphql（Variable 未设置 Default）按 DefaultValue 生成默认值写法
func TestGraphql_ManualVariables(t *testing.T) {
	g := &graphql.Graphql{
		Body: "{\n  items(q:$q,n:$n,l:$l,z:$z)\n}",
		Variables: []*graphql.Variable{
			{Name: "$q", Type: "String", HasDefault: true, DefaultValue: "say \"hi\""},
			{Name: "$n", Type: "Int", HasDefault: true, DefaultValue: 3},
			{Name: "$l", Type: "[Int]", HasDefault: true, DefaultValue: []interface{}{1, 2}},
			{Name: "$z", Type: "String", HasDefault: true},
			{Name: "$e", Type: "Sort", HasDefault: true, DefaultValue: "ASC", Default: "ASC"},
		},
		Fragments: []*graphql.Fragment{{Name: "F", Type: "T", Body: "fragment F on T{\n  id\n}"}},
	}
	got, err := g.Query("Manual")
	if err != nil {
		t.Fatal(err)
	}
	want := `fragment F on T{
  id
}
query Manual($q:String="say \"hi\"",$n:Int=3,$l:[Int]=[1,2],$z:String=null,$e:Sort=ASC) {
  items(q:$q,n:$n,l:$l,z:$z)
}`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
