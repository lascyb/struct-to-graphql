package test_graphql

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// ========== 输出稳定性：变量定义、Fragment、字段参数的顺序 ==========

type DetImage struct {
	URL string `json:"url" graphql:"url"`
}

type DetAuthor struct {
	ID     string   `json:"id" graphql:"id"`
	Avatar DetImage `json:"avatar" graphql:"avatar"`
}

type DetComment struct {
	Body   string    `json:"body" graphql:"body"`
	Author DetAuthor `json:"author" graphql:"author"`
}

type DeterministicQuery struct {
	Posts struct {
		Nodes []struct {
			Title    string       `json:"title" graphql:"title"`
			Author   DetAuthor    `json:"author" graphql:"author"`
			Comments []DetComment `json:"comments" graphql:"comments(last:$commentsLast:Int=5, sortKey:CREATED_AT)"`
		} `json:"nodes" graphql:"nodes"`
	} `json:"posts" graphql:"posts(first:$first:Int!, after:$:String, query:$query:String, reverse:true)"`
	Pinned DetComment `json:"pinned" graphql:"pinned(id:$pinnedId:ID!)"`
	Cover  DetImage   `json:"cover" graphql:"cover(size:$size:Int=100)"`
}

// Fragment 名称由 "包名+类型名" 组成，因此带有 Test_graphql 前缀
const deterministicExpected = `fragment DetAuthor on DetAuthor{
  id
  avatar{ ...DetImage }
}
fragment DetImage on DetImage{
  url
}
fragment DetComment on DetComment{
  body
  author{ ...DetAuthor }
}
query Deterministic($first:Int!,$posts_after:String,$query:String,$commentsLast:Int=5,$pinnedId:ID!,$size:Int=100) {
  posts(first:$first,after:$posts_after,query:$query,reverse:true){
    nodes{
      title
      author{ ...DetAuthor }
      comments(last:$commentsLast,sortKey:CREATED_AT){ ...DetComment }
    }
  }
  pinned(id:$pinnedId){ ...DetComment }
  cover(size:$size){ ...DetImage }
}`

func TestDeterministic_ExactOutput(t *testing.T) {
	exec, err := graphql.Marshal(DeterministicQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	query, err := exec.Query("Deterministic")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if query != deterministicExpected {
		t.Fatalf("unexpected output.\n--- got ---\n%s\n--- want ---\n%s", query, deterministicExpected)
	}

	wantVars := []string{"$first", "$posts_after", "$query", "$commentsLast", "$pinnedId", "$size"}
	if len(exec.Variables) != len(wantVars) {
		t.Fatalf("got %d variables, want %d", len(exec.Variables), len(wantVars))
	}
	for i, v := range exec.Variables {
		if v.Name != wantVars[i] {
			t.Errorf("Variables[%d] = %s, want %s", i, v.Name, wantVars[i])
		}
	}
	wantFragments := []string{"DetAuthor", "DetImage", "DetComment"}
	if len(exec.Fragments) != len(wantFragments) {
		t.Fatalf("got %d fragments, want %d", len(exec.Fragments), len(wantFragments))
	}
	for i, f := range exec.Fragments {
		if f.Name != wantFragments[i] {
			t.Errorf("Fragments[%d] = %s, want %s", i, f.Name, wantFragments[i])
		}
	}
}

func TestDeterministic_RepeatedGeneration(t *testing.T) {
	var first string
	for i := 0; i < 50; i++ {
		exec, err := graphql.Marshal(DeterministicQuery{})
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		query, err := exec.Query("Deterministic")
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if i == 0 {
			first = query
			continue
		}
		if query != first {
			t.Fatalf("output differs on run %d.\n--- first ---\n%s\n--- now ---\n%s", i, first, query)
		}
	}
}
