package core

import (
	"reflect"
)

type Parser struct {
	types    map[reflect.Type]*TypeParser
	visiting map[reflect.Type]bool // 循环引用检测
	root     *TypeParser
}

func NewParser() *Parser {
	return &Parser{
		types:    make(map[reflect.Type]*TypeParser),
		visiting: make(map[reflect.Type]bool),
	}
}
