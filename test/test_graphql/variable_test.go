package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试变量定义
type ItemConnection struct {
	Nodes []struct {
		ID string `json:"id" graphql:"id"`
	} `json:"nodes" graphql:"nodes"`
}

type VariableQuery struct {
	Items ItemConnection `json:"items" graphql:"items(first:$first:Int!, after:$:String, filter:$filter:String)"`
}

// 测试默认值
type DefaultValueQuery struct {
	Items ItemConnection `json:"items" graphql:"items(page:$page:Int=1, limit:$limit:Int=10)"`
}

func TestVariableDefinition(t *testing.T) {
	exec, err := graphql.Marshal(VariableQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("VariableTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证变量声明在查询头部
	if !strings.Contains(query, "$first:Int!") {
		t.Error("Missing required variable $first:Int!")
	}
	// 检查自动生成的变量名（格式为 $<path>_<argName>）
	if !strings.Contains(query, "$items_after:String") {
		t.Errorf("Missing variable with auto-generated name for 'after' argument.\nQuery: %s", query)
	}
	if !strings.Contains(query, "$filter:String") {
		t.Error("Missing optional variable $filter:String")
	}
	// 验证字段参数中使用了正确的变量引用
	if !strings.Contains(query, "after:$items_after") {
		t.Errorf("Field argument 'after' should use auto-generated variable '$items_after'.\nQuery: %s", query)
	}
}

func TestDefaultValue(t *testing.T) {
	exec, err := graphql.Marshal(DefaultValueQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("DefaultValueTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证默认值
	if !strings.Contains(query, "$page:Int=1") {
		t.Error("Missing default value for page")
	}
	if !strings.Contains(query, "$limit:Int=10") {
		t.Error("Missing default value for limit")
	}
}
