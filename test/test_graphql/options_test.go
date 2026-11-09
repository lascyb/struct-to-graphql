package test_graphql

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== Marshal 可选参数：WithFragment / WithIndent ==========

type OptTag struct {
	Name string `json:"name" graphql:"name"`
}

// OptAuthor 被多次引用，内部又引用了同样被多次引用的 OptTag（A -> B 两级复用）
type OptAuthor struct {
	ID       string   `json:"id" graphql:"id"`
	Tags     []OptTag `json:"tags" graphql:"tags(first:3)"`
	MainTag  *OptTag  `json:"mainTag" graphql:"mainTag"`
	Nickname string   `json:"nickname" graphql:"nickname"`
}

type OptNestedQuery struct {
	Author  OptAuthor    `json:"author" graphql:"author"`
	Editors []*OptAuthor `json:"editors" graphql:"editors"`
	Posts   struct {
		Nodes []struct {
			Title  string    `json:"title" graphql:"title"`
			Author OptAuthor `json:"author" graphql:"author"`
		} `json:"nodes" graphql:"nodes"`
	} `json:"posts" graphql:"posts"`
	Tag OptTag `json:"tag" graphql:"tag"`
}

// OptBase 既被匿名嵌入（平铺），又作为普通字段使用
type OptBase struct {
	ID        string `json:"id" graphql:"id"`
	CreatedAt string `json:"createdAt" graphql:"createdAt"`
}

type OptPost struct {
	OptBase
	Title string `json:"title" graphql:"title"`
}

type OptEmbeddedQuery struct {
	Base OptBase `json:"base" graphql:"base"`
	Post OptPost `json:"post" graphql:"post"`
	Feed struct {
		Items []struct {
			OptBase
			Score int `json:"score" graphql:"score"`
		} `json:"items" graphql:"items"`
	} `json:"feed" graphql:"feed"`
}

// union 分支类型 TextContent 在两个 union 字段和一个普通字段中都出现
type OptUnionQuery struct {
	Content Content     `json:"content" graphql:"content"`
	Other   Content     `json:"other" graphql:"other"`
	Pinned  TextContent `json:"pinned" graphql:"pinned"`
}

// 复用结构体内的匿名变量、具名变量与指令
type OptVarAuthor struct {
	ID     string `json:"id" graphql:"id"`
	Avatar string `json:"avatar" graphql:"avatar(size:$:Int=64)"`
	Posts  []struct {
		Title string `json:"title" graphql:"title"`
	} `json:"posts" graphql:"posts(first:$first:Int!, after:$:String)"`
	Email string `json:"email" graphql:"email,directive='include(if:$withEmail)'"`
	Bio   string `json:"bio" graphql:"bio,directive='skip(if:false)'"`
}

type OptVarQuery struct {
	Author OptVarAuthor   `json:"author" graphql:"author(id:$id:ID!)"`
	Others []OptVarAuthor `json:"others" graphql:"others"`
}

func TestWithFragmentFalse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		v    any
		want string
	}{
		{name: "NestedReuse", v: OptNestedQuery{}, want: `query Q {
  author{
    id
    tags(first:3){
      name
    }
    mainTag{
      name
    }
    nickname
  }
  editors{
    id
    tags(first:3){
      name
    }
    mainTag{
      name
    }
    nickname
  }
  posts{
    nodes{
      title
      author{
        id
        tags(first:3){
          name
        }
        mainTag{
          name
        }
        nickname
      }
    }
  }
  tag{
    name
  }
}`},
		{name: "EmbeddedReuse", v: OptEmbeddedQuery{}, want: `query Q {
  base{
    id
    createdAt
  }
  post{
    id
    createdAt
    title
  }
  feed{
    items{
      id
      createdAt
      score
    }
  }
}`},
		{name: "UnionBranchReuse", v: OptUnionQuery{}, want: `query Q {
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
  other{
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
  pinned{
    text
  }
}`},
		{name: "VariablesAndDirectives", v: OptVarQuery{}, want: `query Q($id:ID!,$author_avatar_size:Int=64,$first:Int!,$author_posts_after:String,$withEmail:Boolean!,$others_avatar_size:Int=64,$others_posts_after:String) {
  author(id:$id){
    id
    avatar(size:$author_avatar_size)
    posts(first:$first,after:$author_posts_after){
      title
    }
    email @include(if:$withEmail)
    bio @skip(if:false)
  }
  others{
    id
    avatar(size:$others_avatar_size)
    posts(first:$first,after:$others_posts_after){
      title
    }
    email @include(if:$withEmail)
    bio @skip(if:false)
  }
}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			exec, got := testutil.BuildWithOptions(t, c.v, testutil.OpQuery, "Q", graphql.WithFragment(false))
			testutil.AssertEqual(t, got, c.want)
			if exec.Fragments == nil || len(exec.Fragments) != 0 {
				t.Fatalf("Fragments = %#v, want empty non-nil slice", exec.Fragments)
			}
		})
	}
}

// WithFragment(false) 时每个引用处的匿名变量按实际路径命名，具名变量跨引用去重并记录所有路径
func TestWithFragmentFalse_VariablePaths(t *testing.T) {
	t.Parallel()
	exec, _ := testutil.BuildWithOptions(t, OptVarQuery{}, testutil.OpQuery, "Q", graphql.WithFragment(false))
	testutil.AssertVariables(t, exec.Variables, []graphql.Variable{
		{Name: "$id", Type: "ID!", Paths: []string{"author"}},
		{Name: "$author_avatar_size", Type: "Int", HasDefault: true, Default: "64", Paths: []string{"author/avatar"}},
		{Name: "$first", Type: "Int!", Paths: []string{"author/posts", "others/posts"}},
		{Name: "$author_posts_after", Type: "String", Paths: []string{"author/posts"}},
		{Name: "$withEmail", Type: "Boolean!", Paths: []string{"author/email", "others/email"}},
		{Name: "$others_avatar_size", Type: "Int", HasDefault: true, Default: "64", Paths: []string{"others/avatar"}},
		{Name: "$others_posts_after", Type: "String", Paths: []string{"others/posts"}},
	})
}

// 默认（及 WithFragment(true)）时同一结构体只生成一次 Fragment，匿名变量只按首次出现的路径命名
func TestWithFragmentTrue_VariablesAndFragments(t *testing.T) {
	t.Parallel()
	exec, got := testutil.BuildWithOptions(t, OptVarQuery{}, testutil.OpQuery, "Q", graphql.WithFragment(true))
	testutil.AssertFragmentNames(t, exec.Fragments, "OptVarAuthor")
	testutil.AssertEqual(t, got, `fragment OptVarAuthor on OptVarAuthor{
  id
  avatar(size:$author_avatar_size)
  posts(first:$first,after:$author_posts_after){
    title
  }
  email @include(if:$withEmail)
  bio @skip(if:false)
}
query Q($id:ID!,$author_avatar_size:Int=64,$first:Int!,$author_posts_after:String,$withEmail:Boolean!) {
  author(id:$id){ ...OptVarAuthor }
  others{ ...OptVarAuthor }
}`)
}

// 默认开启 Fragment 时的对照输出：嵌套复用、匿名嵌入复用、union 分支复用
func TestWithFragmentTrue_Reuse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		v         any
		fragments []string
		want      string
	}{
		{name: "NestedReuse", v: OptNestedQuery{}, fragments: []string{"OptAuthor", "OptTag"}, want: `fragment OptAuthor on OptAuthor{
  id
  tags(first:3){ ...OptTag }
  mainTag{ ...OptTag }
  nickname
}
fragment OptTag on OptTag{
  name
}
query Q {
  author{ ...OptAuthor }
  editors{ ...OptAuthor }
  posts{
    nodes{
      title
      author{ ...OptAuthor }
    }
  }
  tag{ ...OptTag }
}`},
		{name: "EmbeddedReuse", v: OptEmbeddedQuery{}, fragments: []string{"OptBase"}, want: `fragment OptBase on OptBase{
  id
  createdAt
}
query Q {
  base{ ...OptBase }
  post{
    ...OptBase
    title
  }
  feed{
    items{
      ...OptBase
      score
    }
  }
}`},
		{name: "UnionBranchReuse", v: OptUnionQuery{}, fragments: []string{"Content", "TextContent"}, want: `fragment Content on Content{
  __typename
  ... on TextContent { ...TextContent }
  ... on ImageContent {
    url
    width
    height
  }
}
fragment TextContent on TextContent{
  text
}
query Q {
  content{ ...Content }
  other{ ...Content }
  pinned{ ...TextContent }
}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			exec, got := testutil.BuildWithOptions(t, c.v, testutil.OpQuery, "Q")
			testutil.AssertEqual(t, got, c.want)
			testutil.AssertFragmentNames(t, exec.Fragments, c.fragments...)
		})
	}
}

// WithFragment(true) 与不传参数的 Marshal(v) 输出完全一致
func TestWithFragmentTrue_MatchesDefault(t *testing.T) {
	t.Parallel()
	for name, v := range map[string]any{
		"Combined": CombinedQuery{}, "FragmentReuse": FragmentQuery{}, "UnionReuse": UnionReuseQuery{},
		"Nested": OptNestedQuery{}, "Embedded": OptEmbeddedQuery{}, "Union": OptUnionQuery{}, "Vars": OptVarQuery{},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, def := testutil.Build(t, v, testutil.OpQuery, "Q")
			_, explicit := testutil.BuildWithOptions(t, v, testutil.OpQuery, "Q", graphql.WithFragment(true))
			testutil.AssertEqual(t, explicit, def)
		})
	}
}

// 同一选项传多次时以最后一次为准
func TestWithFragment_LastWins(t *testing.T) {
	t.Parallel()
	_, off := testutil.BuildWithOptions(t, FragmentQuery{}, testutil.OpQuery, "Q", graphql.WithFragment(false))
	_, got := testutil.BuildWithOptions(t, FragmentQuery{}, testutil.OpQuery, "Q", graphql.WithFragment(true), graphql.WithFragment(false))
	testutil.AssertEqual(t, got, off)
	_, on := testutil.BuildWithOptions(t, FragmentQuery{}, testutil.OpQuery, "Q")
	_, got = testutil.BuildWithOptions(t, FragmentQuery{}, testutil.OpQuery, "Q", graphql.WithFragment(false), nil, graphql.WithFragment(true))
	testutil.AssertEqual(t, got, on)
}

// WithFragment(false) 与 WithIndent 组合：原地展开的结构体使用指定缩进
func TestWithFragmentFalse_WithIndent(t *testing.T) {
	t.Parallel()
	_, got := testutil.BuildWithOptions(t, OptUnionQuery{}, testutil.OpQuery, "Q", graphql.WithFragment(false), graphql.WithIndent("\t"))
	testutil.AssertEqual(t, got, "query Q {\n\tcontent{\n\t\t__typename\n\t\t... on TextContent {\n\t\t\ttext\n\t\t}\n\t\t... on ImageContent {\n\t\t\turl\n\t\t\twidth\n\t\t\theight\n\t\t}\n\t}\n"+
		"\tother{\n\t\t__typename\n\t\t... on TextContent {\n\t\t\ttext\n\t\t}\n\t\t... on ImageContent {\n\t\t\turl\n\t\t\twidth\n\t\t\theight\n\t\t}\n\t}\n\tpinned{\n\t\ttext\n\t}\n}")
}
