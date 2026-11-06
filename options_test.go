package graphql_test

import (
	"strings"
	"sync"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 以下测试会修改全局缩进，不能并行，并在结束时恢复默认值

type OptIndentUser struct {
	ID string `json:"id" graphql:"id"`
}

type optIndentQuery struct {
	Viewer struct {
		Friend OptIndentUser `json:"friend" graphql:"friend"`
	} `json:"viewer" graphql:"viewer"`
	Owner OptIndentUser `json:"owner" graphql:"owner"`
}

func marshalBody(t testing.TB, opts ...graphql.Option) (string, []*graphql.Fragment) {
	t.Helper()
	g, err := graphql.Marshal(optIndentQuery{}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return g.Body, g.Fragments
}

func resetIndent(t *testing.T) {
	t.Helper()
	graphql.SetIndent("  ")
	t.Cleanup(func() { graphql.SetIndent("  ") })
}

func TestWithIndent(t *testing.T) {
	resetIndent(t)
	inline := func(ind string) string {
		return "{\n" + ind + "viewer{\n" + ind + ind + "friend{\n" + ind + ind + ind + "id\n" + ind + ind + "}\n" + ind + "}\n" +
			ind + "owner{\n" + ind + ind + "id\n" + ind + "}\n}"
	}
	for _, ind := range []string{"\t", "    ", "", "  "} {
		got, frags := marshalBody(t, graphql.WithFragment(false), graphql.WithIndent(ind))
		if got != inline(ind) || len(frags) != 0 {
			t.Errorf("WithIndent(%q): got %q", ind, got)
		}
	}
	// Fragment 的 Body 同样使用 WithIndent 指定的缩进
	_, frags := marshalBody(t, graphql.WithIndent("\t"))
	if len(frags) != 1 || frags[0].Body != "fragment OptIndentUser on OptIndentUser{\n\tid\n}" {
		t.Fatalf("fragment body: %q", frags[0].Body)
	}
	// 不修改全局缩进，后续不带参数的调用仍使用默认两个空格
	if got, _ := marshalBody(t); !strings.Contains(got, "\n  viewer{\n    friend{ ...OptIndentUser }") {
		t.Fatalf("global indent changed: %q", got)
	}
}

// WithIndent 优先于 SetIndent；未传 WithIndent 时使用 SetIndent 的值
func TestWithIndent_OverridesSetIndent(t *testing.T) {
	resetIndent(t)
	graphql.SetIndent("\t")
	if got, _ := marshalBody(t); !strings.HasPrefix(got, "{\n\tviewer{\n\t\tfriend") {
		t.Fatalf("SetIndent not applied: %q", got)
	}
	if got, _ := marshalBody(t, graphql.WithIndent("    ")); !strings.HasPrefix(got, "{\n    viewer{\n        friend") {
		t.Fatalf("WithIndent did not override SetIndent: %q", got)
	}
	if got, _ := marshalBody(t, graphql.WithIndent("")); !strings.HasPrefix(got, "{\nviewer{\nfriend") {
		t.Fatalf("WithIndent(\"\") did not override SetIndent: %q", got)
	}
	// 多次传入以最后一次为准
	if got, _ := marshalBody(t, graphql.WithIndent("    "), graphql.WithIndent(" ")); !strings.HasPrefix(got, "{\n viewer{\n  friend") {
		t.Fatalf("last WithIndent should win: %q", got)
	}
}

// nil Option 会被忽略，不会 panic
func TestMarshal_NilOption(t *testing.T) {
	resetIndent(t)
	want, _ := marshalBody(t)
	got, frags := marshalBody(t, nil)
	if got != want || len(frags) != 1 {
		t.Fatalf("Marshal(v, nil) = %q, want %q", got, want)
	}
	if got, _ := marshalBody(t, nil, graphql.WithIndent("\t"), nil); !strings.HasPrefix(got, "{\n\tviewer") {
		t.Fatalf("options around nil not applied: %q", got)
	}
	var opts []graphql.Option
	if got, _ := marshalBody(t, opts...); got != want {
		t.Fatalf("Marshal(v, nil slice...) = %q", got)
	}
}

// 并发混用 WithFragment、WithIndent 与 SetIndent：-race 下无数据竞争，每次结果只取决于自身参数和开始时的全局缩进
func TestMarshalOptions_Concurrent(t *testing.T) {
	resetIndent(t)
	defTwo, _ := marshalBody(t)
	defTab := strings.ReplaceAll(defTwo, "  ", "\t")
	offFour, _ := marshalBody(t, graphql.WithFragment(false), graphql.WithIndent("    "))
	onTab, _ := marshalBody(t, graphql.WithFragment(true), graphql.WithIndent("\t"))
	offTwo, _ := marshalBody(t, graphql.WithFragment(false))
	offTab := strings.ReplaceAll(offTwo, "  ", "\t")

	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for i := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 100 {
				var got string
				var ok bool
				switch i % 4 {
				case 0:
					if j%2 == 0 {
						graphql.SetIndent("\t")
					} else {
						graphql.SetIndent("  ")
					}
					continue
				case 1:
					got, _ = marshalBody(t, graphql.WithFragment(false), graphql.WithIndent("    "))
					ok = got == offFour
				case 2:
					got, _ = marshalBody(t, graphql.WithFragment(true), graphql.WithIndent("\t"))
					ok = got == onTab
				default:
					if j%2 == 0 {
						got, _ = marshalBody(t)
						ok = got == defTwo || got == defTab
					} else {
						got, _ = marshalBody(t, graphql.WithFragment(false))
						ok = got == offTwo || got == offTab
					}
				}
				if !ok {
					errs <- got
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for got := range errs {
		t.Fatalf("unexpected output under concurrency: %q", got)
	}
}
