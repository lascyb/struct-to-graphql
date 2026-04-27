package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试联合类型（Union）
type TextContent struct {
	Text string `json:"text" graphql:"text"`
}

type ImageContent struct {
	URL    string `json:"url" graphql:"url"`
	Width  int    `json:"width" graphql:"width"`
	Height int    `json:"height" graphql:"height"`
}

type Content struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	TextContent
	ImageContent
}

type UnionQuery struct {
	ID      string  `json:"id" graphql:"id"`
	Content Content `json:"content" graphql:"content"`
}

// 测试匿名结构体 + Union（匿名结构体不支持作为联合类型成员，应返回错误）
type AnonymousUnionContent struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	TextData struct {
		Text string `json:"text" graphql:"text"`
	}
	ImageData struct {
		URL   string `json:"url" graphql:"url"`
		Width int    `json:"width" graphql:"width"`
	}
}

// 匿名结构体即使设置 type flag 也不应作为 union 分支
type AnonymousUnionWithTypeContent struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	TextData struct {
		Text string `json:"text" graphql:"text"`
	} `graphql:"textData,type=TextData"`
	ImageData struct {
		URL   string `json:"url" graphql:"url"`
		Width int    `json:"width" graphql:"width"`
	} `graphql:"imageData,type=ImageData"`
}

type AnonymousUnionWithTypeQuery struct {
	ID      string                        `json:"id" graphql:"id"`
	Content AnonymousUnionWithTypeContent `json:"content" graphql:"content"`
}

type AnonymousUnionQuery struct {
	ID      string                `json:"id" graphql:"id"`
	Content AnonymousUnionContent `json:"content" graphql:"content"`
}

func TestAnonymousStructUnion(t *testing.T) {
	_, err := graphql.Marshal(AnonymousUnionQuery{})
	if err == nil {
		t.Error("Expected error for anonymous struct in union type, but got nil")
	}
	t.Logf("Expected error: %v", err)
}

func TestAnonymousStructUnionWithTypeFlag(t *testing.T) {
	_, err := graphql.Marshal(AnonymousUnionWithTypeQuery{})
	if err == nil {
		t.Error("Expected error for anonymous struct in union type (even with type flag), but got nil")
	}
	t.Logf("Expected error: %v", err)
}

func TestUnionType(t *testing.T) {
	exec, err := graphql.Marshal(UnionQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	query, err := exec.Query("UnionTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Generated Query:\n%s", query)

	// 验证 __typename 字段
	if !strings.Contains(query, "__typename") {
		t.Error("Missing __typename field")
	}

	// 验证 inline fragment 语法
	if !strings.Contains(query, "... on") {
		t.Error("Should use inline fragment syntax (... on)")
	}

	// 验证 TextContent 分支
	if !strings.Contains(query, "TextContent") {
		t.Error("Missing TextContent type")
	}

	// 验证 ImageContent 分支
	if !strings.Contains(query, "ImageContent") {
		t.Error("Missing ImageContent type")
	}
}

// ========== 联合类型：encoding/json 往返测试 ==========
// union 使用 __typename + 匿名嵌入，encoding/json 原生支持。

func TestUnionUnmarshal(t *testing.T) {
	// TextContent 分支
	resp := `{"id":"1","content":{"__typename":"TextContent","text":"hello"}}`
	var got UnionQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("got ID=%q, want %q", got.ID, "1")
	}
	if got.Content.Typename != "TextContent" {
		t.Errorf("got __typename=%q, want %q", got.Content.Typename, "TextContent")
	}
	if got.Content.TextContent.Text != "hello" {
		t.Errorf("got Text=%q, want %q", got.Content.TextContent.Text, "hello")
	}

	// ImageContent 分支
	resp2 := `{"id":"2","content":{"__typename":"ImageContent","url":"http://img.png","width":100,"height":200}}`
	var got2 UnionQuery
	if err := json.Unmarshal([]byte(resp2), &got2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if got2.Content.Typename != "ImageContent" {
		t.Errorf("got __typename=%q, want %q", got2.Content.Typename, "ImageContent")
	}
	if got2.Content.ImageContent.URL != "http://img.png" {
		t.Errorf("got URL=%q, want %q", got2.Content.ImageContent.URL, "http://img.png")
	}
	if got2.Content.ImageContent.Width != 100 || got2.Content.ImageContent.Height != 200 {
		t.Errorf("got Width=%d Height=%d, want 100 200", got2.Content.ImageContent.Width, got2.Content.ImageContent.Height)
	}
}
