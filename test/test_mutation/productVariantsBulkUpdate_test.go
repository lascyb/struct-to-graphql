package test_mutation

import (
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// ========== Mutation 示例 ==========
// 参考 Shopify Admin API 的 productVariantsBulkUpdate 变更

type ProductVariant struct {
	ID    string `json:"id" graphql:"id"`
	Title string `json:"title" graphql:"title"`
	Price string `json:"price" graphql:"price"`
}

type ProductVariantsBulkUpdateMutation struct {
	ProductVariantsBulkUpdate struct {
		Product struct {
			ID string `json:"id" graphql:"id"`
		} `json:"product" graphql:"product"`
		ProductVariants []ProductVariant `json:"productVariants" graphql:"productVariants"`
		UserErrors      []struct {
			Field   []string `json:"field" graphql:"field"`
			Message string   `json:"message" graphql:"message"`
		} `json:"userErrors" graphql:"userErrors"`
	} `json:"productVariantsBulkUpdate" graphql:"productVariantsBulkUpdate(productId:$productId:ID!, variants:$variants:[ProductVariantsBulkInput!]!, allowPartialUpdates:$allowPartialUpdates:Boolean=false)"`
}

const productVariantsBulkUpdateExpected = `mutation ProductVariantsBulkUpdate($productId:ID!,$variants:[ProductVariantsBulkInput!]!,$allowPartialUpdates:Boolean=false) {
  productVariantsBulkUpdate(productId:$productId,variants:$variants,allowPartialUpdates:$allowPartialUpdates){
    product{
      id
    }
    productVariants{
      id
      title
      price
    }
    userErrors{
      field
      message
    }
  }
}`

func TestProductVariantsBulkUpdateMutation(t *testing.T) {
	t.Parallel()
	_, mutation := testutil.Build(t, ProductVariantsBulkUpdateMutation{}, testutil.OpMutation, "ProductVariantsBulkUpdate")
	testutil.AssertEqual(t, mutation, productVariantsBulkUpdateExpected)
	testutil.AssertResponseKeys(t, ProductVariantsBulkUpdateMutation{})
}
