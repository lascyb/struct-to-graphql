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
}

// Arg 包装 tagkit.ArgValue，GraphQLType 为变量的 GraphQL 类型或字面量的自定义类型
type Arg struct {
	tagkit.ArgValue
	GraphQLType string
}

// TagValue 包装 tagkit.TagValue，Args 的 value 使用本包的 Arg 以携带 Type
type TagValue struct {
	*tagkit.TagValue
	Args map[string]*Arg
}

func (p *Parser) ParseField(field reflect.StructField) (*FieldParser, error) {
	tagValue, err := parseFieldTagValue(field.Tag)
	if err != nil {
		return nil, err
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
	if field.Type.Kind() == reflect.Ptr || field.Type.Kind() == reflect.Slice {
		fieldType = field.Type.Elem()
	}

	var typeParser *TypeParser = nil
	if fieldType.Kind() == reflect.Struct {
		typeParser, err = p.ParseType(fieldType)
		if err != nil {
			return nil, err
		}
	}

	var fieldTagValue *TagValue = nil

	if tagValue != nil {
		fieldTagValue = &TagValue{
			TagValue: tagValue,
			Args:     make(map[string]*Arg),
		}
		for name, argVal := range tagValue.Args {
			item := &Arg{ArgValue: argVal}
			if argVal.Type == "variable" {
				item.GraphQLType = argVal.VarType
			}
			if s, ok := argVal.Value.(string); ok && argVal.Type == "literal" {
				strs := strings.SplitN(strings.TrimSpace(s), ":", 2)
				if len(strs) > 0 && strings.TrimSpace(strs[0]) == "" {
					return nil, fmt.Errorf("参数值不能定义为空")
				}
				item.Value = strings.TrimSpace(strs[0])
				if len(strs) == 2 && strings.TrimSpace(strs[1]) != "" {
					item.GraphQLType = strings.TrimSpace(strs[1])
				}
			}
			fieldTagValue.Args[name] = item
		}
	}

	return &FieldParser{
		source:     field,
		TypeParser: typeParser,
		TypeName:   fieldType.Name(),
		FieldName:  fieldName,
		TagValue:   fieldTagValue,
		Inline:     field.Anonymous || (tagValue != nil && hasFlag(tagValue.Flags, "inline")),
	}, nil
}

func parseFieldTagValue(tag reflect.StructTag) (*tagkit.TagValue, error) {
	value, ok := tag.Lookup("graphql")
	if !ok {
		value, ok = tag.Lookup("json")
		if ok {
			tagValue, err := tagkit.ParseTagValue(value)
			if err != nil {
				return nil, err
			}
			return &tagkit.TagValue{
				Name: tagValue.Name,
			}, nil
		}
	}
	return tagkit.ParseTagValue(value)
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
