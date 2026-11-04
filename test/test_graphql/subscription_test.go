package test_graphql

import (
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// 测试 Subscription
type CommentAddedSubscription struct {
	CommentAdded struct {
		ID     string    `json:"id" graphql:"id"`
		Body   string    `json:"body" graphql:"body"`
		Author UserInfo  `json:"author" graphql:"author"`
		Editor *UserInfo `json:"editor" graphql:"editor"`
	} `json:"commentAdded" graphql:"commentAdded(postId:$postId:ID!, since:$:DateTime)"`
}

const commentAddedExpected = `fragment UserInfo on UserInfo{
  id
  name
  email
}
subscription OnCommentAdded($postId:ID!,$comment_added_since:DateTime) {
  commentAdded(postId:$postId,since:$comment_added_since){
    id
    body
    author{ ...UserInfo }
    editor{ ...UserInfo }
  }
}`

func TestSubscriptionOutput(t *testing.T) {
	t.Parallel()
	type Sub struct {
		Ping string `json:"ping" graphql:"ping"`
	}
	runOutputCases(t, []outputCase{
		{name: "Subscription", v: CommentAddedSubscription{}, op: testutil.OpSubscription, opName: "OnCommentAdded", want: commentAddedExpected},
		// 匿名操作：name 为空时只输出 subscription 关键字
		{name: "Anonymous", v: Sub{}, op: testutil.OpSubscription, want: "subscription {\n  ping\n}"},
	})
}
