package test_query

import (
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// ========== Query 示例：列表 / 分页 / 变量默认值 ==========
// 参考 Shopify Admin API 的 discountNodes 查询

type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage" graphql:"hasNextPage"`
	EndCursor   string `json:"endCursor" graphql:"endCursor"`
}

type DiscountCodeBasic struct {
	Title  string `json:"title" graphql:"title"`
	Status string `json:"status" graphql:"status"`
	Codes  struct {
		Nodes []struct {
			Code string `json:"code" graphql:"code"`
		} `json:"nodes" graphql:"nodes"`
	} `json:"codes" graphql:"codes(first:$codesFirst:Int=10)"`
}

type DiscountAutomaticBasic struct {
	StartsAt        string `json:"startsAt" graphql:"startsAt"`
	AsyncUsageCount int    `json:"asyncUsageCount" graphql:"asyncUsageCount"`
}

// Discount 为 union：通过 __typename 区分具体类型
type Discount struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	DiscountCodeBasic
	DiscountAutomaticBasic
}

type DiscountNodesQuery struct {
	DiscountNodes struct {
		Nodes []struct {
			ID       string   `json:"id" graphql:"id"`
			Discount Discount `json:"discount" graphql:"discount"`
		} `json:"nodes" graphql:"nodes"`
		PageInfo PageInfo `json:"pageInfo" graphql:"pageInfo"`
	} `json:"discountNodes" graphql:"discountNodes(first:$first:Int=50, after:$after:String, query:$query:String, sortKey:CREATED_AT, reverse:true)"`
}

const discountNodesExpected = `query DiscountNodes($first:Int=50,$after:String,$query:String,$codesFirst:Int=10) {
  discountNodes(first:$first,after:$after,query:$query,sortKey:CREATED_AT,reverse:true){
    nodes{
      id
      discount{
        __typename
        ... on DiscountCodeBasic {
          title
          status
          codes(first:$codesFirst){
            nodes{
              code
            }
          }
        }
        ... on DiscountAutomaticBasic {
          startsAt
          asyncUsageCount
        }
      }
    }
    pageInfo{
      hasNextPage
      endCursor
    }
  }
}`

func TestDiscountNodesQuery(t *testing.T) {
	exec, err := graphql.Marshal(DiscountNodesQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	query, err := exec.Query("DiscountNodes")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	t.Logf("Generated Query:\n%s", query)
	if query != discountNodesExpected {
		t.Fatalf("unexpected output.\n--- got ---\n%s\n--- want ---\n%s", query, discountNodesExpected)
	}
}
