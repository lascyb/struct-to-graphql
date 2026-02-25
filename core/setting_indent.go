package core

import "strings"

var indentMap = map[uint]string{
	1: "  ",
}

func SetIndent(val string) {
	indentMap = map[uint]string{
		1: val,
	}
}
func indentWithLevel(level uint) string {
	if v, ok := indentMap[level]; ok {
		return v
	}
	indentMap[level] = strings.Repeat(indentMap[1], int(level))
	return indentMap[level]
}
