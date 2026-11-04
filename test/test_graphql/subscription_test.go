package test_graphql

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
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

const commentAddedExpected = `fragment Test_graphqlUserInfo on UserInfo{
  id
  name
  email
}
subscription OnCommentAdded($postId:ID!,$comment_added_since:DateTime) {
  commentAdded(postId:$postId,since:$comment_added_since){
    id
    body
    author{ ...Test_graphqlUserInfo }
    editor{ ...Test_graphqlUserInfo }
  }
}`

func TestSubscription(t *testing.T) {
	exec, err := graphql.Marshal(CommentAddedSubscription{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	subscription, err := exec.Subscription("OnCommentAdded")
	if err != nil {
		t.Fatalf("Subscription failed: %v", err)
	}
	t.Logf("Generated Subscription:\n%s", subscription)
	if subscription != commentAddedExpected {
		t.Fatalf("unexpected output.\n--- got ---\n%s\n--- want ---\n%s", subscription, commentAddedExpected)
	}
}

// 匿名操作：name 为空时只输出 subscription 关键字
func TestSubscription_Anonymous(t *testing.T) {
	type Sub struct {
		Ping string `graphql:"ping"`
	}
	exec, err := graphql.Marshal(Sub{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	got, err := exec.Subscription("")
	if err != nil {
		t.Fatalf("Subscription failed: %v", err)
	}
	want := "subscription {\n  ping\n}"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// nil 接收者应返回错误而不是 panic
func TestSubscription_NilGraphql(t *testing.T) {
	var g *graphql.Graphql
	if _, err := g.Subscription("X"); err == nil {
		t.Fatal("expected error for nil Graphql, got nil")
	}
}
