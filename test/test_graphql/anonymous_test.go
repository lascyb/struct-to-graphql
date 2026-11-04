package test_graphql

import "testing"

// ========== 匿名结构体、别名、匿名嵌入（字段平铺） ==========

type AnonymousQuery struct {
	ID       string `json:"id" graphql:"id"`
	Settings struct {
		Theme    string `json:"theme" graphql:"theme"`
		Language string `json:"language" graphql:"language"`
	} `json:"settings" graphql:"settings"`
	Metadata struct {
		CreatedAt string `json:"createdAt" graphql:"createdAt"`
	} `json:"metadata" graphql:"metadata"`
}

type AnonymousNestedQuery struct {
	Data struct {
		User struct {
			Name string `json:"name" graphql:"name"`
			Age  int    `json:"age" graphql:"age"`
		} `json:"user" graphql:"user"`
		Meta struct {
			CreatedAt string `json:"createdAt" graphql:"createdAt"`
			UpdatedAt string `json:"updatedAt" graphql:"updatedAt"`
		} `json:"meta" graphql:"meta"`
	} `json:"data" graphql:"data"`
}

type AnonymousSliceQuery struct {
	Tags []struct {
		Name  string `json:"name" graphql:"name"`
		Color string `json:"color" graphql:"color"`
	} `json:"tags" graphql:"tags"`
	Edges []struct {
		Node struct {
			ID    string `json:"id" graphql:"id"`
			Title string `json:"title" graphql:"title"`
		} `json:"node" graphql:"node"`
	} `json:"edges" graphql:"edges"`
}

// Go 中每个匿名结构体都是不同的类型，即使字段相同也不会生成 Fragment
type AnonymousFragmentQuery struct {
	Author struct {
		ID   string `json:"id" graphql:"id"`
		Name string `json:"name" graphql:"name"`
	} `json:"author" graphql:"author"`
	Reviewer struct {
		ID   string `json:"id" graphql:"id"`
		Name string `json:"name" graphql:"name"`
	} `json:"reviewer" graphql:"reviewer"`
}

type AliasQuery struct {
	ID          string `json:"id" graphql:"id"`
	DisplayName string `json:"displayName" graphql:"name,alias=displayName"`
	UserEmail   string `json:"userEmail" graphql:"email,alias=userEmail"`
}

type AnonymousAliasQuery struct {
	User struct {
		UserName string `json:"userName" graphql:"name,alias=userName"`
		Email    string `json:"email" graphql:"email"`
	} `json:"user" graphql:"user"`
	Profile struct {
		Age int `json:"age" graphql:"age"`
	} `json:"profile" graphql:"info,alias=profile"`
}

type EmbedAddress struct {
	City   string `json:"city" graphql:"city"`
	Street string `json:"street" graphql:"street"`
}

type EmbedQuery struct {
	Name         string `json:"name" graphql:"name"`
	EmbedAddress        // 匿名嵌入：字段平铺到父级
}

type EmbeddedMeta struct {
	CreatedAt string `json:"createdAt" graphql:"createdAt"`
	UpdatedAt string `json:"updatedAt" graphql:"updatedAt"`
}

// 匿名嵌入可以多层：EmbeddedQuery → EmbedQuery → EmbedAddress
type EmbeddedQuery struct {
	ID string `json:"id" graphql:"id"`
	EmbeddedMeta
	*EmbedQuery
}

func TestAnonymousAliasEmbedOutput(t *testing.T) {
	t.Parallel()
	runOutputCases(t, []outputCase{
		{name: "AnonymousStruct", v: AnonymousQuery{}, opName: "AnonymousTest", want: `query AnonymousTest {
  id
  settings{
    theme
    language
  }
  metadata{
    createdAt
  }
}`},
		{name: "AnonymousNested", v: AnonymousNestedQuery{}, opName: "AnonymousNestedTest", want: `query AnonymousNestedTest {
  data{
    user{
      name
      age
    }
    meta{
      createdAt
      updatedAt
    }
  }
}`},
		{name: "AnonymousSlice", v: AnonymousSliceQuery{}, opName: "AnonymousSliceTest", want: `query AnonymousSliceTest {
  tags{
    name
    color
  }
  edges{
    node{
      id
      title
    }
  }
}`},
		{name: "AnonymousNoFragment", v: AnonymousFragmentQuery{}, opName: "AnonymousFragmentTest", want: `query AnonymousFragmentTest {
  author{
    id
    name
  }
  reviewer{
    id
    name
  }
}`},
		{name: "Alias", v: AliasQuery{}, opName: "AliasTest", want: `query AliasTest {
  id
  displayName:name
  userEmail:email
}`},
		{name: "AnonymousAlias", v: AnonymousAliasQuery{}, opName: "AnonymousAliasTest", want: `query AnonymousAliasTest {
  user{
    userName:name
    email
  }
  profile:info{
    age
  }
}`},
		{name: "Embedded", v: EmbedQuery{}, opName: "EmbedTest", want: `query EmbedTest {
  name
  city
  street
}`},
		{name: "EmbeddedMultiLevel", v: EmbeddedQuery{}, opName: "EmbeddedTest", want: `query EmbeddedTest {
  id
  createdAt
  updatedAt
  name
  city
  street
}`},
	})
}
