// Package other 提供与 test_graphql 中同名的类型，用于测试 Fragment 重名处理。
package other

// UserInfo 与 test_graphql.UserInfo 同名
type UserInfo struct {
	ID string `json:"id" graphql:"id"`
}
