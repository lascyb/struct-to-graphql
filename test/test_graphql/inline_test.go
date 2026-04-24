package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试匿名嵌入（字段平铺到父级）
type EmbedAddress struct {
	City   string `json:"city" graphql:"city"`
	Street string `json:"street" graphql:"street"`
}

type EmbedQuery struct {
	Name string `json:"name" graphql:"name"`
	EmbedAddress // 匿名嵌入：字段平铺到父级
}

// 测试匿名嵌入
type EmbeddedMeta struct {
	CreatedAt string `json:"createdAt" graphql:"createdAt"`
	UpdatedAt string `json:"updatedAt" graphql:"updatedAt"`
}

type EmbeddedQuery struct {
	ID string `json:"id" graphql:"id"`
	EmbeddedMeta
}
type EmbeddedQuery2 struct {
	In struct {
		Name string `json:"name" graphql:"name"`
	}
}

func TestEmbeddedAddress(t *testing.T) {
	exec, err := graphql.Marshal(EmbedQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("EmbedTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query (EmbedTest):\n%s", query)

	// 匿名嵌入的字段应该直接出现在父级
	if !strings.Contains(query, "city") {
		t.Error("Missing city field")
	}
	if !strings.Contains(query, "street") {
		t.Error("Missing street field")
	}
}

func TestEmbeddedStruct(t *testing.T) {
	exec, err := graphql.Marshal(EmbeddedQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("EmbeddedTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query (EmbeddedTest):\n%s", query)

	// 匿名嵌入的字段应该平铺到父级
	if !strings.Contains(query, "createdAt") {
		t.Error("Missing createdAt field from embedded struct")
	}
	if !strings.Contains(query, "updatedAt") {
		t.Error("Missing updatedAt field from embedded struct")
	}
}

// 测试扁平结构体（字段直接定义在顶层）
type FlatQuery struct {
	Name   string `json:"name" graphql:"name"`
	Bio    string `json:"bio" graphql:"bio"`
	Avatar string `json:"avatar" graphql:"avatar"`
}

func TestFlatStruct(t *testing.T) {
	exec, err := graphql.Marshal(FlatQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("FlatTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证字段直接平铺
	if !strings.Contains(query, "name") {
		t.Error("Missing name field")
	}
	if !strings.Contains(query, "bio") {
		t.Error("Missing bio field")
	}
	if !strings.Contains(query, "avatar") {
		t.Error("Missing avatar field")
	}
}

// ========== 匿名嵌入：encoding/json 往返测试 ==========
// 匿名嵌入在 GraphQL 和 encoding/json 中都表现为扁平化，完全兼容。

func TestEmbeddedUnmarshal(t *testing.T) {
	// EmbeddedQuery：匿名嵌入 EmbeddedMeta
	resp := `{"id":"1","createdAt":"2024-01-01","updatedAt":"2024-06-01"}`
	var got EmbeddedQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got ID=%q, want %q", got.ID, "1")
	}
	if got.CreatedAt != "2024-01-01" {
		t.Errorf("got CreatedAt=%q, want %q", got.CreatedAt, "2024-01-01")
	}
	if got.UpdatedAt != "2024-06-01" {
		t.Errorf("got UpdatedAt=%q, want %q", got.UpdatedAt, "2024-06-01")
	}
}

func TestEmbedAddressUnmarshal(t *testing.T) {
	// EmbedQuery：匿名嵌入 EmbedAddress
	resp := `{"name":"Alice","city":"Beijing","street":"Main St"}`
	var got EmbedQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("got Name=%q, want %q", got.Name, "Alice")
	}
	if got.City != "Beijing" {
		t.Errorf("got City=%q, want %q", got.City, "Beijing")
	}
	if got.Street != "Main St" {
		t.Errorf("got Street=%q, want %q", got.Street, "Main St")
	}
}

func TestFlatStructUnmarshal(t *testing.T) {
	// 扁平结构体：字段直接定义在顶层
	type FlatQueryCompatible struct {
		Name   string `json:"name" graphql:"name"`
		Bio    string `json:"bio" graphql:"bio"`
		Avatar string `json:"avatar" graphql:"avatar"`
	}
	resp := `{"name":"Alice","bio":"Hello","avatar":"pic.png"}`
	var got FlatQueryCompatible
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.Name != "Alice" || got.Bio != "Hello" || got.Avatar != "pic.png" {
		t.Errorf("got %+v, unexpected values", got)
	}
}
