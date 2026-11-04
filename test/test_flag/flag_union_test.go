package test_flag

import (
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== union flag 与 type flag ==========
// 匿名结构体等错误分支写法统一在 test/test_error/common_misuse_test.go 中测试。

type FlagUnionText struct {
	Text string `json:"text" graphql:"text"`
}

type FlagUnionImage struct {
	URL string `json:"url" graphql:"url"`
}

// union flag：__typename + 匿名嵌入的命名结构体
type FlagUnionContent struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	FlagUnionText
	*FlagUnionImage // 指针嵌入同样支持
}

type FlagUnionQuery struct {
	Content FlagUnionContent `json:"content" graphql:"content"`
}

// type flag：自定义 "... on" 输出的类型名
type FlagTypeNamedUnion struct {
	Typename       string `json:"__typename" graphql:"__typename,union"`
	FlagUnionText  `graphql:",type=TextData"`
	FlagUnionImage `graphql:",type= ImageData "` // 首尾空白会被去除
}

type FlagTypeQuery struct {
	Content []FlagTypeNamedUnion `json:"content" graphql:"content(first:2)"`
}

func TestFlagUnion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"UnionFlag", FlagUnionQuery{}, `query Flag {
  content{
    __typename
    ... on FlagUnionText {
      text
    }
    ... on FlagUnionImage {
      url
    }
  }
}`},
		{"TypeFlag", FlagTypeQuery{}, `query Flag {
  content(first:2){
    __typename
    ... on TextData {
      text
    }
    ... on ImageData {
      url
    }
  }
}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertEqual(t, testutil.Query(t, c.v, "Flag"), c.want)
			testutil.AssertResponseKeys(t, c.v)
		})
	}
}

// 兼容写法：以逗号开头的单个布尔标记作为字段名（",name"），以及只写 ",union" 的 __typename 字段
type LegacyFlagContent struct {
	Kind string `json:"__typename" graphql:",union"`
	FlagUnionText
}

type LegacyFlagQuery struct {
	Title   string            `json:"title" graphql:",title"`
	Content LegacyFlagContent `json:"content" graphql:"content"`
}

func TestFlagUnion_LegacySyntax(t *testing.T) {
	t.Parallel()
	testutil.AssertEqual(t, testutil.Query(t, LegacyFlagQuery{}, "Legacy"), `query Legacy {
  title
  content{
    __typename
    ... on FlagUnionText {
      text
    }
  }
}`)
}
