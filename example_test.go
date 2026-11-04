package graphql_test

import (
	"fmt"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

type User struct {
	ID   string `json:"id" graphql:"id"`
	Name string `json:"name" graphql:"name"`
}

type Query struct {
	User User `json:"user" graphql:"user(id:$id:ID!)"`
	List struct {
		Nodes []User `json:"nodes" graphql:"nodes"`
	} `json:"list" graphql:"list(first:10, query:$:String!, sortKey:CREATED_AT, status:'active')"`
}

func ExampleMarshal() {
	q, err := graphql.Marshal(Query{})
	if err != nil {
		panic(err)
	}
	query, err := q.Query("GetData")
	if err != nil {
		panic(err)
	}
	fmt.Println(query)
	for _, v := range q.Variables {
		fmt.Println(v.Name, v.Type)
	}
	// Output:
	// fragment User on User{
	//   id
	//   name
	// }
	// query GetData($id:ID!,$list_query:String!) {
	//   user(id:$id){ ...User }
	//   list(first:10,query:$list_query,sortKey:CREATED_AT,status:"active"){
	//     nodes{ ...User }
	//   }
	// }
	// $id ID!
	// $list_query String!
}

func ExampleGraphql_Mutation() {
	type mutation struct {
		ProductUpdate struct {
			Product struct {
				ID    string `json:"id" graphql:"id"`
				Title string `json:"title" graphql:"title"`
			} `json:"product" graphql:"product"`
		} `json:"productUpdate" graphql:"productUpdate(input:$input:ProductInput!)"`
	}
	q, _ := graphql.Marshal(mutation{})
	out, _ := q.Mutation("UpdateProduct")
	fmt.Println(out)
	// Output:
	// mutation UpdateProduct($input:ProductInput!) {
	//   productUpdate(input:$input){
	//     product{
	//       id
	//       title
	//     }
	//   }
	// }
}

func ExampleGraphql_Subscription() {
	type subscription struct {
		CommentAdded struct {
			ID   string `json:"id" graphql:"id"`
			Body string `json:"body" graphql:"body,directive='include(if:$withBody)'"`
		} `json:"commentAdded" graphql:"commentAdded(postId:$postId:ID!)"`
	}
	q, _ := graphql.Marshal(subscription{})
	out, _ := q.Subscription("OnCommentAdded")
	fmt.Println(out)
	// Output:
	// subscription OnCommentAdded($postId:ID!,$withBody:Boolean!) {
	//   commentAdded(postId:$postId){
	//     id
	//     body @include(if:$withBody)
	//   }
	// }
}

func ExampleSetIndent() {
	graphql.SetIndent("\t")
	defer graphql.SetIndent("  ")
	type query struct {
		Viewer struct {
			Login string `json:"login" graphql:"login"`
		} `json:"viewer" graphql:"viewer"`
	}
	q, _ := graphql.Marshal(query{})
	fmt.Printf("%q\n", q.Body)
	// Output:
	// "{\n\tviewer{\n\t\tlogin\n\t}\n}"
}

type benchComment struct {
	ID     string `json:"id" graphql:"id"`
	Body   string `json:"body" graphql:"body(format:MARKDOWN)"`
	Author User   `json:"author" graphql:"author"`
}

type benchQuery struct {
	Posts struct {
		Nodes []struct {
			ID       string         `json:"id" graphql:"id"`
			Title    string         `json:"title" graphql:"title,directive='include(if:$withTitle)'"`
			Author   User           `json:"author" graphql:"author"`
			Comments []benchComment `json:"comments" graphql:"comments(first:$:Int=10, sortKey:CREATED_AT)"`
		} `json:"nodes" graphql:"nodes"`
	} `json:"posts" graphql:"posts(first:$first:Int!, after:$after:String, query:'status:published')"`
	Pinned benchComment `json:"pinned" graphql:"pinned"`
}

func BenchmarkMarshal(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := graphql.Marshal(benchQuery{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalAndQuery(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		q, err := graphql.Marshal(benchQuery{})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := q.Query("Bench"); err != nil {
			b.Fatal(err)
		}
	}
}
