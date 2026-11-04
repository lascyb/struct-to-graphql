package test_mutation

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
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
	exec, err := graphql.Marshal(ProductVariantsBulkUpdateMutation{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	mutation, err := exec.Mutation("ProductVariantsBulkUpdate")
	if err != nil {
		t.Fatalf("Mutation failed: %v", err)
	}
	t.Logf("Generated Mutation:\n%s", mutation)
	if mutation != productVariantsBulkUpdateExpected {
		t.Fatalf("unexpected output.\n--- got ---\n%s\n--- want ---\n%s", mutation, productVariantsBulkUpdateExpected)
	}
}
