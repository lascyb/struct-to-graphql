package test_error

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== Directives 常见错误用法 ==========

// directive 作为布尔标记（缺少值）
type DirectiveBooleanFlag struct {
	ID string `graphql:"id,directive"`
}

// directive 值不是字符串
type DirectiveNumberValue struct {
	ID string `graphql:"id,directive=1"`
}

// directive 值为数组（多个指令应分别声明）
type DirectiveArrayValue struct {
	ID string `graphql:"id,directive=[include,skip]"`
}

// directive 值为空
type DirectiveEmptyValue struct {
	ID string `graphql:"id,directive=''"`
}

// 一个 directive 中写多个指令
type DirectiveMultipleInOne struct {
	ID string `graphql:"id,directive='include(if:true),skip(if:false)'"`
}

// 指令后带有额外的标记位
type DirectiveExtraFlag struct {
	ID string `graphql:"id,directive='include(if:true),skip'"`
}

// 指令参数重复 / 为空
type DirectiveDuplicateArg struct {
	ID string `graphql:"id,directive='cached(ttl:1,ttl:2)'"`
}
type DirectiveEmptyArg struct {
	ID string `graphql:"id,directive='cached(scope:\"\")'"`
}

// 缺少指令名
type DirectiveMissingName struct {
	ID string `graphql:"id,directive='(if:true)'"`
}

// @include 缺少 if 参数
type DirectiveIncludeWithoutIf struct {
	ID string `graphql:"id,directive='include(when:true)'"`
}

// @skip 带有多余参数
type DirectiveSkipExtraArg struct {
	ID string `graphql:"id,directive='skip(if:true,reason:x)'"`
}

// @include 的 if 不是布尔值
type DirectiveIncludeNonBool struct {
	ID string `graphql:"id,directive='include(if:yes)'"`
}

// 同一字段重复 @include
type DirectiveDuplicateInclude struct {
	ID string `graphql:"id,directive='include(if:$a)',directive='include(if:$b)'"`
}

// 匿名嵌入（字段平铺）上不能声明指令
type DirectiveFlattenInner struct {
	Name string `graphql:"name"`
}
type DirectiveOnEmbedded struct {
	ID                    string `graphql:"id"`
	DirectiveFlattenInner `graphql:",directive='include(if:true)'"`
}

// 指令变量与字段变量同名但类型不一致
type DirectiveVariableTypeConflict struct {
	Posts []struct {
		ID string `graphql:"id"`
	} `graphql:"posts(draft:$preview:String),directive='include(if:$preview:Boolean!)'"`
}

func TestDirectiveMisuse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		v     any
		wants []string // 错误信息必须包含的全部片段
	}{
		{"BooleanFlag", DirectiveBooleanFlag{}, []string{"字符串形式"}},
		{"NumberValue", DirectiveNumberValue{}, []string{"字符串形式"}},
		{"ArrayValue", DirectiveArrayValue{}, []string{"字符串形式"}},
		{"EmptyValue", DirectiveEmptyValue{}, []string{"不能为空"}},
		{"MultipleInOne", DirectiveMultipleInOne{}, []string{"field [ID]", "field must be unique"}},
		{"ExtraFlag", DirectiveExtraFlag{}, []string{"field [ID]", "只能包含一个指令调用"}},
		{"DuplicateArg", DirectiveDuplicateArg{}, []string{"directive [cached(ttl:1,ttl:2)]", `duplicate argument "ttl" at position 13`}},
		{"EmptyArg", DirectiveEmptyArg{}, []string{"参数值不能定义为空"}},
		{"MissingName", DirectiveMissingName{}, []string{"缺少合法的指令名"}},
		{"IncludeWithoutIf", DirectiveIncludeWithoutIf{}, []string{"必须且只能包含参数 if"}},
		{"SkipExtraArg", DirectiveSkipExtraArg{}, []string{"必须且只能包含参数 if"}},
		{"IncludeNonBool", DirectiveIncludeNonBool{}, []string{"必须是布尔值或变量"}},
		{"DuplicateInclude", DirectiveDuplicateInclude{}, []string{"不能在同一字段上重复声明"}},
		{"OnEmbedded", DirectiveOnEmbedded{}, []string{"embedded (flattened) field"}},
		{"VariableTypeConflict", DirectiveVariableTypeConflict{}, []string{"类型不统一"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertMarshalError(t, c.v, c.wants...)
		})
	}
}

// 自定义指令的变量未声明类型：Marshal 成功，组装完整操作时报错
type DirectiveCustomMissingType struct {
	ID string `graphql:"id,directive='cached(ttl:$ttl)'"`
}

func TestDirectiveMisuse_CustomVariableMissingType(t *testing.T) {
	t.Parallel()
	exec, err := graphql.Marshal(DirectiveCustomMissingType{})
	if err != nil {
		t.Fatalf("marshal should succeed, got: %v", err)
	}
	_, err = exec.Query("MissingType")
	testutil.AssertError(t, err, "变量 $ttl 缺少类型定义")
}
