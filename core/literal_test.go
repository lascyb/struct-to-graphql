package core

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
)

func mustParseTag(t testing.TB, raw string) *tagkit.TagValue {
	t.Helper()
	tv, err := tagkit.ParseTagValue(raw)
	if err != nil {
		t.Fatalf("tagkit.ParseTagValue(%q): %v", raw, err)
	}
	return tv
}

// 参数名按 tag 原文中的书写顺序返回
func TestNewArgs_Order(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"NoParens", "id", []string{}},
		{"EmptyParens", "items()", []string{}},
		{"Simple", "items(first:10,after:$after:String,query:'x')", []string{"first", "after", "query"}},
		{"Spaces", "items( b : 1 , a : 2 )", []string{"b", "a"}},
		{"Nameless", "(z:1,y:2)", []string{"z", "y"}},
		{"QuotedParensAndCommas", "items(q:'a(b,c:d)',r:\"x,y)\",s:1)", []string{"q", "r", "s"}},
		{"ListLiteral", "items(ids:[[1],[2,3]],types:[A,'b:c'],next:1)", []string{"ids", "types", "next"}},
		{"VariableTypesAndDefaults", "items(a:$:[Int!]!=[1,2],b:$b:String='x,y',c:$c,d:$)", []string{"a", "b", "c", "d"}},
		{"WithFlags", "items(first:1),alias=list,directive='include(if:$x)'", []string{"first"}},
		{"FlagBeforeCall", "alias=list,items(first:1)", []string{"first"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			args, keys, err := newArgs(mustParseTag(t, c.raw))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(keys, c.want) {
				t.Fatalf("keys of %q = %q, want %q", c.raw, keys, c.want)
			}
			if len(args) != len(keys) {
				t.Fatalf("len(args) = %d, len(keys) = %d", len(args), len(keys))
			}
		})
	}
}

// 字面量与变量默认值的 GraphQL 写法
func TestNewArgs_Render(t *testing.T) {
	t.Parallel()
	args, _, err := newArgs(mustParseTag(t, `f(s:'a\'b\\c\d',e:RED,n:null,i:-3,f:1.5,x:1.5e3,b:true,l:[A,'x',null],m:[[1],[2,-3]],v:$v:[ID!]!=[a,'b'],w:$:Int,d:$d:String=bare,o:$o:Sort=ASC,z:$z:String=null)`))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for k, a := range args {
		if a.Type == tagkit.ArgTypeVariable {
			got[k] = "$" + a.GraphQLType + "=" + a.DefaultLiteral
		} else {
			got[k] = a.Literal
		}
	}
	want := map[string]string{
		"s": `"a'b\\c\\d"`, // \' 与 \\ 被还原，其余反斜杠原样保留
		"e": "RED",
		"n": "null",
		"i": "-3",
		"f": "1.5",
		"x": "1.5e3",
		"b": "true",
		"l": `[A,"x",null]`,
		"m": "[[1],[2,-3]]",
		"v": `$[ID!]!=["a","b"]`, // String/ID 类型变量的裸标识符默认值按字符串输出
		"w": "$Int=",
		"d": `$String="bare"`,
		"o": "$Sort=ASC", // 其他类型（枚举）的裸标识符默认值原样输出
		"z": "$String=null",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
}

func TestNewArgs(t *testing.T) {
	t.Parallel()
	args, keys, err := newArgs(mustParseTag(t, `f(s:' padded ',e:ENUM,l:[1,2])`))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"s", "e", "l"}; !slices.Equal(keys, want) {
		t.Fatalf("keys = %q, want %q", keys, want)
	}
	// 字符串首尾空白被去除（兼容旧版本）
	if args["s"].Literal != `"padded"` || args["s"].Value != "padded" {
		t.Fatalf("s = %q / %#v, want trimmed string", args["s"].Literal, args["s"].Value)
	}
	if !reflect.DeepEqual(args["l"].Value, []interface{}{1, 2}) {
		t.Fatalf("l.Value = %#v", args["l"].Value)
	}
	if _, _, err := newArgs(mustParseTag(t, `f(a:'   ')`)); err == nil || !strings.Contains(err.Error(), "参数值不能定义为空") {
		t.Fatalf("expected empty error, got %v", err)
	}
	// 无字段调用时没有参数
	args, keys, err = newArgs(mustParseTag(t, `,inline`))
	if err != nil || len(args) != 0 || len(keys) != 0 {
		t.Fatalf("got %v %v %v", args, keys, err)
	}
}

// 非法 tag 由 tagkit 报告为 *tagkit.SyntaxError，且不会 panic
func TestParseField_SyntaxErrors(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`"\`, `f(a:'\`, `f(a:1,a:2)`, `f(a:$a:Int=x)`, `f(a:-)`, `f(a:1 b:2)`} {
		field := reflect.StructField{Name: "A", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`graphql:` + strconv.Quote(raw))}
		_, err := NewParser().ParseField(field)
		var se *tagkit.SyntaxError
		if !errors.As(err, &se) {
			t.Errorf("ParseField(%q) err = %v, want *tagkit.SyntaxError", raw, err)
			continue
		}
		if !strings.HasPrefix(err.Error(), "field [A]: ") {
			t.Errorf("ParseField(%q) err = %v, want field prefix", raw, err)
		}
	}
}

func TestQuoteGraphQLString(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":             `""`,
		"plain":        `"plain"`,
		`a"b`:          `"a\"b"`,
		`a\b`:          `"a\\b"`,
		"a\nb\rc\td":   `"a\nb\rc\td"`,
		"\b\f":         `"\b\f"`,
		"\x00\x1f\x7f": `"\u0000\u001f\u007f"`,
		"中文 emoji 😀":   `"中文 emoji 😀"`,
	}
	for in, want := range cases {
		if got := quoteGraphQLString(in); got != want {
			t.Errorf("quoteGraphQLString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestIsStringLikeType(t *testing.T) {
	t.Parallel()
	for typ, want := range map[string]bool{
		"String": true, "String!": true, "ID": true, "[ID!]!": true, "[[String]]": true,
		"Int": false, "SortKey": false, "": false, "Strings": false,
	} {
		if got := isStringLikeType(typ); got != want {
			t.Errorf("isStringLikeType(%q) = %v, want %v", typ, got, want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{"a\"b\n", `"a\"b\n"`},
		{true, "true"},
		{false, "false"},
		{42, "42"},
		{1.5, "1.5"},
		{[]interface{}{1, "x", nil, []interface{}{true}}, `[1,"x",null,[true]]`},
	}
	for _, c := range cases {
		if got := FormatValue(c.in); got != c.want {
			t.Errorf("FormatValue(%#v) = %s, want %s", c.in, got, c.want)
		}
	}
}

// FuzzNewArgs 校验：任意输入都不会 panic；tagkit 解析成功时，newArgs 的参数顺序与 tagkit 一致，
// 每个字面量都有非空的 GraphQL 写法，唯一允许的错误是空字符串参数。
func FuzzNewArgs(f *testing.F) {
	for _, seed := range []string{
		"id",
		"items(first:10,after:$after:String,query:'x')",
		"items(q:'a(b,c:d)',r:\"x,y)\",s:1)",
		"items(ids:[[1],[2,3]],types:[A,'b:c'],next:1)",
		"items(a:$:[Int!]!=[1,2],b:$b:String='x,y',c:$c,d:$)",
		"items(first:1),alias=list,directive='include(if:$x)'",
		"alias=list,items(first:1)",
		"items(a:$a:Int b:2)",
		"(z:-1,y:null)",
		`items(s:'it\'s',t:"a\"b")`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		tv, err := tagkit.ParseTagValue(raw)
		if err != nil {
			var se *tagkit.SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("tagkit error %T is not *SyntaxError: %v", err, err)
			}
			return
		}
		args, keys, err := newArgs(tv)
		if err != nil {
			if !strings.Contains(err.Error(), "参数值不能定义为空") {
				t.Fatalf("unexpected error for %q: %v", raw, err)
			}
			return
		}
		if !slices.Equal(keys, tv.ArgKeys) || len(args) != len(keys) {
			t.Fatalf("keys %q, tagkit %q, args %d", keys, tv.ArgKeys, len(args))
		}
		for _, k := range keys {
			a := args[k]
			if (a.Type == tagkit.ArgTypeVariable) == (a.Literal != "") {
				t.Fatalf("arg %q: type %s, literal %q (raw %q)", k, a.Type, a.Literal, raw)
			}
			if a.HasDefault != (a.DefaultLiteral != "") {
				t.Fatalf("arg %q: HasDefault=%v, DefaultLiteral=%q (raw %q)", k, a.HasDefault, a.DefaultLiteral, raw)
			}
		}
	})
}
