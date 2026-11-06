package core

import "testing"

func TestNewBuilder_Options(t *testing.T) {
	t.Parallel()
	b := NewBuilder()
	if !b.fragment || b.indent != Indent() {
		t.Fatalf("defaults: fragment=%v indent=%q", b.fragment, b.indent)
	}
	b = NewBuilder(nil, WithFragment(false), WithIndent("\t"), nil)
	if b.fragment || b.indent != "\t" {
		t.Fatalf("options not applied: fragment=%v indent=%q", b.fragment, b.indent)
	}
	b = NewBuilder(WithFragment(false), WithIndent("\t"), WithFragment(true), WithIndent(""))
	if !b.fragment || b.indent != "" {
		t.Fatalf("last option should win: fragment=%v indent=%q", b.fragment, b.indent)
	}
	if len(b.Fragments()) != 0 || b.Fragments() == nil {
		t.Fatalf("Fragments() should be an empty non-nil slice")
	}
}
