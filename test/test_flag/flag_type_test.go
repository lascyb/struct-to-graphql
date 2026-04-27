package test_flag

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// FlagTypeText 命名 union 分支（可通过 type flag 覆盖输出类型名）
type FlagTypeText struct {
	Text string `graphql:"text"`
}

// FlagTypeImage 命名 union 分支（可通过 type flag 覆盖输出类型名）
type FlagTypeImage struct {
	URL string `graphql:"url"`
}

// FlagTypeNamedUnion type flag 用于自定义 union 输出类型名
type FlagTypeNamedUnion struct {
	Typename      string `graphql:"__typename,union"`
	FlagTypeText  `graphql:",type=TextData"`
	FlagTypeImage `graphql:",type=ImageData"`
}

type FlagTypeQuery struct {
	Content FlagTypeNamedUnion `graphql:"content"`
}

func TestFlagType_UnionNamedStructWithCustomTypeName(t *testing.T) {
	exec, err := graphql.Marshal(FlagTypeQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	query, err := exec.Query("FlagType")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	t.Logf("Query: %s", query)
	if !strings.Contains(query, "... on TextData") {
		t.Fatalf("missing custom type TextData: %s", query)
	}
	if !strings.Contains(query, "... on ImageData") {
		t.Fatalf("missing custom type ImageData: %s", query)
	}
}

// FlagTypeAnonymousUnion 匿名 union 分支，即使设置 type flag 也应报错
type FlagTypeAnonymousUnion struct {
	Typename string `graphql:"__typename,union"`
	BadData  struct {
		Text string `graphql:"text"`
	} `graphql:",type=TextData"`
}

type FlagTypeAnonymousQuery struct {
	Content FlagTypeAnonymousUnion `graphql:"content"`
}

func TestFlagType_UnionAnonymousStructShouldFail(t *testing.T) {
	_, err := graphql.Marshal(FlagTypeAnonymousQuery{})
	if err == nil {
		t.Fatal("expected error for anonymous union member even with type flag, got nil")
	}
	if !strings.Contains(err.Error(), "embedded struct field") {
		t.Fatalf("unexpected error: %v", err)
	}
}
