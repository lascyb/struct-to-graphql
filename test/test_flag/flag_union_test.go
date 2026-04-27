package test_flag

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// FlagUnionText union 分支：文本
type FlagUnionText struct {
	Text string `graphql:"text"`
}

// FlagUnionImage union 分支：图片
type FlagUnionImage struct {
	URL string `graphql:"url"`
}

// FlagUnionContent 使用 union flag
type FlagUnionContent struct {
	Typename string `graphql:"__typename,union"`
	FlagUnionText
	FlagUnionImage
}

// FlagUnionQuery union 查询入口
type FlagUnionQuery struct {
	Content FlagUnionContent `graphql:"content"`
}

// FlagUnionAnonymousContent 匿名 union 分支（未指定 type）应报错
type FlagUnionAnonymousContent struct {
	Typename string `graphql:"__typename,union"`
	Bad      struct {
		Name string `graphql:"name"`
	}
}

type FlagUnionAnonymousQuery struct {
	Content FlagUnionAnonymousContent `graphql:"content"`
}

func TestFlagUnion_GenerateInlineFragments(t *testing.T) {
	exec, err := graphql.Marshal(FlagUnionQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	query, err := exec.Query("FlagUnion")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !strings.Contains(query, "__typename") {
		t.Fatalf("missing __typename: %s", query)
	}
	if !strings.Contains(query, "... on FlagUnionText") {
		t.Fatalf("missing union branch FlagUnionText: %s", query)
	}
	if !strings.Contains(query, "... on FlagUnionImage") {
		t.Fatalf("missing union branch FlagUnionImage: %s", query)
	}
}

func TestFlagUnion_AnonymousMemberShouldFail(t *testing.T) {
	_, err := graphql.Marshal(FlagUnionAnonymousQuery{})
	if err == nil {
		t.Fatal("expected error for anonymous union member, got nil")
	}
	if !strings.Contains(err.Error(), "embedded struct field") && !strings.Contains(err.Error(), "named struct type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
