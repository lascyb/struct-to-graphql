package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试 Fragment 复用
type UserInfo struct {
	ID    string `json:"id" graphql:"id"`
	Name  string `json:"name" graphql:"name"`
	Email string `json:"email" graphql:"email"`
}

type FragmentQuery struct {
	Author   UserInfo `json:"author" graphql:"author"`
	Reviewer UserInfo `json:"reviewer" graphql:"reviewer"`
}

// 测试匿名结构体 + Fragment 复用
// Go 中每个匿名结构体实例都是不同类型，所以不会生成 Fragment，字段会独立展开
type AnonymousFragmentQuery struct {
	Author struct {
		ID   string `json:"id" graphql:"id"`
		Name string `json:"name" graphql:"name"`
	} `json:"author" graphql:"author"`
	Reviewer struct {
		ID   string `json:"id" graphql:"id"`
		Name string `json:"name" graphql:"name"`
	} `json:"reviewer" graphql:"reviewer"`
}

func TestAnonymousStructFragmentReuse(t *testing.T) {
	exec, err := graphql.Marshal(AnonymousFragmentQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("AnonymousFragmentTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证两个字段都存在
	if !strings.Contains(query, "author") {
		t.Error("Missing author field")
	}
	if !strings.Contains(query, "reviewer") {
		t.Error("Missing reviewer field")
	}
	// 验证字段内容正确展开（匿名结构体不会生成 Fragment，而是独立展开）
	if !strings.Contains(query, "id") {
		t.Error("Missing id field")
	}
	if !strings.Contains(query, "name") {
		t.Error("Missing name field")
	}
}

func TestFragmentReuse(t *testing.T) {
	exec, err := graphql.Marshal(FragmentQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("FragmentTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证 Fragment 被生成
	if !strings.Contains(query, "fragment") {
		t.Error("Should generate fragment for reused type")
	}
	
	// 验证 Fragment 被引用
	if !strings.Contains(query, "...") {
		t.Error("Should use fragment spread")
	}
	
	// 验证 author 和 reviewer 都使用了 fragment
	if !strings.Contains(query, "author") {
		t.Error("Missing author field")
	}
	if !strings.Contains(query, "reviewer") {
		t.Error("Missing reviewer field")
	}
}

// ========== Fragment：encoding/json 往返测试 ==========
// Fragment 是查询优化，响应 JSON 字段展开，encoding/json 直接兼容。

func TestFragmentUnmarshal(t *testing.T) {
	resp := `{"author":{"id":"1","name":"Alice","email":"alice@example.com"},"reviewer":{"id":"2","name":"Bob","email":"bob@example.com"}}`
	var got FragmentQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.Author.ID != "1" || got.Author.Name != "Alice" || got.Author.Email != "alice@example.com" {
		t.Errorf("got Author=%+v, unexpected", got.Author)
	}
	if got.Reviewer.ID != "2" || got.Reviewer.Name != "Bob" || got.Reviewer.Email != "bob@example.com" {
		t.Errorf("got Reviewer=%+v, unexpected", got.Reviewer)
	}
}
