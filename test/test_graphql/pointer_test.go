package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试指针类型
type ProfileInfo struct {
	Bio    string `json:"bio" graphql:"bio"`
	Avatar string `json:"avatar" graphql:"avatar"`
}

type PointerQuery struct {
	ID      string       `json:"id" graphql:"id"`
	Profile *ProfileInfo `json:"profile" graphql:"profile"`
	Tags    []*struct {
		Name string `json:"name" graphql:"name"`
	} `json:"tags" graphql:"tags"`
}

func TestPointerType(t *testing.T) {
	exec, err := graphql.Marshal(PointerQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("PointerTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证指针字段被正确处理
	if !strings.Contains(query, "profile") {
		t.Error("Missing profile field")
	}
	if !strings.Contains(query, "bio") {
		t.Error("Missing bio field")
	}
	if !strings.Contains(query, "avatar") {
		t.Error("Missing avatar field")
	}
	if !strings.Contains(query, "tags") {
		t.Error("Missing tags field")
	}
}
