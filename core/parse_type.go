package core

import (
	"fmt"
	"reflect"
)

type TypeParser struct {
	source reflect.Type
	Fields []*FieldParser
	Union  bool
	Reused uint
}

// ParseType 解析结构体类型（可为多层指针 / 切片 / 数组包装）。
// 没有可查询字段（无导出字段或全部被跳过）的结构体返回 nil, nil，由调用方视为标量。
func (p *Parser) ParseType(typ reflect.Type) (*TypeParser, error) {
	if typ == nil {
		return nil, fmt.Errorf("graphql: type cannot be nil")
	}
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("graphql: must be a struct type to convert, got %s", typ)
	}
	// 检查循环引用
	if p.visiting[typ] {
		return nil, fmt.Errorf("graphql: circular reference detected for type %s", typ)
	}
	if v, ok := p.types[typ]; ok {
		if v != nil {
			v.Reused++
		}
		return v, nil
	}

	// 标记为正在访问
	p.visiting[typ] = true
	defer delete(p.visiting, typ)

	fields := make([]*FieldParser, 0, typ.NumField())
	isUnionType := false
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldParser, err := p.ParseField(field)
		if err != nil {
			return nil, err
		}
		if fieldParser == nil {
			continue
		}
		if fieldParser.FieldName == "__typename" && fieldParser.TagValue != nil && fieldParser.TagValue.HasFlag("union") {
			isUnionType = true
		}
		fields = append(fields, fieldParser)
	}
	if len(fields) == 0 {
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
