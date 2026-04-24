package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试嵌套结构体
type Address struct {
	City   string `json:"city" graphql:"city"`
	Street string `json:"street" graphql:"street"`
}

type Company struct {
	Name    string  `json:"name" graphql:"name"`
	Address Address `json:"address" graphql:"address"`
}

type NestedQuery struct {
	User    string  `json:"user" graphql:"user"`
	Company Company `json:"company" graphql:"company"`
}

func TestNestedStruct(t *testing.T) {
	exec, err := graphql.Marshal(NestedQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("NestedTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证嵌套结构
	if !strings.Contains(query, "company") {
		t.Error("Missing company field")
	}
	if !strings.Contains(query, "address") {
		t.Error("Missing address field")
	}
	if !strings.Contains(query, "city") {
		t.Error("Missing city field")
	}
	if !strings.Contains(query, "street") {
		t.Error("Missing street field")
	}
}
