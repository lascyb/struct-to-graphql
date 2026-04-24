package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试字段别名
type AliasQuery struct {
	ID          string `json:"id" graphql:"id"`
	DisplayName string `json:"displayName" graphql:"name,alias=displayName"`
	UserEmail   string `json:"userEmail" graphql:"email,alias=userEmail"`
}

// 测试匿名结构体 + alias
type AnonymousAliasQuery struct {
	User struct {
		UserName string `json:"userName" graphql:"name,alias=userName"`
		Email    string `json:"email" graphql:"email"`
	} `json:"user" graphql:"user"`
	Profile struct {
		Age int `json:"age" graphql:"age"`
	} `json:"profile" graphql:"info,alias=profile"`
}

func TestAnonymousStructWithAlias(t *testing.T) {
	exec, err := graphql.Marshal(AnonymousAliasQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("AnonymousAliasTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证匿名结构体内部字段使用 alias
	if !strings.Contains(query, "userName:name") {
		t.Error("Missing aliased field userName:name inside anonymous struct")
	}
	// 验证匿名结构体字段本身使用 alias
	if !strings.Contains(query, "profile:info") {
		t.Error("Missing aliased field profile:info for anonymous struct field")
	}
	// 验证普通字段仍然正常
	if !strings.Contains(query, "email") {
		t.Error("Missing email field")
	}
	if !strings.Contains(query, "age") {
		t.Error("Missing age field")
	}
}

func TestFieldAlias(t *testing.T) {
	exec, err := graphql.Marshal(AliasQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("AliasTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证别名格式：alias:fieldName
	if !strings.Contains(query, "displayName:name") {
		t.Error("Missing aliased field displayName:name")
	}
	if !strings.Contains(query, "userEmail:email") {
		t.Error("Missing aliased field userEmail:email")
	}
}

// ========== 别名：encoding/json 往返测试 ==========
// alias 机制：GraphQL 查询使用 alias:field 格式，响应 JSON 的 key 是 alias 名。
// 只要 struct 的 json tag 与 alias 名一致，encoding/json 就能直接写入。

func TestAliasUnmarshal(t *testing.T) {
	resp := `{"id":"1","displayName":"Alice","userEmail":"alice@example.com"}`
	var got AliasQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got ID=%q, want %q", got.ID, "1")
	}
	if got.DisplayName != "Alice" {
		t.Errorf("got DisplayName=%q, want %q", got.DisplayName, "Alice")
	}
	if got.UserEmail != "alice@example.com" {
		t.Errorf("got UserEmail=%q, want %q", got.UserEmail, "alice@example.com")
	}
}

func TestAnonymousAliasUnmarshal(t *testing.T) {
	resp := `{"user":{"userName":"Alice","email":"alice@example.com"},"profile":{"age":25}}`
	var got AnonymousAliasQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.User.UserName != "Alice" {
		t.Errorf("got UserName=%q, want %q", got.User.UserName, "Alice")
	}
	if got.Profile.Age != 25 {
		t.Errorf("got Age=%d, want %d", got.Profile.Age, 25)
	}
}
