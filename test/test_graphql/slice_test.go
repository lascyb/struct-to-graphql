package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试切片类型
type Tag struct {
	Name string `json:"name" graphql:"name"`
}

type SliceQuery struct {
	Tags     []Tag    `json:"tags" graphql:"tags"`
	Numbers  []int    `json:"numbers" graphql:"numbers"`
	Nested   [][]Tag  `json:"nested" graphql:"nested"`
}

func TestSliceType(t *testing.T) {
	exec, err := graphql.Marshal(SliceQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("SliceTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证切片字段存在
	if !strings.Contains(query, "tags") {
		t.Error("Missing tags field")
	}
	if !strings.Contains(query, "numbers") {
		t.Error("Missing numbers field")
	}
	if !strings.Contains(query, "nested") {
		t.Error("Missing nested field")
	}
	
	// 验证嵌套类型的字段
	if !strings.Contains(query, "name") {
		t.Error("Missing nested name field")
	}
}
