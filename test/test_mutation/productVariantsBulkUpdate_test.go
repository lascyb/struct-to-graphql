package main

import (
	"fmt"
	"testing"

	"github.com/lascyb/struct-to-graphql"
)

type Product struct {
	Id string `json:"id" graphql:"id"`
}
type ProductVariant struct {
	ID    string `json:"id" graphql:"id"`
	Sku   string `json:"sku" graphql:"sku"`
	Price string `json:"price" graphql:"price"`
}
type ProductVariantsBulkUpdate struct {
	Product         Product          `json:"product" graphql:"product"`
	ProductVariants []ProductVariant `json:"productVariants" graphql:"productVariants"`
}

// ProductVariantsBulkUpdatePayload 生成Mutation的同时，如果定义了正确的 json tag，就可以用来接受请求响应
type Mutation struct {
	// 注意参数定义顺序，参数名:$变量名:变量类型
	ProductVariantsBulkUpdate ProductVariantsBulkUpdate `json:"productVariantsBulkUpdate" graphql:"productVariantsBulkUpdate(productId:$productId:ID!,variants:$variants:[ProductVariantsBulkInput!]!)"`
}

func Test_productVariantsBulkUpdate(test *testing.T) {
	q, _ := graphql.Marshal(Mutation{})

	// 获取完整查询
	query, _ := q.Mutation("MyMutation")
	fmt.Println(query)
	//输出结果 productVariantsBulkUpdate.graphql
	/*
		mutation UpdateProductVariantsOptionValuesInBulk($productId: ID!, $variants: [ProductVariantsBulkInput!]!) {
		    productVariantsBulkUpdate(productId: $productId, variants: $variants){
		        product{
		            id
		        }
		        productVariants{
		            id
		            sku
		            price
		        }
		    }
		}
	*/
}
