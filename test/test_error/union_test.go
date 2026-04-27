package test_error

import (
	"fmt"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

type Query struct {
	Typename string `graphql:"__typename,union"`
	Name     struct {
		Name string `graphql:"name"`
	}
	Height struct {
		Name string `graphql:"name"`
	}
}

func TestName(t *testing.T) {
	marshal, err := graphql.Marshal(Query{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(marshal.Query("AA"))
}
