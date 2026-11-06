// Package testutil 提供各测试包共用的断言与构建辅助函数。
package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// update 为 true 时 AssertGolden 会用实际输出覆盖 testdata 下的 golden 文件：
//
//	go test ./test/test_graphql -update
var update = flag.Bool("update", false, "用实际输出更新 testdata/*.golden 文件")

// Op 操作类型
type Op string

const (
	OpQuery        Op = "query"
	OpMutation     Op = "mutation"
	OpSubscription Op = "subscription"
)

// Build 对 v 执行 Marshal 并组装指定类型的完整操作，任一步失败都会终止测试
func Build(t testing.TB, v any, op Op, name string) (*graphql.Graphql, string) {
	t.Helper()
	return BuildWithOptions(t, v, op, name)
}

// BuildWithOptions 与 Build 相同，但把 opts 透传给 Marshal（如 graphql.WithFragment(false)）
func BuildWithOptions(t testing.TB, v any, op Op, name string, opts ...graphql.Option) (*graphql.Graphql, string) {
	t.Helper()
	exec, err := graphql.Marshal(v, opts...)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var out string
	switch op {
	case OpMutation:
		out, err = exec.Mutation(name)
	case OpSubscription:
		out, err = exec.Subscription(name)
	default:
		out, err = exec.Query(name)
	}
	if err != nil {
		t.Fatalf("%s failed: %v", op, err)
	}
	return exec, out
}

// Query 等价于 Build(t, v, OpQuery, name)，只返回生成的字符串
func Query(t testing.TB, v any, name string) string {
	t.Helper()
	_, out := Build(t, v, OpQuery, name)
	return out
}

// AssertEqual 完整比对生成结果
func AssertEqual(t testing.TB, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected output.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// AssertGolden 与 testdata/<name>.golden 完整比对；带 -update 运行时改为写入
func AssertGolden(t testing.TB, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v (run with -update to create it)", err)
	}
	AssertEqual(t, got, strings.TrimSuffix(string(want), "\n"))
}

// AssertMarshalError 断言 Marshal 失败且错误信息包含 want 中的每一个片段
func AssertMarshalError(t testing.TB, v any, want ...string) error {
	t.Helper()
	_, err := graphql.Marshal(v)
	AssertError(t, err, want...)
	return err
}

// AssertError 断言 err 非空且包含 want 中的每一个片段
func AssertError(t testing.TB, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Fatalf("error %q does not contain %q", err.Error(), w)
		}
	}
}

// AssertVariables 完整比对变量列表（名称、类型、默认值写法、路径）
func AssertVariables(t testing.TB, got []*graphql.Variable, want []graphql.Variable) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d variables, want %d: %s", len(got), len(want), describeVars(got))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Name != w.Name || g.Type != w.Type || g.HasDefault != w.HasDefault || g.Default != w.Default ||
			!reflect.DeepEqual(g.Paths, w.Paths) {
			t.Errorf("Variables[%d] = {Name:%s Type:%s HasDefault:%v Default:%s Paths:%q}, want {Name:%s Type:%s HasDefault:%v Default:%s Paths:%q}",
				i, g.Name, g.Type, g.HasDefault, g.Default, g.Paths, w.Name, w.Type, w.HasDefault, w.Default, w.Paths)
		}
	}
}

func describeVars(vars []*graphql.Variable) string {
	names := make([]string, 0, len(vars))
	for _, v := range vars {
		names = append(names, v.Name)
	}
	return strings.Join(names, ",")
}

// AssertFragmentNames 按顺序比对 Fragment 名称
func AssertFragmentNames(t testing.TB, got []*graphql.Fragment, want ...string) {
	t.Helper()
	names := make([]string, 0, len(got))
	for _, f := range got {
		names = append(names, f.Name)
	}
	if !reflect.DeepEqual(names, want) && !(len(names) == 0 && len(want) == 0) {
		t.Fatalf("fragments = %q, want %q", names, want)
	}
}

// ========== 响应 key 与 json tag 一致性检查 ==========

var (
	quotedRe = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	aliasRe  = regexp.MustCompile(`(?:^|,)\s*alias=([_A-Za-z][_0-9A-Za-z]*)`)
)

// ResponseKeyMismatches 检查 v 的类型树中每个字段：GraphQL 响应中的 key（alias 优先，其次字段名）
// 能否被 encoding/json 写入该字段（json tag 名或 Go 字段名，大小写不敏感，与 encoding/json 一致）。
// 返回所有不一致的描述，空表示一致。
func ResponseKeyMismatches(v any) []string {
	var out []string
	walkResponseKeys(reflect.TypeOf(v), reflect.TypeOf(v).String(), map[reflect.Type]bool{}, &out)
	return out
}

// AssertResponseKeys 断言 ResponseKeyMismatches 为空
func AssertResponseKeys(t testing.TB, v any) {
	t.Helper()
	for _, m := range ResponseKeyMismatches(v) {
		t.Errorf("%T: %s", v, m)
	}
}

func unwrap(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		typ = typ.Elem()
	}
	return typ
}

func walkResponseKeys(typ reflect.Type, path string, visiting map[reflect.Type]bool, out *[]string) {
	typ = unwrap(typ)
	if typ.Kind() != reflect.Struct || visiting[typ] {
		return
	}
	visiting[typ] = true
	defer delete(visiting, typ)
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		gqlTag, hasGQL := f.Tag.Lookup("graphql")
		jsonTag, hasJSON := f.Tag.Lookup("json")
		jsonName, _, _ := strings.Cut(jsonTag, ",")
		if gqlTag == "-" || (!hasGQL && jsonTag == "-") {
			continue
		}
		if f.Anonymous && (!hasJSON || jsonName == "") {
			// 匿名嵌入：GraphQL 与 encoding/json 都把字段平铺到父级
			walkResponseKeys(f.Type, path, visiting, out)
			continue
		}
		key := f.Name
		if hasGQL {
			stripped := quotedRe.ReplaceAllString(gqlTag, "''")
			if name := strings.TrimSpace(strings.FieldsFunc(stripped+",", func(r rune) bool { return r == '(' || r == ',' })[0]); name != "" && !strings.HasPrefix(stripped, ",") && !strings.HasPrefix(stripped, "(") {
				key = name
			}
			if m := aliasRe.FindStringSubmatch(stripped); m != nil {
				key = m[1]
			}
		} else if jsonName != "" {
			key = jsonName
		}
		if jsonTag == "-" {
			*out = append(*out, path+"."+f.Name+": graphql 查询了 "+key+" 但 json:\"-\" 会丢弃该字段")
			continue
		}
		decodeKey := f.Name
		if jsonName != "" {
			decodeKey = jsonName
		}
		if !strings.EqualFold(key, decodeKey) {
			*out = append(*out, path+"."+f.Name+": 响应 key "+key+" 与 json key "+decodeKey+" 不一致")
		}
		walkResponseKeys(f.Type, path+"."+f.Name, visiting, out)
	}
}
