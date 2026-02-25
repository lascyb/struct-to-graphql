package main

import (
	"fmt"
	"testing"

	"github.com/lascyb/struct-to-graphql"
)

type DiscountCodeBasic struct {
	Title   string `json:"title" graphql:"title"`
	Summary string `json:"summary" graphql:"summary"`
	Status  string `json:"status" graphql:"status"`
}
type DiscountAutomaticBasic struct {
	Title   string `json:"title" graphql:"title"`
	Summary string `json:"summary" graphql:"summary"`
	Status  string `json:"status" graphql:"status"`
}
type DiscountCodeBxgy struct {
	Title   string `json:"title" graphql:"title"`
	Summary string `json:"summary" graphql:"summary"`
	Status  string `json:"status" graphql:"status"`
}
type Discount struct {
	Typename string `json:"__typename" graphql:"__typename,union"` //联合类型必须使用 __typename,union 进行标记，否则会识别成内联字段
	DiscountCodeBasic
	DiscountAutomaticBasic
	DiscountCodeBxgy
}

type DiscountNode struct {
	Discount Discount `json:"discount" graphql:"discount"`
	Id       string   `json:"id" graphql:"id"`
}
type PageInfo struct {
	EndCursor       string `json:"endCursor" graphql:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage" graphql:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage" graphql:"hasPreviousPage"`
	StartCursor     string `json:"startCursor" graphql:"startCursor"`
}
type DiscountNodeConnection struct {
	Nodes    []DiscountNode `json:"nodes" graphql:"nodes"`
	PageInfo PageInfo       `json:"pageInfo" graphql:"pageInfo"`
}
type Query struct {
	DiscountNodeConnection DiscountNodeConnection `json:"discountNodes" graphql:"discountNodes(query:$:String=1, first: 10,after:$:[[String!!]!]!)"`
}

func Test_discountNodes(test *testing.T) {
	q, err := graphql.Marshal(Query{})
	if err != nil {
		fmt.Println(err)
		test.Fail()
	}

	// 获取完整查询
	query, err := q.Query("MyQuery")
	if err != nil {
		fmt.Println(err)
		test.Fail()
	}
	fmt.Println(query)

	//输出结果 discountNodes.graphql
	/*
		query MyQuery($query: String!) {
		    discountNodes(first: 10, query: $query){
		        nodes{
		            discount{
		                __typename
		                ... on DiscountCodeBasic {
		                    title
		                    summary
		                    status
		                }
		                ... on DiscountAutomaticBasic {
		                    title
		                    summary
		                    status
		                }
		                ... on DiscountCodeBxgy {
		                    title
		                    summary
		                    status
		                }
		            }
		            id
		        }
		        pageInfo{
		            endCursor
		            hasNextPage
		            hasPreviousPage
		            startCursor
		        }
		    }
		}
	*/
}
