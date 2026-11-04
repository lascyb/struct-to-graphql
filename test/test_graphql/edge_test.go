package test_graphql

import (
	"testing"
	"time"

	"github.com/lascyb/struct-to-graphql/test/test_graphql/internal/other"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== 边界情况 ==========

// 未声明 graphql tag 时回退到 json tag（与 encoding/json 一致），graphql:"-" / json:"-" 跳过字段
type TagFallbackQuery struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name,omitempty"`
	OmitEmpty string `json:",omitempty"` // 名字为空：使用 Go 字段名
	NoTag     string
	Secret    string `json:"-"`
	Internal  string `graphql:"-" json:"internal"`
	Override  string `json:"override" graphql:"overridden,alias=override"`
	hidden    string //nolint:unused // 未导出字段不参与生成
}

// map、interface、time.Time（没有导出字段的结构体）均视为标量
type ScalarLikeQuery struct {
	CreatedAt time.Time         `json:"createdAt" graphql:"createdAt"`
	UpdatedAt *time.Time        `json:"updatedAt" graphql:"updatedAt"`
	Metadata  map[string]string `json:"metadata" graphql:"metadata"`
	Payload   any               `json:"payload" graphql:"payload"`
	History   []time.Time       `json:"history" graphql:"history"`
}

// 泛型类型：Fragment 名称为 类型名_类型参数短名，类型条件为去掉类型参数的类型名
type Page[T any] struct {
	Nodes    []T  `json:"nodes" graphql:"nodes"`
	HasMore  bool `json:"hasMore" graphql:"hasMore"`
	PageSize int  `json:"pageSize" graphql:"pageSize"`
}

type Pair[K, V any] struct {
	Key   K `json:"key" graphql:"key"`
	Value V `json:"value" graphql:"value"`
}

type GenericQuery struct {
	Users    Page[UserInfo]               `json:"users" graphql:"users(first:10)"`
	Admins   Page[UserInfo]               `json:"admins" graphql:"admins(first:5)"`
	Tags     Page[*Tag]                   `json:"tags" graphql:"tags"`
	MoreTags Page[*Tag]                   `json:"moreTags" graphql:"moreTags"`
	Pairs    []Pair[string, int]          `json:"pairs" graphql:"pairs"`
	Pairs2   []Pair[string, int]          `json:"pairs2" graphql:"pairs2"`
	Nested   Page[Pair[string, UserInfo]] `json:"nested" graphql:"nested"`
	Nested2  Page[Pair[string, UserInfo]] `json:"nested2" graphql:"nested2"`
}

// 不同包下的同名类型：先完成构建的使用类型名，后者追加包名后缀
type FragmentCollisionQuery struct {
	Mine    UserInfo        `json:"mine" graphql:"mine"`
	Theirs  other.UserInfo  `json:"theirs" graphql:"theirs"`
	Mine2   UserInfo        `json:"mine2" graphql:"mine2"`
	Theirs2 *other.UserInfo `json:"theirs2" graphql:"theirs2"`
}

// 类型名与另一个类型生成的 Fragment 名相同（Page[UserInfo] → Page_UserInfo）
type Page_UserInfo struct {
	Total int `json:"total" graphql:"total"`
}

type FragmentNameClashQuery struct {
	A Page[UserInfo] `json:"a" graphql:"a"`
	B Page[UserInfo] `json:"b" graphql:"b"`
	C Page_UserInfo  `json:"c" graphql:"c"`
	D Page_UserInfo  `json:"d" graphql:"d"`
}

// 同一类型先作为匿名嵌入（平铺）、再作为普通字段使用
type EmbedReuseQuery struct {
	EmbedAddress
	Office EmbedAddress  `json:"office" graphql:"office"`
	Home   *EmbedAddress `json:"home" graphql:"home"`
}

// 普通字段先生成 Fragment，之后的匿名嵌入直接展开为 Fragment 引用
type EmbedAfterFragmentQuery struct {
	Office EmbedAddress `json:"office" graphql:"office"`
	Home   EmbedAddress `json:"home" graphql:"home"`
	Detail struct {
		Name string `json:"name" graphql:"name"`
		EmbedAddress
	} `json:"detail" graphql:"detail"`
}

func TestEdgeOutput(t *testing.T) {
	t.Parallel()
	runOutputCases(t, []outputCase{
		{name: "TagFallback", v: TagFallbackQuery{}, opName: "Fallback", want: `query Fallback {
  id
  full_name
  OmitEmpty
  NoTag
  override:overridden
}`},
		{name: "ScalarLike", v: ScalarLikeQuery{}, opName: "ScalarLike", want: `query ScalarLike {
  createdAt
  updatedAt
  metadata
  payload
  history
}`},
		{name: "generic", v: GenericQuery{}, opName: "Generic"},
		{name: "FragmentCollision", v: FragmentCollisionQuery{}, opName: "Collision", want: `fragment UserInfo on UserInfo{
  id
  name
  email
}
fragment UserInfo_other on UserInfo{
  id
}
query Collision {
  mine{ ...UserInfo }
  theirs{ ...UserInfo_other }
  mine2{ ...UserInfo }
  theirs2{ ...UserInfo_other }
}`},
		{name: "FragmentNameClash", v: FragmentNameClashQuery{}, opName: "Clash", want: `fragment Page_UserInfo on Page{
  nodes{
    id
    name
    email
  }
  hasMore
  pageSize
}
fragment Page_UserInfo_test_graphql on Page_UserInfo{
  total
}
query Clash {
  a{ ...Page_UserInfo }
  b{ ...Page_UserInfo }
  c{ ...Page_UserInfo_test_graphql }
  d{ ...Page_UserInfo_test_graphql }
}`},
		{name: "EmbedReuse", v: EmbedReuseQuery{}, opName: "EmbedReuse", want: `fragment EmbedAddress on EmbedAddress{
  city
  street
}
query EmbedReuse {
  city
  street
  office{ ...EmbedAddress }
  home{ ...EmbedAddress }
}`},
		{name: "EmbedAfterFragment", v: EmbedAfterFragmentQuery{}, opName: "EmbedAfter", want: `fragment EmbedAddress on EmbedAddress{
  city
  street
}
query EmbedAfter {
  office{ ...EmbedAddress }
  home{ ...EmbedAddress }
  detail{
    name
    ...EmbedAddress
  }
}`},
	})
}

func TestEdge_GenericFragmentNames(t *testing.T) {
	t.Parallel()
	exec, _ := testutil.Build(t, GenericQuery{}, testutil.OpQuery, "Generic")
	testutil.AssertFragmentNames(t, exec.Fragments,
		"Page_UserInfo", "UserInfo", "Page_Tag", "Pair_String_Int", "Page_Pair_String_UserInfo")
	for _, f := range exec.Fragments {
		for _, r := range f.Name + f.Type {
			if !(r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
				t.Fatalf("fragment %q on %q contains invalid character %q", f.Name, f.Type, r)
			}
		}
	}
}
