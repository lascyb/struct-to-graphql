package main

import (
	"fmt"

	graphql "github.com/lascyb/struct-to-graphql"
)

type Foo struct {
	Bar string `graphql:"bar"`
}
type ProductVariant struct {
	ID    string `json:"id" graphql:"id"`
	Price string `json:"price" graphql:"price"`
}
type Mutation struct {
	ProductVariantsBulkUpdate struct {
		Product struct {
			ID string `json:"id" graphql:"id"`
		} `graphql:"product"`
		ProductVariants []ProductVariant `json:"productVariants" graphql:"productVariants"`
	} `graphql:"productVariantsBulkUpdate(productId:$productId:ID!,variants: $variants:[ProductVariantsBulkInput!]!)"`
}

func main() {
	q, _ := graphql.Marshal(Mutation{})

	// 获取完整查询
	query, _ := q.Mutation("productVariantsBulkUpdate")
	fmt.Println(query)
}
