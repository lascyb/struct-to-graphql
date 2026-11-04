package test_graphql

import (
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== Mutation ==========

type ProductPayload struct {
	Product struct {
		ID    string `json:"id" graphql:"id"`
		Title string `json:"title" graphql:"title"`
	} `json:"product" graphql:"product"`
	UserErrors []struct {
		Field   []string `json:"field" graphql:"field"`
		Message string   `json:"message" graphql:"message"`
	} `json:"userErrors" graphql:"userErrors"`
}

type MutationStruct struct {
	ProductCreate ProductPayload `json:"productCreate" graphql:"productCreate(input:$input:ProductInput!)"`
}

func TestMutationOutput(t *testing.T) {
	t.Parallel()
	runOutputCases(t, []outputCase{
		{name: "Mutation", v: MutationStruct{}, op: testutil.OpMutation, opName: "ProductCreate", want: `mutation ProductCreate($input:ProductInput!) {
  productCreate(input:$input){
    product{
      id
      title
    }
    userErrors{
      field
      message
    }
  }
}`},
		// 匿名操作：name 为空时只输出操作关键字
		{name: "AnonymousMutation", v: MutationStruct{}, op: testutil.OpMutation, want: `mutation($input:ProductInput!) {
  productCreate(input:$input){
    product{
      id
      title
    }
    userErrors{
      field
      message
    }
  }
}`},
		{name: "AnonymousQuery", v: AliasQuery{}, want: `query {
  id
  displayName:name
  userEmail:email
}`},
	})
}
