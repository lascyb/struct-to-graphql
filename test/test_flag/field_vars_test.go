package test_flag

import (
	"fmt"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

type Order struct {
	ID string `graphql:"id"`
}
type Query struct {
	Orders []Order `graphql:"orders(query:'financial_status:paid OR financial_status:partially_refunded',name:lascyb)"`
}

type QueryLiteralType struct {
	Orders []Order `graphql:"orders(sort:$desc:String!)"`
}

func TestFieldVars(t *testing.T) {
	marshal, err := graphql.Marshal(Query{})
	if err != nil {
		fmt.Println(err)
		return
	}
	query, qErr := marshal.Query("MyQuery")
	if qErr != nil {
		t.Fatalf("query build failed: %v", qErr)
	}
	fmt.Println(query)
}

func TestFieldVars_LiteralWithColonShouldKeepRawValue(t *testing.T) {
	marshal, err := graphql.Marshal(QueryLiteralType{})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	query, qErr := marshal.Query("MyQuery")
	if qErr != nil {
		t.Fatalf("query build failed: %v", qErr)
	}
	fmt.Println(query)

}
