package main

import (
	"fmt"

	graphql "github.com/lascyb/struct-to-graphql"
)

type Foo struct {
	Bar string `graphql:"bar"`
}

type Query struct {
	Field1 string `graphql:"field1"`
	List   struct {
		Foo1  Foo `graphql:"foo1"`
		Foo2  Foo `graphql:"foo2"`
		Nodes []struct {
			Name string `graphql:"name"`
		} `graphql:"nodes"`
	} `graphql:"list(first:10,query:$:String!,id:$id:Int!)"`
}

func main() {
	q, _ := graphql.Marshal(Query{})

	// 获取完整查询
	query, _ := q.Query("GetData")
	fmt.Println(query)
}
