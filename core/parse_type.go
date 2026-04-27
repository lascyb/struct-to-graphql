package core

import (
	"errors"
	"fmt"
	"reflect"
)

type TypeParser struct {
	source reflect.Type
	Fields []*FieldParser
	Union  bool
	Reused uint
}

func (p *Parser) ParseType(typ reflect.Type) (*TypeParser, error) {
	// 检查循环引用
	if p.visiting[typ] {
		typeName := typ.Name()
		if typeName == "" {
			typeName = typ.String()
		}
		return nil, fmt.Errorf("graphql: circular reference detected for type %s", typeName)
	}

	// 标记为正在访问
	p.visiting[typ] = true
	defer func() {
		delete(p.visiting, typ)
	}()

	if typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("must be a struct type to convert")
	}
	if v, ok := p.types[typ]; ok && v != nil {
		v.Reused++
		return v, nil
	}
	fields := make([]*FieldParser, 0)
	isUnionType := false
	exportedCount := 0
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		exportedCount++
		fieldParser, err := p.ParseField(field)
		if err != nil {
			return nil, err
		}
		if fieldParser.FieldName == "__typename" && fieldParser.TagValue != nil && hasFlag(fieldParser.TagValue.Flags, "union") {
			isUnionType = true
		}
		fields = append(fields, fieldParser)
	}
	if exportedCount == 0 {
		p.types[typ] = nil
		return nil, nil
	}
	if isUnionType {
		for _, field := range fields {
			if field.FieldName == "__typename" {
				continue
			}
			if !field.source.Anonymous {
				return nil, fmt.Errorf("field [%s] in union type [%s] should be an embedded struct field", field.FieldName, typ.String())
			}
			// union 分支必须是命名结构体，避免匿名结构体导致响应无法稳定反序列化
			if field.TypeParser == nil || field.TypeParser.source.Name() == "" {
				return nil, fmt.Errorf("field [%s] in union type [%s] should be a named struct type", field.FieldName, typ.String())
			}
		}
	}
	p.types[typ] = &TypeParser{
		source: typ,
		Fields: fields,
		Union:  isUnionType,
		Reused: 1,
	}
	return p.types[typ], nil
}
