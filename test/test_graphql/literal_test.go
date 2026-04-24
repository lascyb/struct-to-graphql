package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试字面量参数
type LiteralQuery struct {
	Items struct {
		Nodes []struct {
			ID string `json:"id" graphql:"id"`
		} `json:"nodes" graphql:"nodes"`
	} `json:"items" graphql:"items(first:10, status:\"active\")"`
}

func TestLiteralArgument(t *testing.T) {
	exec, err := graphql.Marshal(LiteralQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("LiteralTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证数字字面量
	if !strings.Contains(query, "first:10") {
		t.Error("Missing numeric literal first:10")
	}
	
	// 验证字符串字面量
	if !strings.Contains(query, `status:"active"`) {
		t.Error(`Missing string literal status:"active"`)
	}
}

// 测试匿名结构体 + 变量参数
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

func TestAnonymousStructWithVariable(t *testing.T) {
	exec, err := graphql.Marshal(AnonymousVariableQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("AnonymousVariableTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证变量声明
	if !strings.Contains(query, "$first:Int!") {
		t.Error("Missing required variable $first:Int!")
	}
	if !strings.Contains(query, "$after:String") {
		t.Error("Missing variable $after:String")
	}
	// 验证字段参数中使用了变量
	if !strings.Contains(query, "first:$first") {
		t.Error("Missing first:$first in field args")
	}
	if !strings.Contains(query, "after:$after") {
		t.Error("Missing after:$after in field args")
	}
	// 验证匿名结构体内部字段
	if !strings.Contains(query, "nodes") {
		t.Error("Missing nodes field")
	}
	if !strings.Contains(query, "pageInfo") {
		t.Error("Missing pageInfo field")
	}
	if !strings.Contains(query, "hasNextPage") {
		t.Error("Missing hasNextPage field")
	}
	if !strings.Contains(query, "endCursor") {
		t.Error("Missing endCursor field")
	}
}
