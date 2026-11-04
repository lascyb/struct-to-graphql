package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestCamelToSnake(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"":                "",
		"first":           "first",
		"commentsLast":    "comments_last",
		"Items_after":     "items_after",
		"pinnedID":        "pinned_i_d",
		"posts_nodes_ABC": "posts_nodes__a_b_c",
	} {
		if got := CamelToSnake(in); got != want {
			t.Errorf("CamelToSnake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuild_NilParser(t *testing.T) {
	t.Parallel()
	if _, err := NewBuilder().Build(nil); err == nil || !strings.Contains(err.Error(), "no queryable fields") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestParseType_NilType(t *testing.T) {
	t.Parallel()
	if _, err := NewParser().ParseType(nil); err == nil || !strings.Contains(err.Error(), "type cannot be nil") {
		t.Fatalf("expected error, got %v", err)
	}
}

type builderFieldTag struct {
	A string `graphql:"a(x:'\\"`
}

func TestParseField_InvalidTagDoesNotPanic(t *testing.T) {
	t.Parallel()
	_, err := NewParser().ParseType(reflect.TypeOf(builderFieldTag{}))
	if err == nil || !strings.Contains(err.Error(), "field [A]: invalid tag") {
		t.Fatalf("expected invalid tag error, got %v", err)
	}
}

// 直接构造 FieldParser：普通字段上的参数构建失败、指令构建失败时应返回错误
func TestBuilder_ArgumentErrors(t *testing.T) {
	t.Parallel()
	type q struct {
		A string `graphql:"a(x:$v:Int)"`
		B string `graphql:"b,directive='include(if:$v)'"` // $v 推断为 Boolean!，与 Int 冲突
	}
	tp, err := NewParser().ParseType(reflect.TypeOf(q{}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewBuilder().Build(tp)
	if err == nil || !strings.Contains(err.Error(), "directive @include on field [b]") || !strings.Contains(err.Error(), "类型不统一") {
		t.Fatalf("unexpected error: %v", err)
	}
}
