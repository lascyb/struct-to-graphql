package core

// Option 定制单次构建行为的可选参数，传给 NewBuilder（或根包的 Marshal）
type Option func(*Builder)

// WithFragment 控制是否把被多次引用的命名结构体（Reused > 1）抽取为顶层 Fragment，默认 true。
// 传 false 时这些结构体在每个引用处原地展开为 { ... }；union 分支仍输出 "... on Type { ... }"。
func WithFragment(enable bool) Option {
	return func(b *Builder) {
		b.fragment = enable
	}
}

// WithIndent 设置本次构建的单级缩进，覆盖全局 SetIndent 的值，且不修改全局状态
func WithIndent(val string) Option {
	return func(b *Builder) {
		b.indent = val
	}
}
