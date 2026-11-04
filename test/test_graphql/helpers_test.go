package test_graphql

import (
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// outputCase 一个「结构体 → 完整操作字符串」的表驱动用例
type outputCase struct {
	name   string
	v      any
	op     testutil.Op // 为空时按 query 处理
	opName string
	want   string // 内联期望值；为空时与 testdata/<name>.golden 比对
}

// runOutputCases 并行执行用例并完整比对输出
func runOutputCases(t *testing.T, cases []outputCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, got := testutil.Build(t, c.v, c.op, c.opName)
			if c.want == "" {
				testutil.AssertGolden(t, c.name, got)
				return
			}
			testutil.AssertEqual(t, got, c.want)
		})
	}
}
