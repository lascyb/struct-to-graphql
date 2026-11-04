package core

import (
	"strings"
	"sync/atomic"
)

// DefaultIndent 默认的单级缩进
const DefaultIndent = "  "

var indent atomic.Pointer[string]

func init() {
	SetIndent(DefaultIndent)
}

// SetIndent 设置全局单级缩进，并发安全；已经开始的 Marshal 调用不受影响
func SetIndent(val string) {
	indent.Store(&val)
}

// Indent 返回当前的全局单级缩进
func Indent() string {
	return *indent.Load()
}

// indentWithLevel 返回 level 级缩进
func (g *Builder) indentWithLevel(level uint) string {
	return strings.Repeat(g.indent, int(level))
}
