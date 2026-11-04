package test_graphql

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// ========== Directives（指令） ==========
// 语法：在 tag 中追加 directive='name(arg:value,...)'，可重复声明，按书写顺序输出。
// 参数值与字段参数一致：字面量、$、$name、$:Type、$name:Type（可带默认值）。

func buildQuery(t *testing.T, v any, name string) (*graphql.Graphql, string) {
	t.Helper()
	exec, err := graphql.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	query, err := exec.Query(name)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	t.Logf("Generated Query:\n%s", query)
	return exec, query
}

func assertQuery(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected output.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// 字面量参数：布尔、数字、标识符/字符串
type DirectiveLiteralQuery struct {
	ID      string `json:"id" graphql:"id,directive='include(if:true)'"`
	Name    string `json:"name" graphql:"name,directive='skip(if:false)'"`
	Avatar  string `json:"avatar" graphql:"avatar,directive='cached(ttl:60,scope:PUBLIC)'"`
	Summary string `json:"summary" graphql:"summary,directive='truncate(suffix:\"...\",length:120)'"`
}

func TestDirective_Literal(t *testing.T) {
	_, query := buildQuery(t, DirectiveLiteralQuery{}, "DirectiveLiteral")
	assertQuery(t, query, `query DirectiveLiteral {
  id @include(if:true)
  name @skip(if:false)
  avatar @cached(ttl:60,scope:"PUBLIC")
  summary @truncate(suffix:"...",length:120)
}`)
}

// 变量参数：带类型、匿名占位符（@include/@skip 未声明类型时推断为 Boolean!）、默认值、可选前导 @
type DirectiveVariableQuery struct {
	Email string `json:"email" graphql:"email,directive='include(if:$withEmail:Boolean!)'"`
	Phone string `json:"phone" graphql:"phone,directive='@skip(if:$)'"`
	Bio   string `json:"bio" graphql:"bio,directive='include(if:$withBio:Boolean=true)'"`
	Tags  string `json:"tags" graphql:"tags,directive='cached(ttl:$:Int=30)'"`
}

func TestDirective_Variables(t *testing.T) {
	exec, query := buildQuery(t, DirectiveVariableQuery{}, "DirectiveVariable")
	assertQuery(t, query, `query DirectiveVariable($withEmail:Boolean!,$phone_skip_if:Boolean!,$withBio:Boolean=true,$tags_cached_ttl:Int=30) {
  email @include(if:$withEmail)
  phone @skip(if:$phone_skip_if)
  bio @include(if:$withBio)
  tags @cached(ttl:$tags_cached_ttl)
}`)
	if len(exec.Variables) != 4 {
		t.Fatalf("expected 4 variables, got %d", len(exec.Variables))
	}
}

// 同一字段多个指令 + 别名 + 字段参数 + 嵌套结构体 + 无参指令
type DirectiveCombinedQuery struct {
	User struct {
		ID      string `json:"id" graphql:"id"`
		Friends []struct {
			Name string `json:"name" graphql:"name"`
		} `json:"friends" graphql:"friends(first:$friendsFirst:Int=10, orderBy:NAME),alias=topFriends,directive='include(if:$withFriends:Boolean!)',directive='skip(if:$isGuest:Boolean!)',directive=live"`
	} `json:"user" graphql:"user(id:$id:ID!),alias=me,directive='include(if:$withUser:Boolean!)'"`
}

func TestDirective_CombinedWithAliasArgsAndNested(t *testing.T) {
	_, query := buildQuery(t, DirectiveCombinedQuery{}, "DirectiveCombined")
	assertQuery(t, query, `query DirectiveCombined($id:ID!,$withUser:Boolean!,$friendsFirst:Int=10,$withFriends:Boolean!,$isGuest:Boolean!) {
  me:user(id:$id) @include(if:$withUser){
    id
    topFriends:friends(first:$friendsFirst,orderBy:"NAME") @include(if:$withFriends) @skip(if:$isGuest) @live{
      name
    }
  }
}`)
}

// 指令位于被复用的类型中（生成 Fragment），变量依然进入操作头部的变量定义
type DirectiveProfile struct {
	ID    string `json:"id" graphql:"id"`
	Email string `json:"email" graphql:"email,directive='include(if:$withEmail:Boolean!)'"`
}

type DirectiveFragmentQuery struct {
	Author   DirectiveProfile `json:"author" graphql:"author,directive='skip(if:$hideAuthor:Boolean!)'"`
	Reviewer DirectiveProfile `json:"reviewer" graphql:"reviewer"`
}

func TestDirective_InsideFragment(t *testing.T) {
	exec, query := buildQuery(t, DirectiveFragmentQuery{}, "DirectiveFragment")
	assertQuery(t, query, `fragment Test_graphqlDirectiveProfile on DirectiveProfile{
  id
  email @include(if:$withEmail)
}
query DirectiveFragment($hideAuthor:Boolean!,$withEmail:Boolean!) {
  author @skip(if:$hideAuthor){ ...Test_graphqlDirectiveProfile }
  reviewer{ ...Test_graphqlDirectiveProfile }
}`)
	if len(exec.Fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(exec.Fragments))
	}
}

// union 分支（inline fragment）上的指令
type DirectiveTextBranch struct {
	Text string `json:"text" graphql:"text"`
}

type DirectiveImageBranch struct {
	URL string `json:"url" graphql:"url"`
}

type DirectiveUnionContent struct {
	Typename             string `json:"__typename" graphql:"__typename,union"`
	DirectiveTextBranch  `graphql:",type=Text"`
	DirectiveImageBranch `graphql:",type=Image,directive='include(if:$withImages:Boolean!)'"`
}

type DirectiveUnionQuery struct {
	Content DirectiveUnionContent `json:"content" graphql:"content"`
}

func TestDirective_OnUnionBranch(t *testing.T) {
	_, query := buildQuery(t, DirectiveUnionQuery{}, "DirectiveUnion")
	assertQuery(t, query, `query DirectiveUnion($withImages:Boolean!) {
  content{
    __typename
    ... on Text {
      text
    }
    ... on Image @include(if:$withImages) {
      url
    }
  }
}`)
}

// 同名变量在字段参数与指令参数间复用
type DirectiveSharedVarQuery struct {
	Posts []struct {
		ID string `json:"id" graphql:"id"`
	} `json:"posts" graphql:"posts(draft:$preview:Boolean!),directive='include(if:$preview:Boolean!)'"`
	Drafts []struct {
		ID string `json:"id" graphql:"id"`
	} `json:"drafts" graphql:"drafts,directive='include(if:$preview)'"`
}

func TestDirective_SharedVariable(t *testing.T) {
	exec, query := buildQuery(t, DirectiveSharedVarQuery{}, "DirectiveShared")
	assertQuery(t, query, `query DirectiveShared($preview:Boolean!) {
  posts(draft:$preview) @include(if:$preview){
    id
  }
  drafts @include(if:$preview){
    id
  }
}`)
	if len(exec.Variables) != 1 || len(exec.Variables[0].Paths) != 3 {
		t.Fatalf("expected 1 variable used at 3 paths, got %+v", exec.Variables)
	}
}

// Mutation / Subscription 同样适用
func TestDirective_Subscription(t *testing.T) {
	type Sub struct {
		OrderUpdated struct {
			ID     string `json:"id" graphql:"id"`
			Status string `json:"status" graphql:"status,directive='include(if:$withStatus:Boolean!)'"`
		} `json:"orderUpdated" graphql:"orderUpdated(id:$id:ID!)"`
	}
	exec, err := graphql.Marshal(Sub{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	got, err := exec.Subscription("OnOrderUpdated")
	if err != nil {
		t.Fatalf("Subscription failed: %v", err)
	}
	assertQuery(t, got, `subscription OnOrderUpdated($id:ID!,$withStatus:Boolean!) {
  orderUpdated(id:$id){
    id
    status @include(if:$withStatus)
  }
}`)
}

// 未使用指令时输出与之前完全一致（向后兼容）
func TestDirective_BackwardCompatible(t *testing.T) {
	_, query := buildQuery(t, AliasQuery{}, "NoDirective")
	assertQuery(t, query, `query NoDirective {
  id
  displayName:name
  userEmail:email
}`)
}

// README 中的指令示例，保证文档与实际输出一致
type DirectiveReadmeQuery struct {
	User struct {
		ID      string `graphql:"id"`
		Email   string `graphql:"email,directive='include(if:$withEmail:Boolean!)'"`
		Friends []struct {
			Name string `graphql:"name"`
		} `graphql:"friends(first:10),alias=topFriends,directive='skip(if:$hideFriends)',directive='cached(ttl:60)'"`
	} `graphql:"user(id:$id:ID!),directive='include(if:$withUser:Boolean=true)'"`
}

func TestDirective_ReadmeExample(t *testing.T) {
	_, query := buildQuery(t, DirectiveReadmeQuery{}, "GetUser")
	assertQuery(t, query, `query GetUser($id:ID!,$withUser:Boolean=true,$withEmail:Boolean!,$hideFriends:Boolean!) {
  user(id:$id) @include(if:$withUser){
    id
    email @include(if:$withEmail)
    topFriends:friends(first:10) @skip(if:$hideFriends) @cached(ttl:60){
      name
    }
  }
}`)
}
