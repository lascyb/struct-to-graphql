package test_graphql

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== 字段参数：字面量、变量、默认值 ==========

type IDNode struct {
	ID string `json:"id" graphql:"id"`
}

// 字面量规则：带引号 → GraphQL 字符串；裸标识符 → 原样（枚举）；数字 / 布尔 / null → 原样；列表逐元素
type LiteralQuery struct {
	Items struct {
		Nodes []IDNode `json:"nodes" graphql:"nodes"`
	} `json:"items" graphql:"items(first:10, status:\"active\", sortKey:CREATED_AT, reverse:true, after:null, offset:-5, ratio:0.5, ids:[1,2], types:[SALE,'gift card'])"`
}

// 字符串中的特殊字符会被转义为合法的 GraphQL 字符串
type EscapeQuery struct {
	Search []IDNode `json:"search" graphql:"search(path:'C:\\\\dir', quote:'say \"hi\"', single:'it\\'s', multi:'line1\nline2\ttab', unicode:'中文')"`
}

type AnonymousVariableQuery struct {
	Items struct {
		Nodes []struct {
			ID    string `json:"id" graphql:"id"`
			Title string `json:"title" graphql:"title"`
		} `json:"nodes" graphql:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage" graphql:"hasNextPage"`
			EndCursor   string `json:"endCursor" graphql:"endCursor"`
		} `json:"pageInfo" graphql:"pageInfo"`
	} `json:"items" graphql:"items(first:$first:Int!, after:$after:String)"`
}

type ItemConnection struct {
	Nodes []IDNode `json:"nodes" graphql:"nodes"`
}

// 匿名占位符 $ 的变量名为 <路径>_<参数名>（驼峰转下划线）
type VariableQuery struct {
	Items ItemConnection `json:"items" graphql:"items(first:$first:Int!, after:$:String, filter:$filter:String)"`
}

// 默认值：数字、布尔、字符串、枚举、null、列表
type DefaultValueQuery struct {
	Items ItemConnection `json:"items" graphql:"items(page:$page:Int=1, limit:$limit:Int=10, ratio:$ratio:Float=0.5, all:$all:Boolean=false, query:$query:String='a \"b\"', bare:$bare:String=hello, sortKey:$sortKey:SortKey=CREATED_AT, after:$after:String=null, ids:$ids:[ID!]=[a,'b'], sizes:$sizes:[Int]=[1,2])"`
}

func TestArgumentOutput(t *testing.T) {
	t.Parallel()
	runOutputCases(t, []outputCase{
		{name: "Literal", v: LiteralQuery{}, opName: "LiteralTest", want: `query LiteralTest {
  items(first:10,status:"active",sortKey:CREATED_AT,reverse:true,after:null,offset:-5,ratio:0.5,ids:[1,2],types:[SALE,"gift card"]){
    nodes{
      id
    }
  }
}`},
		{name: "Escape", v: EscapeQuery{}, opName: "EscapeTest", want: `query EscapeTest {
  search(path:"C:\\dir",quote:"say \"hi\"",single:"it's",multi:"line1\nline2\ttab",unicode:"中文"){
    id
  }
}`},
		{name: "AnonymousVariable", v: AnonymousVariableQuery{}, opName: "AnonymousVariableTest", want: `query AnonymousVariableTest($first:Int!,$after:String) {
  items(first:$first,after:$after){
    nodes{
      id
      title
    }
    pageInfo{
      hasNextPage
      endCursor
    }
  }
}`},
		{name: "Variable", v: VariableQuery{}, opName: "VariableTest", want: `query VariableTest($first:Int!,$items_after:String,$filter:String) {
  items(first:$first,after:$items_after,filter:$filter){
    nodes{
      id
    }
  }
}`},
		{name: "DefaultValue", v: DefaultValueQuery{}, opName: "DefaultValueTest", want: `query DefaultValueTest($page:Int=1,$limit:Int=10,$ratio:Float=0.5,$all:Boolean=false,$query:String="a \"b\"",$bare:String="hello",$sortKey:SortKey=CREATED_AT,$after:String=null,$ids:[ID!]=["a","b"],$sizes:[Int]=[1,2]) {
  items(page:$page,limit:$limit,ratio:$ratio,all:$all,query:$query,bare:$bare,sortKey:$sortKey,after:$after,ids:$ids,sizes:$sizes){
    nodes{
      id
    }
  }
}`},
	})
}

func TestArgument_VariableMetadata(t *testing.T) {
	t.Parallel()
	exec, _ := testutil.Build(t, VariableQuery{}, testutil.OpQuery, "VariableTest")
	testutil.AssertVariables(t, exec.Variables, []graphql.Variable{
		{Name: "$first", Type: "Int!", Paths: []string{"items"}},
		{Name: "$items_after", Type: "String", Paths: []string{"items"}},
		{Name: "$filter", Type: "String", Paths: []string{"items"}},
	})

	exec, _ = testutil.Build(t, DefaultValueQuery{}, testutil.OpQuery, "DefaultValueTest")
	if got := exec.Variables[0]; !got.HasDefault || got.DefaultValue != 1 || got.Default != "1" {
		t.Fatalf("unexpected default for $page: %+v", got)
	}
}
