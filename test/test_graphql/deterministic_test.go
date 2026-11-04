package test_graphql

import (
	"sync"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== 输出稳定性：变量定义、Fragment、字段参数的顺序 ==========

type DetImage struct {
	URL string `json:"url" graphql:"url(width:$width:Int, height:$height:Int, format:WEBP, crop:CENTER, scale:2)"`
}

type DetAuthor struct {
	ID     string   `json:"id" graphql:"id"`
	Avatar DetImage `json:"avatar" graphql:"avatar"`
}

type DetComment struct {
	Body   string    `json:"body" graphql:"body(format:$bodyFormat:Format=MARKDOWN, truncate:$:Int, locale:'zh-CN')"`
	Author DetAuthor `json:"author" graphql:"author"`
}

type DeterministicQuery struct {
	Posts struct {
		Nodes []struct {
			Title    string       `json:"title" graphql:"title"`
			Author   DetAuthor    `json:"author" graphql:"author"`
			Comments []DetComment `json:"comments" graphql:"comments(last:$commentsLast:Int=5, sortKey:CREATED_AT, reverse:false, query:$:String, before:$before:String)"`
		} `json:"nodes" graphql:"nodes"`
	} `json:"posts" graphql:"posts(first:$first:Int!, after:$:String, query:$query:String, reverse:true, sortKey:$sortKey:PostSortKeys=UPDATED_AT, savedSearchId:$savedSearchId:ID)"`
	Pinned DetComment `json:"pinned" graphql:"pinned(id:$pinnedId:ID!)"`
	Cover  DetImage   `json:"cover" graphql:"cover(size:$size:Int=100)"`
}

func TestDeterministic_ExactOutput(t *testing.T) {
	t.Parallel()
	exec, query := testutil.Build(t, DeterministicQuery{}, testutil.OpQuery, "Deterministic")
	testutil.AssertGolden(t, "deterministic", query)
	testutil.AssertFragmentNames(t, exec.Fragments, "DetAuthor", "DetImage", "DetComment")
	testutil.AssertVariables(t, exec.Variables, []graphql.Variable{
		{Name: "$first", Type: "Int!", Paths: []string{"posts"}},
		{Name: "$posts_after", Type: "String", Paths: []string{"posts"}},
		{Name: "$query", Type: "String", Paths: []string{"posts"}},
		{Name: "$sortKey", Type: "PostSortKeys", HasDefault: true, Default: "UPDATED_AT", Paths: []string{"posts"}},
		{Name: "$savedSearchId", Type: "ID", Paths: []string{"posts"}},
		{Name: "$width", Type: "Int", Paths: []string{"posts/nodes/author/avatar/url"}},  // Fragment 只生成一次，路径记录首次出现的位置,
		{Name: "$height", Type: "Int", Paths: []string{"posts/nodes/author/avatar/url"}}, // Fragment 只生成一次，路径记录首次出现的位置,
		{Name: "$commentsLast", Type: "Int", HasDefault: true, Default: "5", Paths: []string{"posts/nodes/comments"}},
		{Name: "$posts_nodes_comments_query", Type: "String", Paths: []string{"posts/nodes/comments"}},
		{Name: "$before", Type: "String", Paths: []string{"posts/nodes/comments"}},
		{Name: "$bodyFormat", Type: "Format", HasDefault: true, Default: "MARKDOWN", Paths: []string{"posts/nodes/comments/body"}},
		{Name: "$posts_nodes_comments_body_truncate", Type: "Int", Paths: []string{"posts/nodes/comments/body"}},
		{Name: "$pinnedId", Type: "ID!", Paths: []string{"pinned"}},
		{Name: "$size", Type: "Int", HasDefault: true, Default: "100", Paths: []string{"cover"}},
	})
}

// 并发、重复生成的结果必须完全一致
func TestDeterministic_RepeatedGeneration(t *testing.T) {
	t.Parallel()
	want := testutil.Query(t, DeterministicQuery{}, "Deterministic")
	const goroutines, rounds = 8, 25
	var wg sync.WaitGroup
	errs := make(chan string, goroutines*rounds)
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				exec, err := graphql.Marshal(DeterministicQuery{})
				if err != nil {
					errs <- err.Error()
					return
				}
				got, err := exec.Query("Deterministic")
				if err != nil {
					errs <- err.Error()
					return
				}
				if got != want {
					errs <- "output differs:\n" + got
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
}
