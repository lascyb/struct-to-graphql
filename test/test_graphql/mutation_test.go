package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试 Mutation
type ProductInput struct {
	Title string `json:"title" graphql:"title"`
	Price string `json:"price" graphql:"price"`
}

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

func TestMutation(t *testing.T) {
	exec, err := graphql.Marshal(MutationStruct{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 使用 Mutation 方法
	mutation, err := exec.Mutation("ProductCreate")
	if err != nil {
		t.Fatalf("Mutation failed: %v", err)
	}

	t.Logf("Generated Mutation:\n%s", mutation)

	// 验证 mutation 关键字
	if !strings.HasPrefix(strings.TrimSpace(mutation), "mutation") {
		t.Error("Mutation should start with 'mutation' keyword")
	}

	// 验证变量
	if !strings.Contains(mutation, "$input:ProductInput!") {
		t.Error("Missing input variable")
	}

	// 验证返回字段
	if !strings.Contains(mutation, "productCreate") {
		t.Error("Missing productCreate field")
	}
	if !strings.Contains(mutation, "userErrors") {
		t.Error("Missing userErrors field")
	}
}
