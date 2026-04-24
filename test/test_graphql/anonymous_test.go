package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试匿名结构体
type AnonymousQuery struct {
	ID string `json:"id" graphql:"id"`
	Settings struct {
		Theme    string `json:"theme" graphql:"theme"`
		Language string `json:"language" graphql:"language"`
	} `json:"settings" graphql:"settings"`
	Metadata struct {
		CreatedAt string `json:"createdAt" graphql:"createdAt"`
	} `json:"metadata" graphql:"metadata"`
}

func TestAnonymousStruct(t *testing.T) {
	exec, err := graphql.Marshal(AnonymousQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("AnonymousTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证匿名结构体字段被正确处理
	if !strings.Contains(query, "settings") {
		t.Error("Missing settings field")
	}
	if !strings.Contains(query, "theme") {
		t.Error("Missing theme field")
	}
	if !strings.Contains(query, "language") {
		t.Error("Missing language field")
	}
	if !strings.Contains(query, "metadata") {
		t.Error("Missing metadata field")
	}
	if !strings.Contains(query, "createdAt") {
		t.Error("Missing createdAt field")
	}
}

// 测试匿名结构体嵌套
type AnonymousNestedQuery struct {
	Data struct {
		User struct {
			Name string `json:"name" graphql:"name"`
			Age  int    `json:"age" graphql:"age"`
		} `json:"user" graphql:"user"`
		Meta struct {
			CreatedAt string `json:"createdAt" graphql:"createdAt"`
			UpdatedAt string `json:"updatedAt" graphql:"updatedAt"`
		} `json:"meta" graphql:"meta"`
	} `json:"data" graphql:"data"`
}

func TestAnonymousStructNested(t *testing.T) {
	exec, err := graphql.Marshal(AnonymousNestedQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("AnonymousNestedTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证外层匿名结构体
	if !strings.Contains(query, "data") {
		t.Error("Missing data field")
	}
	// 验证内层匿名结构体
	if !strings.Contains(query, "user") {
		t.Error("Missing user field")
	}
	if !strings.Contains(query, "name") {
		t.Error("Missing name field")
	}
	if !strings.Contains(query, "age") {
		t.Error("Missing age field")
	}
	if !strings.Contains(query, "meta") {
		t.Error("Missing meta field")
	}
	if !strings.Contains(query, "createdAt") {
		t.Error("Missing createdAt field")
	}
	if !strings.Contains(query, "updatedAt") {
		t.Error("Missing updatedAt field")
	}
}

// 测试匿名结构体切片
type AnonymousSliceQuery struct {
	Tags []struct {
		Name  string `json:"name" graphql:"name"`
		Color string `json:"color" graphql:"color"`
	} `json:"tags" graphql:"tags"`
	Edges []struct {
		Node struct {
			ID    string `json:"id" graphql:"id"`
			Title string `json:"title" graphql:"title"`
		} `json:"node" graphql:"node"`
	} `json:"edges" graphql:"edges"`
}

func TestAnonymousStructSlice(t *testing.T) {
	exec, err := graphql.Marshal(AnonymousSliceQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("AnonymousSliceTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证简单匿名结构体切片
	if !strings.Contains(query, "tags") {
		t.Error("Missing tags field")
	}
	if !strings.Contains(query, "name") {
		t.Error("Missing name field in tags")
	}
	if !strings.Contains(query, "color") {
		t.Error("Missing color field in tags")
	}
	// 验证嵌套匿名结构体切片
	if !strings.Contains(query, "edges") {
		t.Error("Missing edges field")
	}
	if !strings.Contains(query, "node") {
		t.Error("Missing node field in edges")
	}
	if !strings.Contains(query, "id") {
		t.Error("Missing id field in node")
	}
	if !strings.Contains(query, "title") {
		t.Error("Missing title field in node")
	}
}

// ========== 匿名结构体：encoding/json 往返测试 ==========

func TestAnonymousUnmarshal(t *testing.T) {
	resp := `{"id":"1","settings":{"theme":"dark","language":"go"},"metadata":{"createdAt":"2024-01-01"}}`
	var got AnonymousQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got ID=%q, want %q", got.ID, "1")
	}
	if got.Settings.Theme != "dark" {
		t.Errorf("got Theme=%q, want %q", got.Settings.Theme, "dark")
	}
	if got.Settings.Language != "go" {
		t.Errorf("got Language=%q, want %q", got.Settings.Language, "go")
	}
	if got.Metadata.CreatedAt != "2024-01-01" {
		t.Errorf("got CreatedAt=%q, want %q", got.Metadata.CreatedAt, "2024-01-01")
	}
}

func TestAnonymousNestedUnmarshal(t *testing.T) {
	resp := `{"data":{"user":{"name":"Alice","age":30},"meta":{"createdAt":"2024-01-01","updatedAt":"2024-06-01"}}}`
	var got AnonymousNestedQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.Data.User.Name != "Alice" {
		t.Errorf("got Name=%q, want %q", got.Data.User.Name, "Alice")
	}
	if got.Data.User.Age != 30 {
		t.Errorf("got Age=%d, want %d", got.Data.User.Age, 30)
	}
	if got.Data.Meta.CreatedAt != "2024-01-01" {
		t.Errorf("got CreatedAt=%q, want %q", got.Data.Meta.CreatedAt, "2024-01-01")
	}
}

func TestAnonymousSliceUnmarshal(t *testing.T) {
	resp := `{"tags":[{"name":"go","color":"blue"},{"name":"rust","color":"orange"}],"edges":[{"node":{"id":"1","title":"Hello"}}]}`
	var got AnonymousSliceQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(got.Tags))
	}
	if got.Tags[0].Name != "go" || got.Tags[0].Color != "blue" {
		t.Errorf("got tag[0]=%+v, want {go blue}", got.Tags[0])
	}
	if len(got.Edges) != 1 || got.Edges[0].Node.ID != "1" {
		t.Errorf("got edges unexpected: %+v", got.Edges)
	}
}
