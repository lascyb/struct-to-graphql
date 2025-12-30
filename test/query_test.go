package main

import (
	"fmt"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

func TestQuery(t *testing.T) {
	type Query struct {
		Field1 string `graphql:"field1"`
		List   struct {
			Nodes []struct {
				Name string `graphql:"name"`
			} `graphql:"nodes"`
		} `graphql:"list(first:10,query:$:String!,id:$id:Int!)"`
	}

	q, err := graphql.Marshal(Query{})
	if err != nil {
		panic(err)
	}

	query, err := q.Query("GetData")
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 完整查询 ===")
	fmt.Println(query)
}
