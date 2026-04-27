package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 综合测试：组合多种特性
type MediaInfo struct {
	URL    string `json:"url" graphql:"url"`
	Width  int    `json:"width" graphql:"width"`
	Height int    `json:"height" graphql:"height"`
}

type AuthorInfo struct {
	ID    string `json:"id" graphql:"id"`
	Name  string `json:"name" graphql:"name"`
	Email string `json:"email" graphql:"email"`
}

type ContentUnion struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	MediaInfo
	AuthorInfo
}

type Comment struct {
	ID     string     `json:"id" graphql:"id"`
	Author AuthorInfo `json:"author" graphql:"author"`
	Body   string     `json:"body" graphql:"body"`
}

type Meta struct {
	Views int `json:"views" graphql:"views"`
	Likes int `json:"likes" graphql:"likes"`
}

type CombinedQuery struct {
	ID       string       `json:"id" graphql:"id"`
	Title    string       `json:"title" graphql:"title,alias=headline"`
	Content  ContentUnion `json:"content" graphql:"content"`
	Author   AuthorInfo   `json:"author" graphql:"author"`
	Editor   AuthorInfo   `json:"editor" graphql:"editor"`
	Comments []Comment    `json:"comments" graphql:"comments(first:$first:Int=10)"`
	Meta                  // 匿名嵌入：字段平铺到父级
}

func TestCombinedFeatures(t *testing.T) {
	exec, err := graphql.Marshal(CombinedQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("CombinedTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证基本字段
	if !strings.Contains(query, "id") {
		t.Error("Missing id field")
	}

	// 验证别名
	if !strings.Contains(query, "headline:title") {
		t.Error("Missing aliased title field")
	}

	// 验证联合类型
	if !strings.Contains(query, "__typename") {
		t.Error("Missing __typename for union")
	}
	if !strings.Contains(query, "... on") {
		t.Error("Missing inline fragment for union")
	}

	// 验证 Fragment 复用（AuthorInfo 被多处使用）
	if !strings.Contains(query, "fragment") {
		t.Error("Should generate fragment for reused AuthorInfo")
	}

	// 验证匿名嵌入
	if !strings.Contains(query, "views") {
		t.Error("Missing views field from embedded meta")
	}
	if !strings.Contains(query, "likes") {
		t.Error("Missing likes field from embedded meta")
	}

	// 验证变量默认值
	if !strings.Contains(query, "$first:Int=10") {
		t.Error("Missing default value for first variable")
	}
}

// ========== alias 兼容性说明 ==========
// alias 机制在 GraphQL 中使用 alias:field 格式，响应 JSON 的 key 是 alias 名。
// 要使 encoding/json 兼容，struct 的 json tag 必须与 alias 名一致。
//
// ✅ 兼容：json tag 匹配 alias 名
//   DisplayName string `json:"displayName" graphql:"name,alias=displayName"`
//   → 查询：displayName:name → 响应：{"displayName":"Alice"} → json tag "displayName" 匹配 ✅
//
// ❌ 不兼容：json tag 不匹配 alias 名
//   Title string `json:"title" graphql:"title,alias=headline"`
//   → 查询：headline:title → 响应：{"headline":"My Title"} → json tag "title" 不匹配 ❌
//
// 修正：将 json tag 改为 alias 名
//   Title string `json:"headline" graphql:"title,alias=headline"`
//   → 响应：{"headline":"My Title"} → json tag "headline" 匹配 ✅

// 综合场景：验证 alias json tag 不匹配时 encoding/json 的行为
func TestCombinedUnmarshal_AliasMismatch(t *testing.T) {
	resp := `{
		"id": "1",
		"headline": "My Title",
		"content": {"__typename": "MediaInfo", "url": "http://img.png", "width": 100, "height": 200},
		"author": {"id": "10", "name": "Alice", "email": "alice@example.com"},
		"editor": {"id": "11", "name": "Bob", "email": "bob@example.com"},
		"comments": [{"id": "c1", "author": {"id": "10", "name": "Alice", "email": "alice@example.com"}, "body": "Nice!"}],
		"views": 100,
		"likes": 50
	}`
	var got CombinedQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got ID=%q, want %q", got.ID, "1")
	}
	// alias 不兼容：json tag 是 "title"，但响应 key 是 "headline"，无法匹配
	if got.Title != "" {
		t.Errorf("Title should be empty because json tag 'title' doesn't match response key 'headline', got %q", got.Title)
	}
	// union 正常
	if got.Content.Typename != "MediaInfo" {
		t.Errorf("got __typename=%q, want %q", got.Content.Typename, "MediaInfo")
	}
	if got.Content.MediaInfo.URL != "http://img.png" {
		t.Errorf("got URL=%q, want %q", got.Content.MediaInfo.URL, "http://img.png")
	}
	// 匿名嵌入正常：views/likes 直接扁平化写入
	if got.Views != 100 {
		t.Errorf("got Views=%d, want 100", got.Views)
	}
	if got.Likes != 50 {
		t.Errorf("got Likes=%d, want 50", got.Likes)
	}
}

// 综合场景：修正后的兼容版本
func TestCombinedUnmarshal_Compatible(t *testing.T) {
	// 修正：json tag 与 alias 名一致
	type MetaEmbed struct {
		Views int `json:"views" graphql:"views"`
		Likes int `json:"likes" graphql:"likes"`
	}
	type CombinedQueryCompatible struct {
		ID        string       `json:"id" graphql:"id"`
		Title     string       `json:"headline" graphql:"title,alias=headline"` // json tag 匹配 alias
		Content   ContentUnion `json:"content" graphql:"content"`
		Author    AuthorInfo   `json:"author" graphql:"author"`
		Editor    AuthorInfo   `json:"editor" graphql:"editor"`
		Comments  []Comment    `json:"comments" graphql:"comments(first:$first:Int=10)"`
		MetaEmbed              // 匿名嵌入：encoding/json 天然扁平化
	}
	resp := `{
		"id": "1",
		"headline": "My Title",
		"content": {"__typename": "MediaInfo", "url": "http://img.png", "width": 100, "height": 200},
		"author": {"id": "10", "name": "Alice", "email": "alice@example.com"},
		"editor": {"id": "11", "name": "Bob", "email": "bob@example.com"},
		"comments": [{"id": "c1", "author": {"id": "10", "name": "Alice", "email": "alice@example.com"}, "body": "Nice!"}],
		"views": 100,
		"likes": 50
	}`
	var got CombinedQueryCompatible
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got ID=%q, want %q", got.ID, "1")
	}
	// alias 兼容：json tag "headline" 匹配响应 key "headline"
	if got.Title != "My Title" {
		t.Errorf("got Title=%q, want %q", got.Title, "My Title")
	}
	// 匿名嵌入兼容：views/likes 直接扁平化
	if got.Views != 100 {
		t.Errorf("got Views=%d, want 100", got.Views)
	}
	if got.Likes != 50 {
		t.Errorf("got Likes=%d, want 50", got.Likes)
	}
	// union 正常
	if got.Content.Typename != "MediaInfo" {
		t.Errorf("got __typename=%q, want %q", got.Content.Typename, "MediaInfo")
	}
	if got.Content.MediaInfo.URL != "http://img.png" {
		t.Errorf("got URL=%q, want %q", got.Content.MediaInfo.URL, "http://img.png")
	}
}

// ========== 指针与切片：encoding/json 往返测试 ==========

func TestPointersUnmarshal(t *testing.T) {
	type PtrQuery struct {
		Name   string   `json:"name" graphql:"name"`
		Values []string `json:"values" graphql:"values"`
	}
	resp := `{"name":"test","values":["a","b","c"]}`
	var got PtrQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("got Name=%q, want %q", got.Name, "test")
	}
	if len(got.Values) != 3 || got.Values[0] != "a" {
		t.Errorf("got Values=%v, unexpected", got.Values)
	}
}

func TestNilPointerUnmarshal(t *testing.T) {
	type NilPtrQuery struct {
		Name    string `json:"name" graphql:"name"`
		Profile *struct {
			Age int `json:"age" graphql:"age"`
		} `json:"profile" graphql:"profile"`
	}
	// profile 为 null
	resp := `{"name":"test","profile":null}`
	var got NilPtrQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.Profile != nil {
		t.Errorf("got Profile=%+v, want nil", got.Profile)
	}

	// profile 有值
	resp2 := `{"name":"test","profile":{"age":25}}`
	var got2 NilPtrQuery
	if err := json.Unmarshal([]byte(resp2), &got2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got2.Profile == nil || got2.Profile.Age != 25 {
		t.Errorf("got Profile.Age=%v, want 25", got2.Profile)
	}
}
