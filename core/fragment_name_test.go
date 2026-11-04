package core

import (
	"reflect"
	"testing"
)

type fragGeneric[T any] struct{ V T }
type fragPair[K, V any] struct {
	K K
	V V
}
type fragItem struct{ ID string }

func TestFragmentBaseName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		typ  reflect.Type
		want string
	}{
		{reflect.TypeOf(fragItem{}), "FragItem"},
		{reflect.TypeOf(fragGeneric[fragItem]{}), "FragGeneric_FragItem"},
		{reflect.TypeOf(fragGeneric[*fragItem]{}), "FragGeneric_FragItem"},
		{reflect.TypeOf(fragGeneric[[]fragItem]{}), "FragGeneric_FragItem"},
		{reflect.TypeOf(fragPair[string, int]{}), "FragPair_String_Int"},
		{reflect.TypeOf(fragGeneric[map[string]fragItem]{}), "FragGeneric_Map_String_FragItem"},
		{reflect.TypeOf(fragGeneric[fragGeneric[fragItem]]{}), "FragGeneric_FragGeneric_FragItem"},
		{reflect.TypeOf(fragGeneric[struct{ A int }]{}), "FragGeneric_Struct_A_Int"},
	}
	for _, c := range cases {
		if got := fragmentBaseName(c.typ.Name()); got != c.want {
			t.Errorf("fragmentBaseName(%q) = %q, want %q", c.typ.Name(), got, c.want)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"Abc":         "Abc",
		"a-b.c":       "a_b_c",
		"__x__":       "x",
		"a__b":        "a_b",
		"1abc":        "F_1abc",
		"":            "Fragment",
		"---":         "Fragment",
		"类型Name":      "Name",
		"struct-to-x": "struct_to_x",
	} {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := upperFirst(""); got != "" {
		t.Errorf("upperFirst(\"\") = %q", got)
	}
	if got := packageName(""); got != "" {
		t.Errorf("packageName(\"\") = %q", got)
	}
	if got := packageName("github.com/a/struct-to-graphql"); got != "struct_to_graphql" {
		t.Errorf("packageName = %q", got)
	}
}

type fragA struct{ ID string }
type fragB struct{ ID string }
type fragC struct{ ID string }

func TestFragmentName_Collisions(t *testing.T) {
	t.Parallel()
	b := NewBuilder()
	ta, tb, tc := reflect.TypeOf(fragA{}), reflect.TypeOf(fragB{}), reflect.TypeOf(fragC{})
	// 预先占用名称，模拟不同类型生成相同名称的情况
	b.fragmentNames["FragB"] = ta
	b.fragmentNames["FragC"] = ta
	b.fragmentNames["FragC_core"] = tb
	b.fragmentNames["FragC_core_2"] = tb

	if got := b.fragmentName(ta); got != "FragA" {
		t.Errorf("fragA = %q", got)
	}
	if got := b.fragmentName(ta); got != "FragA" { // 同一类型重复分配得到相同名称
		t.Errorf("fragA again = %q", got)
	}
	if got := b.fragmentName(tb); got != "FragB_core" {
		t.Errorf("fragB = %q", got)
	}
	if got := b.fragmentName(tc); got != "FragC_core_3" {
		t.Errorf("fragC = %q", got)
	}
	// 没有包路径的类型（如 struct{}）不追加包名，直接追加序号
	anon := reflect.TypeOf(struct{ X int }{})
	b.fragmentNames["Fragment"] = ta
	if got := b.fragmentName(anon); got != "Fragment_2" {
		t.Errorf("anonymous = %q", got)
	}
}
