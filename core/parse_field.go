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
	TypeName   string
	Inline     bool
	FieldName  string
	TagValue   *TagValue
	Directives []*Directive // 字段指令，按书写顺序排列
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
	tagValue, rawTag, skip, err := parseFieldTagValue(field.Tag)
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
			// tagkit v1 将 "name" 解析为布尔标记而非字段调用，Name 为空；仅一个布尔标记时用其 Name 作为字段名
			if len(tagValue.Flags) == 1 && tagValue.Flags[0].IsBoolean {
				fieldName = tagValue.Flags[0].Name
			}
			// "__typename,union" 解析为两个布尔标记，需补全为 __typename 以便联合类型检测与输出
			if hasFlag(tagValue.Flags, "union") {
				fieldName = "__typename"
			}
		}
		if f := flagByName(tagValue.Flags, "alias"); f != nil && !f.IsBoolean && f.Value != nil {
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
	typeName := fieldType.Name()
	if tagValue != nil {
		if f := flagByName(tagValue.Flags, "type"); f != nil && !f.IsBoolean && f.Value != nil {
			if customType, ok := f.Value.(string); ok && strings.TrimSpace(customType) != "" {
				typeName = strings.TrimSpace(customType)
			}
		}
	}

	var fieldTagValue *TagValue = nil
	var directives []*Directive

	if tagValue != nil {
		args, keys, err := newArgs(rawTag, tagValue.Args)
		if err != nil {
			return nil, fmt.Errorf("field [%s]: %w", field.Name, err)
		}
		fieldTagValue = &TagValue{
			TagValue: tagValue,
			Args:     args,
			ArgKeys:  keys,
		}
		directives, err = parseDirectives(tagValue.Flags)
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
	}, nil
}

// parseFieldTagValue 解析字段的 graphql tag，同时返回 tag 原文；skip 为 true 表示该字段不参与生成。
//   - graphql:"-" 跳过字段；
//   - 未声明 graphql tag 时回退到 json tag，与 encoding/json 一致：仅取逗号前的名字作为字段名，
//     json:"-" 跳过字段（json:"-," 表示 key 为 "-"，不跳过），名字为空（如 json:",omitempty"）时使用 Go 字段名。
func parseFieldTagValue(tag reflect.StructTag) (tagValue *tagkit.TagValue, raw string, skip bool, err error) {
	value, ok := tag.Lookup("graphql")
	if !ok {
		jsonTag, ok := tag.Lookup("json")
		if !ok {
			return nil, "", false, nil
		}
		if jsonTag == "-" {
			return nil, "", true, nil
		}
		name, _, _ := strings.Cut(jsonTag, ",")
		return &tagkit.TagValue{Name: name}, "", false, nil
	}
	if strings.TrimSpace(value) == "-" {
		return nil, "", true, nil
	}
	tagValue, err = parseTagValue(value)
	return tagValue, value, false, err
}

// newArgs 将 tagkit 解析出的参数转换为本包的 Arg，并结合 tag 原文扫描结果确定：
//   - 参数的书写顺序（ArgKeys）；
//   - 字面量参数的 GraphQL 写法：带引号的字符串输出为 GraphQL 字符串，裸标识符原样输出（枚举值），
//     数字 / 布尔 / null 原样输出，列表逐元素按相同规则输出；
//   - 变量默认值的 GraphQL 写法：String / ID 类型变量的裸标识符默认值按字符串输出，其余同上。
func newArgs(raw string, src map[string]tagkit.ArgValue) (map[string]*Arg, []string, error) {
	scanned := scanCallArgs(raw)
	byKey := make(map[string]*scannedArg, len(scanned))
	order := make([]string, 0, len(scanned))
	for i := range scanned {
		a := &scanned[i]
		if _, dup := byKey[a.Key]; dup {
			return nil, nil, fmt.Errorf("参数 %s 重复声明", a.Key)
		}
		byKey[a.Key] = a
		order = append(order, a.Key)
	}
	args := make(map[string]*Arg, len(scanned))
	for name, argVal := range src {
		sa := byKey[name]
		if sa == nil {
			return nil, nil, fmt.Errorf("参数 %s 无法从 tag 原文中定位", name)
		}
		item := &Arg{ArgValue: argVal}
		if argVal.Type == "variable" {
			item.GraphQLType = argVal.VarType
			if sa.Default != nil {
				item.DefaultLiteral = sa.Default.render(isStringLikeType(argVal.VarType))
			}
		} else {
			value := *sa.Value
			if value.Kind == kindString {
				// 字符串字面量去除首尾空白（与旧版本行为一致），全空白视为未填写
				value.Text = strings.TrimSpace(value.Text)
				if value.Text == "" {
					return nil, nil, fmt.Errorf("参数值不能定义为空")
				}
				item.Value = value.Text
			}
			item.Literal = value.render(false)
		}
		args[name] = item
	}
	// tagkit 会丢弃列表字面量参数（如 ids:[1,2]），这里根据扫描结果补回
	for _, a := range scanned {
		if _, ok := args[a.Key]; !ok && a.Value != nil && a.Value.Kind == kindList {
			literal := a.Value.render(false)
			args[a.Key] = &Arg{ArgValue: tagkit.ArgValue{Type: "literal", Value: literal}, Literal: literal}
		}
	}
	return args, orderedArgKeys(order, args), nil
}

func hasFlag(flags []tagkit.FlagInfo, name string) bool {
	return flagByName(flags, name) != nil
}

func flagByName(flags []tagkit.FlagInfo, name string) *tagkit.FlagInfo {
	for i := range flags {
		if flags[i].Name == name {
			return &flags[i]
		}
	}
	return nil
}
