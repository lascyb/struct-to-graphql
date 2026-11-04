package core

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

type indentInner struct {
	B string `graphql:"b"`
}

type indentQuery struct {
	A struct {
		Inner indentInner `graphql:"inner"`
	} `graphql:"a"`
}

func buildIndentQuery(t testing.TB) string {
	t.Helper()
	tp, err := NewParser().ParseType(reflect.TypeOf(indentQuery{}))
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewBuilder().Build(tp)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// 修改全局缩进的测试不能并行，并在结束时恢复默认值
func setIndentForTest(t *testing.T, val string) {
	t.Helper()
	old := Indent()
	t.Cleanup(func() { SetIndent(old) })
	SetIndent(val)
}

func TestSetIndent(t *testing.T) {
	if Indent() != DefaultIndent {
		t.Fatalf("default indent = %q, want %q", Indent(), DefaultIndent)
	}
	setIndentForTest(t, "\t")
	want := "{\n\ta{\n\t\tinner{\n\t\t\tb\n\t\t}\n\t}\n}"
	if got := buildIndentQuery(t); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	SetIndent("")
	if got := buildIndentQuery(t); got != "{\na{\ninner{\nb\n}\n}\n}" {
		t.Fatalf("empty indent: got %q", got)
	}
}

// 并发 Marshal 与 SetIndent：-race 下不能有数据竞争，且每次结果都只能使用同一种缩进
func TestSetIndent_ConcurrentWithBuild(t *testing.T) {
	setIndentForTest(t, DefaultIndent)
	two := buildIndentQuery(t)
	tab := strings.ReplaceAll(two, "  ", "\t")

	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 50 {
				if i%4 == 0 {
					if j%2 == 0 {
						SetIndent("\t")
					} else {
						SetIndent(DefaultIndent)
					}
					continue
				}
				if got := buildIndentQuery(t); got != two && got != tab {
					errs <- got
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for got := range errs {
		t.Fatalf("mixed indentation in output: %q", got)
	}
}
