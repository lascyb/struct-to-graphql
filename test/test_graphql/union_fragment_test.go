package test_graphql

import (
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== 联合类型（Union）与 Fragment 复用 ==========

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

// union 分支类型同时在别处使用：分支内改为引用 Fragment
type UnionReuseQuery struct {
	Content Content     `json:"content" graphql:"content"`
	Pinned  TextContent `json:"pinned" graphql:"pinned"`
}

type UserInfo struct {
	ID    string `json:"id" graphql:"id"`
	Name  string `json:"name" graphql:"name"`
	Email string `json:"email" graphql:"email"`
}

type FragmentQuery struct {
	Author   UserInfo   `json:"author" graphql:"author"`
	Reviewer *UserInfo  `json:"reviewer" graphql:"reviewer"`
	Editors  []UserInfo `json:"editors" graphql:"editors(first:3)"`
}

// 综合：别名 + union + Fragment 复用 + 变量默认值 + 匿名嵌入
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
	Title    string       `json:"headline" graphql:"title,alias=headline"` // json tag 必须与 alias 一致
	Content  ContentUnion `json:"content" graphql:"content"`
	Author   AuthorInfo   `json:"author" graphql:"author"`
	Editor   AuthorInfo   `json:"editor" graphql:"editor"`
	Comments []Comment    `json:"comments" graphql:"comments(first:$first:Int=10)"`
	Meta                  // 匿名嵌入：字段平铺到父级
}

func TestUnionFragmentOutput(t *testing.T) {
	t.Parallel()
	runOutputCases(t, []outputCase{
		{name: "Union", v: UnionQuery{}, opName: "UnionTest", want: `query UnionTest {
  id
  content{
    __typename
    ... on TextContent {
      text
    }
    ... on ImageContent {
      url
      width
      height
    }
  }
}`},
		{name: "UnionBranchReused", v: UnionReuseQuery{}, opName: "UnionReuse", want: `fragment TextContent on TextContent{
  text
}
query UnionReuse {
  content{
    __typename
    ... on TextContent { ...TextContent }
    ... on ImageContent {
      url
      width
      height
    }
  }
  pinned{ ...TextContent }
}`},
		{name: "FragmentReuse", v: FragmentQuery{}, opName: "FragmentTest", want: `fragment UserInfo on UserInfo{
  id
  name
  email
}
query FragmentTest {
  author{ ...UserInfo }
  reviewer{ ...UserInfo }
  editors(first:3){ ...UserInfo }
}`},
		{name: "combined", v: CombinedQuery{}, opName: "CombinedTest"},
	})
}

func TestFragment_Metadata(t *testing.T) {
	t.Parallel()
	exec, _ := testutil.Build(t, CombinedQuery{}, testutil.OpQuery, "CombinedTest")
	testutil.AssertFragmentNames(t, exec.Fragments, "AuthorInfo")
	if f := exec.Fragments[0]; f.Type != "AuthorInfo" {
		t.Fatalf("fragment type = %s, want AuthorInfo", f.Type)
	}
}
