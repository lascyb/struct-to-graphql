package graphql

import (
	"fmt"
	"reflect"
	"slices"
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
type Arg struct {
	*tagkit.Arg
	Type string
}
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
		if tagValue.FieldName != "" {
			fieldName = tagValue.FieldName
		}
		if slices.Contains(tagValue.Flags, "alias") {
			if alias, ok := tagValue.FlagValues["alias"]; ok && alias != "" {
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
		for s, arg := range tagValue.Args {
			item := Arg{
				Arg: arg,
			}
			strs := strings.SplitN(strings.TrimSpace(item.Value), ":", 2)
			if strs[0] == "" {
				return nil, fmt.Errorf("参数值不能定义为空")
			}
			item.Value = strs[0]
			if len(strs) == 2 && strings.TrimSpace(strs[1]) != "" {
				item.Type = strings.TrimSpace(strs[1])
			}
			fieldTagValue.Args[s] = &item
		}
	}

	return &FieldParser{
		source:     field,
		TypeParser: typeParser,
		TypeName:   fieldType.Name(),
		FieldName:  fieldName,
		TagValue:   fieldTagValue,
		Inline:     field.Anonymous || (tagValue != nil && slices.Contains(tagValue.Flags, "inline")),
	}, nil
}

func parseFieldTagValue(tag reflect.StructTag) (*tagkit.TagValue, error) {
	value, ok := tag.Lookup("graphql")
	if !ok {
		value, ok = tag.Lookup("json")
		if ok {
			tagValue, err := tagkit.ParseValue(value)
			if err != nil {
				return nil, err
			}
			return &tagkit.TagValue{
				FieldName: tagValue.FieldName,
			}, nil
		}
	}
	return tagkit.ParseValue(value)
}
