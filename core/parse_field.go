package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/lascyb/tagkit"
)

type FieldParser struct {
	source     reflect.StructField
	TypeParser *TypeParser
	TypeName   string // union 分支 "... on" 使用的类型名：tag 中的 type=X > 结构体 GraphQLType() > Go 类型名（去掉泛型参数）
	Inline     bool
	FieldName  string
	TagValue   *TagValue
	Directives []*Directive // 字段指令，按书写顺序排列

	typeFlag     string // tag 中的 type=X，未设置时为空
	fragmentFlag string // tag 中的 fragment=X，未设置时为空
}

// Arg 包装 tagkit.ArgValue，GraphQLType 为变量的 GraphQL 类型或字面量的自定义类型
type Arg struct {
	tagkit.ArgValue
	GraphQLType    string
	Literal        string // 字面量参数的 GraphQL 写法（如 "abc"、CREATED_AT、10、[1,2]）
	DefaultLiteral string // 变量默认值的 GraphQL 写法，未声明默认值时为空
}

// TagValue 包装 tagkit.TagValue，Args 的 value 使用本包的 Arg 以携带 Type
type TagValue struct {
	*tagkit.TagValue
	Args    map[string]*Arg
	ArgKeys []string // 参数名列表，按 tag 原文中的书写顺序排列，用于稳定输出
}

// ParseField 解析单个结构体字段；字段被 tag 标记为跳过（graphql:"-" 或 json:"-"）时返回 nil, nil
func (p *Parser) ParseField(field reflect.StructField) (*FieldParser, error) {
	tagValue, skip, err := parseFieldTagValue(field.Tag)
	if err != nil {
		return nil, fmt.Errorf("field [%s]: %w", field.Name, err)
	}
	if skip {
		return nil, nil
	}
	fieldName := field.Name
	if tagValue != nil {
		if tagValue.Name != "" {
			fieldName = tagValue.Name
		} else {
			// tag 以逗号开头（如 ",name"）时 tagkit 不产生字段调用，Name 为空；
			// 此时若只有一个裸标记，沿用其名字作为字段名（兼容旧写法）
			if len(tagValue.Flags) == 1 && tagValue.Flags[0].IsBoolean {
				fieldName = tagValue.Flags[0].Name
			}
			// "__typename,union" 解析为两个布尔标记，需补全为 __typename 以便联合类型检测与输出
			if tagValue.HasFlag("union") {
				fieldName = "__typename"
			}
		}
		if f, ok := tagValue.Flag("alias"); ok && f.HasValue() {
			if alias, ok := f.Value.(string); ok && alias != "" {
				fieldName = alias + ":" + fieldName
			}
		}
	}
	fieldType := field.Type
	// 支持多层指针 / 切片 / 数组（例如 []*MetaInfo、[][]*MetaInfo、[3]Item），一路解开直到命中基础结构体类型；
	// map、interface 等其他类型以及没有导出字段的结构体（如 time.Time）均视为标量字段
	for fieldType.Kind() == reflect.Ptr || fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		fieldType = fieldType.Elem()
	}

	var typeParser *TypeParser = nil
	if fieldType.Kind() == reflect.Struct {
		typeParser, err = p.ParseType(fieldType)
		if err != nil {
			return nil, err
		}
	}
	typeFlagValue, err := nameFlag(tagValue, typeFlag)
	if err != nil {
		return nil, fmt.Errorf("field [%s]: %w", field.Name, err)
	}
	fragmentFlagValue, err := nameFlag(tagValue, fragmentFlag)
	if err != nil {
		return nil, fmt.Errorf("field [%s]: %w", field.Name, err)
	}
	typeName := goTypeName(fieldType)
	switch {
	case typeFlagValue != "":
		typeName = typeFlagValue
	case typeParser != nil && typeParser.graphQLType != "":
		typeName = typeParser.graphQLType
	}

	var fieldTagValue *TagValue = nil
	var directives []*Directive

	if tagValue != nil {
		args, keys, err := newArgs(tagValue)
		if err != nil {
			return nil, fmt.Errorf("field [%s]: %w", field.Name, err)
		}
		fieldTagValue = &TagValue{
			TagValue: tagValue,
			Args:     args,
			ArgKeys:  keys,
		}
		directives, err = parseDirectives(tagValue)
		if err != nil {
			return nil, fmt.Errorf("field [%s]: %w", field.Name, err)
		}
	}

	return &FieldParser{
		source:     field,
		TypeParser: typeParser,
		TypeName:   typeName,
		FieldName:  fieldName,
		TagValue:   fieldTagValue,
		Inline:     field.Anonymous,
		Directives: directives,

		typeFlag:     typeFlagValue,
		fragmentFlag: fragmentFlagValue,
	}, nil
}

// parseFieldTagValue 解析字段的 graphql tag；skip 为 true 表示该字段不参与生成。
//   - graphql:"-" 跳过字段；
//   - 未声明 graphql tag 时回退到 json tag，与 encoding/json 一致：仅取逗号前的名字作为字段名，
//     json:"-" 跳过字段（json:"-," 表示 key 为 "-"，不跳过），名字为空（如 json:",omitempty"）时使用 Go 字段名。
func parseFieldTagValue(tag reflect.StructTag) (tagValue *tagkit.TagValue, skip bool, err error) {
	value, ok := tag.Lookup("graphql")
	if !ok {
		jsonTag, ok := tag.Lookup("json")
		if !ok {
			return nil, false, nil
		}
		if jsonTag == "-" {
			return nil, true, nil
		}
		name, _, _ := strings.Cut(jsonTag, ",")
		return &tagkit.TagValue{Name: name}, false, nil
	}
	if strings.TrimSpace(value) == "-" {
		return nil, true, nil
	}
	tagValue, err = tagkit.ParseTagValue(value)
	return tagValue, false, err
}

// newArgs 将 tagkit 解析出的参数转换为本包的 Arg，按书写顺序（tv.ArgKeys）返回参数名列表，并确定：
//   - 字面量参数的 GraphQL 写法：带引号的字符串输出为 GraphQL 字符串，裸标识符原样输出（枚举值），
//     数字 / 布尔 / null 原样输出，列表逐元素按相同规则输出；
//   - 变量默认值的 GraphQL 写法：String / ID 类型变量的裸标识符默认值按字符串输出，其余同上。
//
// 语法错误（重复参数、非法字面量、默认值与类型不符等）已由 tagkit 报告。
func newArgs(tv *tagkit.TagValue) (map[string]*Arg, []string, error) {
	ordered := tv.OrderedArgs()
	args := make(map[string]*Arg, len(ordered))
	keys := make([]string, 0, len(ordered))
	for _, a := range ordered {
		item := &Arg{ArgValue: a.ArgValue}
		if a.Type == tagkit.ArgTypeVariable {
			item.GraphQLType = a.VarType
			if a.Default != nil {
				item.DefaultLiteral = renderLiteral(a.Default, isStringLikeType(a.VarType))
			}
		} else {
			lit := a.Literal
			if lit.Kind == tagkit.KindString {
				// 字符串字面量去除首尾空白（与旧版本行为一致），全空白视为未填写
				text := strings.TrimSpace(lit.Value.(string))
				if text == "" {
					return nil, nil, fmt.Errorf("参数值不能定义为空")
				}
				item.Value = text
				item.Literal = quoteGraphQLString(text)
			} else {
				item.Literal = renderLiteral(lit, false)
			}
		}
		args[a.Name] = item
		keys = append(keys, a.Name)
	}
	return args, keys, nil
}
