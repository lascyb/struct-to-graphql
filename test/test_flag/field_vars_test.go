package test_flag

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

type Order struct {
	ID string `json:"id" graphql:"id"`
}

// 带引号的字面量中可以包含 ":"、空格等字符，整体作为 GraphQL 字符串输出；裸标识符按枚举值原样输出
type QuotedLiteralQuery struct {
	Orders []Order `json:"orders" graphql:"orders(query:'financial_status:paid OR financial_status:partially_refunded',sortKey:PROCESSED_AT)"`
}

// $name:Type 中 ":" 之后的部分是变量类型，而不是字面量的一部分
type VariableTypeQuery struct {
	Orders []Order `json:"orders" graphql:"orders(sort:$desc:String!)"`
}

func TestFieldVars_QuotedLiteralKeepsColon(t *testing.T) {
	t.Parallel()
	testutil.AssertEqual(t, testutil.Query(t, QuotedLiteralQuery{}, "MyQuery"), `query MyQuery {
  orders(query:"financial_status:paid OR financial_status:partially_refunded",sortKey:PROCESSED_AT){
    id
  }
}`)
}

func TestFieldVars_VariableTypeAfterColon(t *testing.T) {
	t.Parallel()
	exec, got := testutil.Build(t, VariableTypeQuery{}, testutil.OpQuery, "MyQuery")
	testutil.AssertEqual(t, got, `query MyQuery($desc:String!) {
  orders(sort:$desc){
    id
  }
}`)
	testutil.AssertVariables(t, exec.Variables, []graphql.Variable{
		{Name: "$desc", Type: "String!", Paths: []string{"orders"}},
	})
}
