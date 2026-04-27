package test_error

import (
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// UnionAnonymousMemberError union 分支使用匿名结构体（未指定 type）应报错
type UnionAnonymousMemberError struct {
	Typename string `graphql:"__typename,union"`
	Item     struct {
		Name string `graphql:"name"`
	}
}

type UnionAnonymousMemberQuery struct {
	Content UnionAnonymousMemberError `graphql:"content"`
}

func TestCommonMisuse_UnionAnonymousMemberShouldFail(t *testing.T) {
	_, err := graphql.Marshal(UnionAnonymousMemberQuery{})
	if err == nil {
		t.Fatal("expected error for anonymous struct in union member, got nil")
	}
	if !strings.Contains(err.Error(), "embedded struct field") && !strings.Contains(err.Error(), "named struct type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// VariableTypeConflictError 同名变量类型冲突应报错
type VariableTypeConflictError struct {
	Products []struct {
		ID string `graphql:"id"`
	} `graphql:"products(author:$author:Int!)"`
	Contents []struct {
		ID string `graphql:"id"`
	} `graphql:"contents(author:$author:String!)"`
}

func TestCommonMisuse_VariableTypeConflictShouldFail(t *testing.T) {
	_, err := graphql.Marshal(VariableTypeConflictError{})
	if err == nil {
		t.Fatal("expected variable type conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "类型不统一") && !strings.Contains(err.Error(), "author") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// EmptyLiteralArgumentError 参数字面量为空应报错
type EmptyLiteralArgumentError struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(query::String!)"`
}

func TestCommonMisuse_EmptyLiteralArgumentShouldFail(t *testing.T) {
	_, err := graphql.Marshal(EmptyLiteralArgumentError{})
	if err == nil {
		t.Fatal("expected empty literal argument error, got nil")
	}
	if !strings.Contains(err.Error(), "参数值不能定义为空") && !strings.Contains(err.Error(), "unexpected token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// MissingVariableTypeError 变量未定义类型，组装完整 Query 时应报错
type MissingVariableTypeError struct {
	Items []struct {
		ID string `graphql:"id"`
	} `graphql:"items(id:$id)"`
}

func TestCommonMisuse_MissingVariableTypeShouldFail(t *testing.T) {
	exec, err := graphql.Marshal(MissingVariableTypeError{})
	if err != nil {
		t.Fatalf("marshal should succeed, got: %v", err)
	}
	_, err = exec.Query("MissingType")
	if err == nil {
		t.Fatal("expected missing variable type error, got nil")
	}
	if !strings.Contains(err.Error(), "缺少类型定义") {
		t.Fatalf("unexpected error: %v", err)
	}
}
