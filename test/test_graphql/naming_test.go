package test_graphql

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/test_graphql/internal/other"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== 自定义 Fragment 名与类型条件：GraphQLType() / GraphQLFragment() 与 type= / fragment= ==========

// 值接收者实现 GraphQLType
type UserDTO struct {
	ID   string `json:"id" graphql:"id"`
	Name string `json:"name" graphql:"name"`
}

func (UserDTO) GraphQLType() string { return "User" }

// 指针接收者实现 GraphQLType；两个 DTO 映射到同一个 Schema 类型
type UserCardDTO struct {
	ID string `json:"id" graphql:"id"`
}

func (*UserCardDTO) GraphQLType() string { return " User " } // 首尾空白会被去除

type UserDetailDTO struct {
	ID    string `json:"id" graphql:"id"`
	Email string `json:"email" graphql:"email"`
}

func (UserDetailDTO) GraphQLType() string { return "User" }

// 只实现 GraphQLFragment（指针接收者）
type NamedFragDTO struct {
	ID string `json:"id" graphql:"id"`
}

func (*NamedFragDTO) GraphQLFragment() string { return "UserFields" }

// 同时实现两者（值接收者）
type BothNamedDTO struct {
	ID string `json:"id" graphql:"id"`
}

func (BothNamedDTO) GraphQLType() string     { return "User" }
func (BothNamedDTO) GraphQLFragment() string { return "UserFields" }

// 返回空 / 全空白视为未声明
type BlankNamedDTO struct {
	ID string `json:"id" graphql:"id"`
}

func (BlankNamedDTO) GraphQLType() string      { return "  " }
func (*BlankNamedDTO) GraphQLFragment() string { return "" }

type NmTypeQuery struct {
	A UserDTO       `json:"a" graphql:"a"`
	B *UserDTO      `json:"b" graphql:"b"`
	C []UserDTO     `json:"c" graphql:"c"`
	D UserCardDTO   `json:"d" graphql:"d"`
	E []UserCardDTO `json:"e" graphql:"e"`
	F UserDetailDTO `json:"f" graphql:"f"`
	G UserDetailDTO `json:"g" graphql:"g"`
}

type NmFragQuery struct {
	A NamedFragDTO  `json:"a" graphql:"a"`
	B NamedFragDTO  `json:"b" graphql:"b"`
	C BothNamedDTO  `json:"c" graphql:"c"`
	D *BothNamedDTO `json:"d" graphql:"d"`
}

type NmBlankQuery struct {
	A BlankNamedDTO `json:"a" graphql:"a"`
	B BlankNamedDTO `json:"b" graphql:"b"`
}

// union 分支：分支结构体实现 GraphQLType，无需 type=；type= 优先于 GraphQLType()
type ImageDTO struct {
	URL string `json:"url" graphql:"url"`
}

func (ImageDTO) GraphQLType() string { return "Image" }

type VideoDTO struct {
	Src string `json:"src" graphql:"src"`
}

func (*VideoDTO) GraphQLType() string { return "Video" }

type NmMedia struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	ImageDTO
	VideoDTO `graphql:",type=Clip"`
}

type NmUnionQuery struct {
	Media NmMedia `json:"media" graphql:"media"`
}

type NmUnionReuseQuery struct {
	Media NmMedia  `json:"media" graphql:"media"`
	Cover ImageDTO `json:"cover" graphql:"cover"`
}

// 实现接口名 GraphQLType 的共享分支：两个 union 的分支 type= 不同也合法
type SharedNode struct {
	ID string `json:"id" graphql:"id"`
}

func (SharedNode) GraphQLType() string { return "Node" }

type NmUnionA struct {
	Typename   string `json:"__typename" graphql:"__typename,union"`
	SharedNode `graphql:",type=BranchA"`
}

type NmUnionB struct {
	Typename   string `json:"__typename" graphql:"__typename,union"`
	SharedNode `graphql:",type=BranchB"`
}

type NmSharedQuery struct {
	A NmUnionA `json:"a" graphql:"a"`
	B NmUnionB `json:"b" graphql:"b"`
}

func TestCustomTypeAndFragmentNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		v         any
		fragments [][2]string // {Name, Type}
		want      string
	}{
		{name: "GraphQLType", v: NmTypeQuery{}, fragments: [][2]string{{"UserDTO", "User"}, {"UserCardDTO", "User"}, {"UserDetailDTO", "User"}}, want: `fragment UserDTO on User{
  id
  name
}
fragment UserCardDTO on User{
  id
}
fragment UserDetailDTO on User{
  id
  email
}
query Q {
  a{ ...UserDTO }
  b{ ...UserDTO }
  c{ ...UserDTO }
  d{ ...UserCardDTO }
  e{ ...UserCardDTO }
  f{ ...UserDetailDTO }
  g{ ...UserDetailDTO }
}`},
		{name: "Blank", v: NmBlankQuery{}, fragments: [][2]string{{"BlankNamedDTO", "BlankNamedDTO"}}, want: `fragment BlankNamedDTO on BlankNamedDTO{
  id
}
query Q {
  a{ ...BlankNamedDTO }
  b{ ...BlankNamedDTO }
}`},
		{name: "UnionBranch", v: NmUnionQuery{}, want: `query Q {
  media{
    __typename
    ... on Image {
      url
    }
    ... on Clip {
      src
    }
  }
}`},
		{name: "UnionBranchReused", v: NmUnionReuseQuery{}, fragments: [][2]string{{"ImageDTO", "Image"}}, want: `fragment ImageDTO on Image{
  url
}
query Q {
  media{
    __typename
    ... on Image { ...ImageDTO }
    ... on Clip {
      src
    }
  }
  cover{ ...ImageDTO }
}`},
		{name: "SharedInterfaceBranch", v: NmSharedQuery{}, fragments: [][2]string{{"SharedNode", "Node"}}, want: `fragment SharedNode on Node{
  id
}
query Q {
  a{
    __typename
    ... on BranchA { ...SharedNode }
  }
  b{
    __typename
    ... on BranchB { ...SharedNode }
  }
}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			exec, got := testutil.BuildWithOptions(t, c.v, testutil.OpQuery, "Q")
			testutil.AssertEqual(t, got, c.want)
			if len(exec.Fragments) != len(c.fragments) {
				t.Fatalf("got %d fragments, want %d:\n%s", len(exec.Fragments), len(c.fragments), got)
			}
			for i, f := range exec.Fragments {
				if f.Name != c.fragments[i][0] || f.Type != c.fragments[i][1] {
					t.Errorf("Fragments[%d] = %s on %s, want %s on %s", i, f.Name, f.Type, c.fragments[i][0], c.fragments[i][1])
				}
			}
		})
	}
}

// GraphQLFragment() 单独实现，以及与 GraphQLType() 同时实现
func TestGraphQLFragment_Output(t *testing.T) {
	t.Parallel()
	type q struct {
		A NamedFragDTO `json:"a" graphql:"a"`
		B NamedFragDTO `json:"b" graphql:"b"`
	}
	_, got := testutil.BuildWithOptions(t, q{}, testutil.OpQuery, "Q")
	testutil.AssertEqual(t, got, `fragment UserFields on NamedFragDTO{
  id
}
query Q {
  a{ ...UserFields }
  b{ ...UserFields }
}`)
	type q2 struct {
		C BothNamedDTO  `json:"c" graphql:"c"`
		D *BothNamedDTO `json:"d" graphql:"d"`
	}
	_, got = testutil.BuildWithOptions(t, q2{}, testutil.OpQuery, "Q")
	testutil.AssertEqual(t, got, `fragment UserFields on User{
  id
}
query Q {
  c{ ...UserFields }
  d{ ...UserFields }
}`)
}

// 自动生成的名称与另一个类型显式声明的名称相同：显式名称保持不变，自动名称追加包名后缀，与字段顺序无关
type UserFields struct {
	Name string `json:"name" graphql:"name"`
}

type NmCollideAutoFirst struct {
	A UserFields   `json:"a" graphql:"a"`
	B UserFields   `json:"b" graphql:"b"`
	C NamedFragDTO `json:"c" graphql:"c"`
	D NamedFragDTO `json:"d" graphql:"d"`
}

type NmCollideExplicitFirst struct {
	C NamedFragDTO `json:"c" graphql:"c"`
	D NamedFragDTO `json:"d" graphql:"d"`
	A UserFields   `json:"a" graphql:"a"`
	B UserFields   `json:"b" graphql:"b"`
}

func TestGraphQLFragment_ExplicitWinsCollision(t *testing.T) {
	t.Parallel()
	for name, v := range map[string]any{"AutoFirst": NmCollideAutoFirst{}, "ExplicitFirst": NmCollideExplicitFirst{}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			exec, got := testutil.BuildWithOptions(t, v, testutil.OpQuery, "Q")
			names := map[string]string{}
			for _, f := range exec.Fragments {
				names[f.Type] = f.Name
			}
			if names["NamedFragDTO"] != "UserFields" || names["UserFields"] != "UserFields_test_graphql" {
				t.Fatalf("unexpected fragment names %v:\n%s", names, got)
			}
			if !strings.Contains(got, "fragment UserFields_test_graphql on UserFields{") || !strings.Contains(got, "fragment UserFields on NamedFragDTO{") {
				t.Fatalf("unexpected output:\n%s", got)
			}
		})
	}
}

// 第三方类型（无法添加方法）：通过字段 tag 的 type= / fragment= 指定
type NmThirdPartyOne struct {
	A other.UserInfo  `json:"a" graphql:"a,type=User,fragment=UserFields"`
	B *other.UserInfo `json:"b" graphql:"b"`
}

type NmThirdPartyBoth struct {
	A other.UserInfo   `json:"a" graphql:"a,type=User,fragment=UserFields"`
	B []other.UserInfo `json:"b" graphql:"b, type = User , fragment = 'UserFields'"`
}

func TestFieldFlagFallback(t *testing.T) {
	t.Parallel()
	want := `fragment UserFields on User{
  id
}
query Q {
  a{ ...UserFields }
  b{ ...UserFields }
}`
	for name, v := range map[string]any{"One": NmThirdPartyOne{}, "Both": NmThirdPartyBoth{}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			exec, got := testutil.BuildWithOptions(t, v, testutil.OpQuery, "Q")
			testutil.AssertEqual(t, got, want)
			if f := exec.Fragments[0]; f.Name != "UserFields" || f.Type != "User" {
				t.Fatalf("fragment = %s on %s", f.Name, f.Type)
			}
		})
	}
}

// type= 与 GraphQLType() 一致时不冲突；union 分支的 type= 与 GraphQLType() 不同也不冲突（见 SharedInterfaceBranch）
func TestFieldFlagConsistentWithGraphQLType(t *testing.T) {
	t.Parallel()
	type q struct {
		A UserDTO `json:"a" graphql:"a,type=User"`
		B UserDTO `json:"b" graphql:"b"`
	}
	exec, _ := testutil.BuildWithOptions(t, q{}, testutil.OpQuery, "Q")
	if f := exec.Fragments[0]; f.Name != "UserDTO" || f.Type != "User" {
		t.Fatalf("fragment = %s on %s", f.Name, f.Type)
	}
}

// 冲突的 type= / fragment=：开启 Fragment 时报错，WithFragment(false) 时合法
type NmConflictType struct {
	A other.UserInfo `json:"a" graphql:"a,type=User"`
	B other.UserInfo `json:"b" graphql:"b,type=Admin"`
}

type NmConflictFragment struct {
	A other.UserInfo `json:"a" graphql:"a,fragment=F1"`
	B other.UserInfo `json:"b" graphql:"b,fragment=F2"`
}

type NmBranchShared struct {
	ID string `json:"id" graphql:"id"`
}

type NmConflictUnionA struct {
	Typename       string `json:"__typename" graphql:"__typename,union"`
	NmBranchShared `graphql:",type=BranchA"`
}

type NmConflictUnionB struct {
	Typename       string `json:"__typename" graphql:"__typename,union"`
	NmBranchShared `graphql:",type=BranchB"`
}

type NmConflictUnion struct {
	A NmConflictUnionA `json:"a" graphql:"a"`
	B NmConflictUnionB `json:"b" graphql:"b"`
}

func TestConflictingFlags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		v    any
		want []string
		off  string
	}{
		{"Type", NmConflictType{}, []string{"conflicting type= flags on reused struct other.UserInfo", "field NmConflictType.A has type=User", "field NmConflictType.B has type=Admin", "implement GraphQLType()", "WithFragment(false)"},
			"query Q {\n  a{\n    id\n  }\n  b{\n    id\n  }\n}"},
		{"Fragment", NmConflictFragment{}, []string{"conflicting fragment= flags", "fragment=F1", "fragment=F2", "implement GraphQLFragment()"},
			"query Q {\n  a{\n    id\n  }\n  b{\n    id\n  }\n}"},
		{"UnionBranches", NmConflictUnion{}, []string{"conflicting type= flags on reused struct test_graphql.NmBranchShared", "NmConflictUnionA.NmBranchShared has type=BranchA"},
			"query Q {\n  a{\n    __typename\n    ... on BranchA {\n      id\n    }\n  }\n  b{\n    __typename\n    ... on BranchB {\n      id\n    }\n  }\n}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertMarshalError(t, c.v, c.want...)
			_, got := testutil.BuildWithOptions(t, c.v, testutil.OpQuery, "Q", graphql.WithFragment(false))
			testutil.AssertEqual(t, got, c.off)
		})
	}
}

// 非 union 字段上的 type= 与结构体 GraphQLType() 不一致
func TestFieldFlagConflictsWithGraphQLType(t *testing.T) {
	t.Parallel()
	type q struct {
		A UserDTO `json:"a" graphql:"a"`
		B UserDTO `json:"b" graphql:"b,type=Member"`
	}
	testutil.AssertMarshalError(t, q{}, "field q.B sets type=Member, which conflicts with test_graphql.UserDTO.GraphQLType() = \"User\"")
	if _, err := graphql.Marshal(q{}, graphql.WithFragment(false)); err != nil {
		t.Fatalf("WithFragment(false) should not check fragment type: %v", err)
	}
}

// 两个类型显式声明同一个 Fragment 名
type DupFragA struct {
	ID string `json:"id" graphql:"id"`
}

func (DupFragA) GraphQLFragment() string { return "Dup" }

type NmDupFragment struct {
	A  DupFragA       `json:"a" graphql:"a"`
	A2 DupFragA       `json:"a2" graphql:"a2"`
	B  other.UserInfo `json:"b" graphql:"b,fragment=Dup"`
	B2 other.UserInfo `json:"b2" graphql:"b2"`
}

func TestDuplicateExplicitFragmentName(t *testing.T) {
	t.Parallel()
	testutil.AssertMarshalError(t, NmDupFragment{}, `fragment name "Dup" is declared by both test_graphql.DupFragA and other.UserInfo`)
	// 两个结构体的 GraphQLFragment() 都返回 UserFields
	testutil.AssertMarshalError(t, NmFragQuery{}, `fragment name "UserFields" is declared by both test_graphql.NamedFragDTO and test_graphql.BothNamedDTO`)
	for _, v := range []any{NmDupFragment{}, NmFragQuery{}} {
		if _, err := graphql.Marshal(v, graphql.WithFragment(false)); err != nil {
			t.Fatal(err)
		}
	}
}

// 非法名称
type BadFragOn struct {
	ID string `graphql:"id"`
}

func (BadFragOn) GraphQLFragment() string { return "on" }

type BadFragDash struct {
	ID string `graphql:"id"`
}

func (*BadFragDash) GraphQLFragment() string { return "bad-name" }

type BadFragDigit struct {
	ID string `graphql:"id"`
}

func (BadFragDigit) GraphQLFragment() string { return "123" }

type BadType struct {
	ID string `graphql:"id"`
}

func (BadType) GraphQLType() string { return "bad-type" }

func TestInvalidNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		v    any
		want []string
	}{
		{"FragmentOn", struct{ A BadFragOn }{}, []string{"test_graphql.BadFragOn.GraphQLFragment()", `"on" is not a valid GraphQL fragment name`}},
		{"FragmentDash", struct{ A BadFragDash }{}, []string{"test_graphql.BadFragDash.GraphQLFragment()", `"bad-name"`}},
		{"FragmentDigit", struct{ A BadFragDigit }{}, []string{`"123" is not a valid GraphQL fragment name`}},
		{"Type", struct{ A BadType }{}, []string{"test_graphql.BadType.GraphQLType()", `"bad-type" is not a valid GraphQL type name`}},
		{"TypeFlag", struct {
			A UserFields `graphql:"a,type='bad-type'"`
		}{}, []string{"field [A]: type flag", `"bad-type" is not a valid GraphQL type name`}},
		{"FragmentFlagOn", struct {
			A UserFields `graphql:"a,fragment='on'"`
		}{}, []string{"field [A]: fragment flag", `"on" is not a valid GraphQL fragment name`}},
		{"FragmentFlagIdentOn", struct {
			A UserFields `graphql:"a,fragment=on"`
		}{}, []string{`"on" is not a valid GraphQL fragment name`}},
		{"TypeFlagNotName", struct {
			A UserFields `graphql:"a,type=123"`
		}{}, []string{"field [A]: type flag must be a name"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertMarshalError(t, c.v, c.want...)
			// 名称校验发生在解析阶段，与是否生成 Fragment 无关
			if _, err := graphql.Marshal(c.v, graphql.WithFragment(false)); err == nil {
				t.Fatal("expected error with WithFragment(false)")
			}
		})
	}
}

// 编译期检查：根包导出的接口别名
var (
	_ graphql.TypeNamer     = UserDTO{}
	_ graphql.TypeNamer     = (*UserCardDTO)(nil)
	_ graphql.FragmentNamer = (*NamedFragDTO)(nil)
	_ graphql.FragmentNamer = BothNamedDTO{}
)

// 空白的 type= / fragment= 视为未设置
func TestBlankFieldFlags(t *testing.T) {
	t.Parallel()
	type q struct {
		A other.UserInfo `json:"a" graphql:"a,type='  ',fragment=''"`
		B other.UserInfo `json:"b" graphql:"b"`
	}
	exec, _ := testutil.BuildWithOptions(t, q{}, testutil.OpQuery, "Q")
	if f := exec.Fragments[0]; f.Name != "UserInfo" || f.Type != "UserInfo" {
		t.Fatalf("fragment = %s on %s", f.Name, f.Type)
	}
}

// 泛型结构体作为 union 分支时，... on 使用去掉类型参数的 Go 类型名
type NmBox[T any] struct {
	Value T `json:"value" graphql:"value"`
}

type NmGenericUnion struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	NmBox[int]
}

func TestGenericUnionBranchTypeName(t *testing.T) {
	t.Parallel()
	type q struct {
		U NmGenericUnion `json:"u" graphql:"u"`
	}
	testutil.AssertEqual(t, testutil.Query(t, q{}, "Q"), "query Q {\n  u{\n    __typename\n    ... on NmBox {\n      value\n    }\n  }\n}")
}

// union 分支用 type= 改名且在别处复用：Fragment 的类型条件同样使用 type= 的值
func TestUnionBranchTypeFlagReused(t *testing.T) {
	t.Parallel()
	type union struct {
		Typename   string `json:"__typename" graphql:"__typename,union"`
		UserFields `graphql:",type=Profile"`
	}
	type q struct {
		U    union      `json:"u" graphql:"u"`
		Main UserFields `json:"main" graphql:"main"`
	}
	testutil.AssertEqual(t, testutil.Query(t, q{}, "Q"), `fragment UserFields on Profile{
  name
}
query Q {
  u{
    __typename
    ... on Profile { ...UserFields }
  }
  main{ ...UserFields }
}`)
}
