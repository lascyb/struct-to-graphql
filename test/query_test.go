package main

import (
	"strings"
	"testing"

	"github.com/lascyb/struct-to-graphql"
)

func TestQuery(t *testing.T) {
	type Query struct {
		Field1 string `graphql:"field1"`
		List   struct {
			Nodes []struct {
				Name string `graphql:"name"`
			} `graphql:"nodes"`
		} `graphql:"list(first:10,query:$:String!,id:$id:Int!)"`
	}

	q, err := graphql.Marshal(Query{})
	if err != nil {
		t.Fatalf("Marshal 返回错误: %v", err)
	}

	query, err := q.Query("GetData")
	if err != nil {
		t.Fatalf("Query 返回错误: %v", err)
	}

	// 验证变量定义中包含 $ 符号
	if !strings.Contains(query, "query GetData($") {
		t.Fatalf("变量定义中缺少 $ 符号，实际输出:\n%s", query)
	}

	// 验证所有变量都有 $ 符号
	expectedVars := []string{"$list_query", "$id"}
	for _, expectedVar := range expectedVars {
		if !strings.Contains(query, expectedVar+":") {
			t.Fatalf("缺少变量 %s，实际输出:\n%s", expectedVar, query)
		}
	}
}
