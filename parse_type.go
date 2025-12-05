package graphql

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
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
		return nil, fmt.Errorf("graphql: 检测到类型 %s 的循环引用", typeName)
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
		return nil, errors.New("必须是结构体类型才需要转换")
	}
	if v, ok := p.types[typ]; ok {
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
		if fieldParser.FieldName == "__typename" && slices.Contains(fieldParser.TagValue.Flags, "union") {
			isUnionType = true
		}
		fields = append(fields, fieldParser)
	}
	if exportedCount == 0 {
		p.types[typ] = nil
		return nil, nil
	}
	p.types[typ] = &TypeParser{
		source: typ,
		Fields: fields,
		Union:  isUnionType,
		Reused: 1,
	}
	return p.types[typ], nil
}
