package core

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/lascyb/tagkit"
)

func TestArgKeyOrder(t *testing.T) {
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
		{"ListLiteral", "items(ids:[1,[2,3]],types:[A,'b:c'],next:1)", []string{"ids", "types", "next"}},
		{"VariableTypesAndDefaults", "items(a:$:[Int!]!=[1,2],b:$b:String='x,y',c:$c,d:$)", []string{"a", "b", "c", "d"}},
		{"WithFlags", "items(first:1),alias=list,directive='include(if:$x)'", []string{"first"}},
		{"FlagBeforeCall", "alias=list,items(first:1)", []string{"first"}},
		{"NoCommaBetweenArgs", "items(a:$a:Int b:2)", []string{"a", "b"}},
		{"Duplicate", "items(a:1,a:2)", []string{"a", "a"}},
		{"Unclosed", "items(a:1", []string{"a"}},
		{"MissingColon", "items(a 1)", []string{}},
		{"MissingValue", "items(a:)", []string{}},
		{"BadDefault", "items(a:$a:Int=,b:1)", []string{}},
		{"UnclosedList", "items(a:[1,2", []string{}},
		{"BadListType", "items(a:$a:[Int)", []string{"a"}},
		{"BadListElement", "items(a:[1,:],b:1)", []string{}},
		{"BadType", "items(a:$a:!)", []string{"a"}},
		{"LeadingComma", "items(,a:1)", []string{"a"}},
		{"NotIdentKey", "items(1:2)", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := argKeyOrder(c.raw); !slices.Equal(got, c.want) {
				t.Fatalf("argKeyOrder(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestScanCallArgs_Values(t *testing.T) {
	t.Parallel()
	args := scanCallArgs(`f(s:'a\'b\\c\d',e:RED,n:null,i:-3,f:1.5,b:true,l:[A,'x',1,[2]],v:$v:[ID!]!=[a,'b'],w:$:Int)`)
	render := map[string]string{}
	for _, a := range args {
		switch {
		case a.IsVariable:
			render[a.Key] = "$" + a.VarType
			if a.Default != nil {
				render[a.Key] += "=" + a.Default.render(isStringLikeType(a.VarType))
			}
		default:
			render[a.Key] = a.Value.render(false)
		}
	}
	want := map[string]string{
		"s": `"a'b\\c\\d"`,
		"e": "RED",
		"n": "null",
		"i": "-3",
		"f": "1.5",
		"b": "true",
		"l": `[A,"x",1,[2]]`,
		"v": `$[ID!]!=["a","b"]`,
		"w": "$Int",
	}
	if !reflect.DeepEqual(render, want) {
		t.Fatalf("got  %v\nwant %v", render, want)
	}
}

// tagkit 的词法分析器在以反斜杠结尾的未闭合字符串上会 panic，这里应转换为错误
func TestParseTagValue_RecoversPanic(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`"\`, `f(a:'\`} {
		if _, err := parseTagValue(raw); err == nil || !strings.Contains(err.Error(), "invalid tag") {
			t.Errorf("parseTagValue(%q) err = %v, want invalid tag error", raw, err)
		}
		if got := scanCallArgs(raw); len(got) != 0 {
			t.Errorf("scanCallArgs(%q) = %v, want empty", raw, got)
		}
	}
}

func TestScanType(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"f(a:$:Int)":       "Int",
		"f(a:$:Int!)":      "Int!",
		"f(a:$:[Int])":     "[Int]",
		"f(a:$:[Int!]!)":   "[Int!]!",
		"f(a:$:[[Int]!]!)": "[[Int]!]!",
		"f(a:$:[Int!!])":   "[Int!]", // 与 tagkit 一致：重复的 ! 只保留一个
		"f(a:$:[Int)":      "Unknown",
		"f(a:$:!)":         "Unknown",
	}
	for raw, want := range cases {
		args := scanCallArgs(raw)
		if len(args) != 1 || args[0].VarType != want {
			t.Errorf("scanCallArgs(%q) = %+v, want VarType %q", raw, args, want)
		}
	}
}

func TestOrderedArgKeys(t *testing.T) {
	t.Parallel()
	args := map[string]int{"b": 1, "a": 2, "z": 3, "c": 4}
	// order 中缺失的参数按字典序追加；order 中多余或重复的名字被忽略
	got := orderedArgKeys([]string{"z", "missing", "b", "z"}, args)
	if want := []string{"z", "b", "a", "c"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
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

func TestUnescapeTagString(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		`plain`:    `plain`,
		`a\'b`:     `a'b`,
		`a\"b`:     `a"b`,
		`a\\b`:     `a\b`,
		`a\nb`:     `a\nb`, // 其他转义保持原样
		`trail\`:   `trail\`,
		`\\\'\"\x`: `\'"\x`,
	}
	for in, want := range cases {
		if got := unescapeTagString(in); got != want {
			t.Errorf("unescapeTagString(%q) = %q, want %q", in, got, want)
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

func TestNewArgs(t *testing.T) {
	t.Parallel()
	parse := func(raw string) (map[string]*Arg, []string, error) {
		tv, err := tagkit.ParseTagValue(raw)
		if err != nil {
			t.Fatalf("tagkit: %v", err)
		}
		return newArgs(raw, tv.Args)
	}

	args, keys, err := parse(`f(s:' padded ',e:ENUM,l:[1,2],v:$v:String=bare,w:$w:Sort=ASC)`)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"s", "e", "l", "v", "w"}; !slices.Equal(keys, want) {
		t.Fatalf("keys = %q, want %q", keys, want)
	}
	got := map[string]string{}
	for k, a := range args {
		got[k] = a.Literal + "|" + a.DefaultLiteral
	}
	want := map[string]string{
		"s": `"padded"|`, // 字符串首尾空白被去除（兼容旧版本）
		"e": "ENUM|",
		"l": "[1,2]|",  // tagkit 会丢弃列表参数，由扫描结果补回
		"v": `|"bare"`, // String 类型变量的裸标识符默认值按字符串输出
		"w": "|ASC",    // 其他类型（枚举）的裸标识符默认值原样输出
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
	if args["s"].Value != "padded" {
		t.Fatalf("Value = %#v, want trimmed string", args["s"].Value)
	}

	if _, _, err := parse(`f(a:1,a:2)`); err == nil || !strings.Contains(err.Error(), "参数 a 重复声明") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if _, _, err := parse(`f(a:'   ')`); err == nil || !strings.Contains(err.Error(), "参数值不能定义为空") {
		t.Fatalf("expected empty error, got %v", err)
	}
	// tagkit 解析出的参数在原文中找不到（防御性分支）
	if _, _, err := newArgs("", map[string]tagkit.ArgValue{"x": {Type: "literal", Value: "1"}}); err == nil ||
		!strings.Contains(err.Error(), "参数 x 无法从 tag 原文中定位") {
		t.Fatalf("expected locate error, got %v", err)
	}
}

// FuzzArgKeyOrder 校验扫描器与 tagkit 的语法保持一致：
//   - 任意输入都不能 panic；
//   - tagkit 解析成功时，tagkit 得到的每个参数都能在扫描结果中找到（列表参数会被 tagkit 丢弃，因此只要求包含关系）；
//   - 扫描到的字面量种类与 tagkit 一致（变量 / 字面量）。
func FuzzArgKeyOrder(f *testing.F) {
	for _, seed := range []string{
		"id",
		"items(first:10,after:$after:String,query:'x')",
		"items(q:'a(b,c:d)',r:\"x,y)\",s:1)",
		"items(ids:[1,[2,3]],types:[A,'b:c'],next:1)",
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
		scanned := scanCallArgs(raw)
		keys := argKeyOrder(raw)
		if len(keys) != len(scanned) {
			t.Fatalf("argKeyOrder and scanCallArgs disagree: %q vs %d", keys, len(scanned))
		}
		for _, a := range scanned {
			if a.IsVariable == (a.Value != nil) {
				t.Fatalf("arg %q: IsVariable=%v but Value=%v", a.Key, a.IsVariable, a.Value)
			}
			if a.Value != nil {
				_ = a.Value.render(false)
			}
		}
		tv, err := parseTagValue(raw)
		if err != nil {
			return
		}
		byKey := map[string]scannedArg{}
		for _, a := range scanned {
			byKey[a.Key] = a
		}
		for name, v := range tv.Args {
			a, ok := byKey[name]
			if !ok {
				t.Fatalf("tagkit arg %q not found by scanner in %q (scanned %q)", name, raw, keys)
			}
			if (v.Type == "variable") != a.IsVariable {
				t.Fatalf("arg %q: tagkit type %s, scanner IsVariable=%v (raw %q)", name, v.Type, a.IsVariable, raw)
			}
			if v.Type == "variable" && v.HasDefault != (a.Default != nil) {
				t.Fatalf("arg %q: tagkit HasDefault=%v, scanner default=%v (raw %q)", name, v.HasDefault, a.Default, raw)
			}
		}
	})
}
