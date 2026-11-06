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

func ExampleWithFragment() {
	// 关闭 Fragment：复用的 User 在每个引用处原地展开，不生成顶层 fragment
	q, err := graphql.Marshal(Query{}, graphql.WithFragment(false))
	if err != nil {
		panic(err)
	}
	query, _ := q.Query("GetData")
	fmt.Println(query)
	fmt.Println(len(q.Fragments))
	// Output:
	// query GetData($id:ID!,$list_query:String!) {
	//   user(id:$id){
	//     id
	//     name
	//   }
	//   list(first:10,query:$list_query,sortKey:CREATED_AT,status:"active"){
	//     nodes{
	//       id
	//       name
	//     }
	//   }
	// }
	// 0
}

func ExampleWithIndent() {
	type query struct {
		Viewer struct {
			Login string `json:"login" graphql:"login"`
		} `json:"viewer" graphql:"viewer"`
	}
	// 只对本次调用生效，不影响全局 SetIndent
	q, _ := graphql.Marshal(query{}, graphql.WithIndent("\t"))
	fmt.Printf("%q\n", q.Body)
	q, _ = graphql.Marshal(query{})
	fmt.Printf("%q\n", q.Body)
	// Output:
	// "{\n\tviewer{\n\t\tlogin\n\t}\n}"
	// "{\n  viewer{\n    login\n  }\n}"
}

// AccountDTO 在 Go 里叫 AccountDTO，在 GraphQL Schema 里叫 User
type AccountDTO struct {
	ID    string `json:"id" graphql:"id"`
	Email string `json:"email" graphql:"email"`
}

// GraphQLType 指定 Fragment 的类型条件（fragment ... on User）
func (AccountDTO) GraphQLType() string { return "User" }

// GraphQLFragment 指定 Fragment 名称（可选，默认为 Go 类型名 AccountDTO）
func (AccountDTO) GraphQLFragment() string { return "AccountFields" }

func ExampleTypeNamer() {
	type query struct {
		Viewer AccountDTO   `json:"viewer" graphql:"viewer"`
		Admins []AccountDTO `json:"admins" graphql:"admins"`
	}
	q, err := graphql.Marshal(query{})
	if err != nil {
		panic(err)
	}
	out, _ := q.Query("Accounts")
	fmt.Println(out)
	// Output:
	// fragment AccountFields on User{
	//   id
	//   email
	// }
	// query Accounts {
	//   viewer{ ...AccountFields }
	//   admins{ ...AccountFields }
	// }
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

func BenchmarkMarshal_NoFragment(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := graphql.Marshal(benchQuery{}, graphql.WithFragment(false)); err != nil {
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
