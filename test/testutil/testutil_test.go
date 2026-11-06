package testutil

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	graphql "github.com/lascyb/struct-to-graphql"
)

type keysEmbedded struct {
	Views int `json:"views" graphql:"views"`
}

type keysGood struct {
	ID       string    `json:"id" graphql:"id"`
	Title    string    `json:"headline" graphql:"title,alias=headline"`
	Name     string    `graphql:"name"` // 无 json tag：encoding/json 大小写不敏感匹配 Go 字段名
	Items    []keysSub `json:"items" graphql:"items(query:'a,alias=x',first:10)"`
	When     time.Time `json:"when" graphql:"when"`
	Ignored  string    `json:"-" graphql:"-"`
	Fallback string    `json:"fallback"`
	keysEmbedded
}

type keysSub struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	Self     *keysSub
}

type keysBad struct {
	Title   string `json:"title" graphql:"title,alias=headline"`
	Dropped string `json:"-" graphql:"dropped"`
	Nested  struct {
		Name string `json:"fullName" graphql:"name"`
	} `json:"nested" graphql:"nested"`
}

func TestResponseKeyMismatches(t *testing.T) {
	t.Parallel()
	if got := ResponseKeyMismatches(keysGood{}); len(got) != 0 {
		t.Fatalf("expected no mismatches, got %q", got)
	}
	want := []string{
		`testutil.keysBad.Title: 响应 key headline 与 json key title 不一致`,
		`testutil.keysBad.Dropped: graphql 查询了 dropped 但 json:"-" 会丢弃该字段`,
		`testutil.keysBad.Nested.Name: 响应 key name 与 json key fullName 不一致`,
	}
	if got := ResponseKeyMismatches(keysBad{}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// fakeTB 记录断言失败而不终止外层测试，用于验证断言辅助函数本身
type fakeTB struct {
	testing.TB
	failed bool
	msg    string
}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Errorf(format string, args ...any) {
	f.failed = true
	f.msg = fmt.Sprintf(format, args...)
}
func (f *fakeTB) Fatalf(format string, args ...any) {
	f.Errorf(format, args...)
	runtime.Goexit()
}
func (f *fakeTB) Fatal(args ...any) {
	f.failed = true
	f.msg = fmt.Sprint(args...)
	runtime.Goexit()
}

// expectFail 在独立 goroutine 中运行 fn（Fatal 通过 Goexit 结束该 goroutine），断言其报告了失败
func expectFail(t *testing.T, name string, fn func(tb testing.TB), wantMsg string) {
	t.Helper()
	f := &fakeTB{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(f)
	}()
	<-done
	if !f.failed {
		t.Errorf("%s: expected failure, got none", name)
		return
	}
	if !strings.Contains(f.msg, wantMsg) {
		t.Errorf("%s: failure message %q does not contain %q", name, f.msg, wantMsg)
	}
}

type simpleQuery struct {
	ID string `json:"id" graphql:"id(x:$x)"`
}

func TestAssertionsReportFailures(t *testing.T) {
	t.Parallel()
	expectFail(t, "BuildMarshal", func(tb testing.TB) { Build(tb, 1, OpQuery, "Q") }, "Marshal failed")
	expectFail(t, "BuildQuery", func(tb testing.TB) { Build(tb, simpleQuery{}, OpQuery, "Q") }, "query failed")
	expectFail(t, "AssertEqual", func(tb testing.TB) { AssertEqual(tb, "a", "b") }, "--- want ---\nb")
	expectFail(t, "AssertErrorNil", func(tb testing.TB) { AssertError(tb, nil, "x") }, "got nil")
	expectFail(t, "AssertErrorText", func(tb testing.TB) { AssertError(tb, errors.New("boom"), "bang") }, `does not contain "bang"`)
	expectFail(t, "AssertVariablesLen", func(tb testing.TB) {
		AssertVariables(tb, []*graphql.Variable{{Name: "$a"}}, nil)
	}, "got 1 variables, want 0: $a")
	expectFail(t, "AssertVariablesField", func(tb testing.TB) {
		AssertVariables(tb, []*graphql.Variable{{Name: "$a", Type: "Int"}}, []graphql.Variable{{Name: "$a", Type: "String"}})
	}, "Variables[0]")
	expectFail(t, "AssertFragmentNames", func(tb testing.TB) {
		AssertFragmentNames(tb, []*graphql.Fragment{{Name: "A"}}, "B")
	}, `fragments = ["A"], want ["B"]`)
	expectFail(t, "AssertResponseKeys", func(tb testing.TB) { AssertResponseKeys(tb, keysBad{}) }, "fullName")
	expectFail(t, "AssertGoldenMissing", func(tb testing.TB) { AssertGolden(tb, "does-not-exist", "x") }, "run with -update")
}

func TestBuildWithOptions(t *testing.T) {
	t.Parallel()
	exec, got := BuildWithOptions(t, keysEmbedded{}, OpQuery, "Q", graphql.WithIndent("\t"), graphql.WithFragment(false))
	if !strings.HasPrefix(got, "query Q {\n\t") || len(exec.Fragments) != 0 {
		t.Fatalf("options not applied: %q", got)
	}
}

func TestOps(t *testing.T) {
	t.Parallel()
	for op, prefix := range map[Op]string{OpQuery: "query Q", OpMutation: "mutation Q", OpSubscription: "subscription Q", "": "query Q"} {
		if _, got := Build(t, keysEmbedded{}, op, "Q"); !strings.HasPrefix(got, prefix) {
			t.Errorf("op %q: got %q", op, got)
		}
	}
	AssertFragmentNames(t, nil)
}

// -update 模式会写入 golden 文件，之后的比对读取同一文件（修改全局 flag 与工作目录，不能并行）
func TestAssertGoldenUpdate(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := flag.Set("update", "true"); err != nil {
		t.Fatal(err)
	}
	AssertGolden(t, "sample", "line1\nline2")
	if err := flag.Set("update", "false"); err != nil {
		t.Fatal(err)
	}
	AssertGolden(t, "sample", "line1\nline2")
	data, err := os.ReadFile(filepath.Join("testdata", "sample.golden"))
	if err != nil || string(data) != "line1\nline2\n" {
		t.Fatalf("golden file = %q, %v", data, err)
	}
}
