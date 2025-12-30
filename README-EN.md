# go-struct-to-graphql

English | [中文](README.md)

A package that automatically converts Go structs to GraphQL query strings. Say goodbye to manually concatenating GraphQL queries with type-safe struct definitions.

## Installation
```bash
go get github.com/lascyb/struct-to-graphql
```

## Quick Start
```go
import (
	graphql "github.com/lascyb/struct-to-graphql"
)

type Foo struct {
	Bar string `graphql:"bar"`
}
type Query struct {
	Field1 string `graphql:"field1"`
	List   struct {
		Foo1  Foo `graphql:"foo1"`
		Foo2  Foo `graphql:"foo2"`
		Nodes []struct {
			Name string `graphql:"name"`
		} `graphql:"nodes"`
	} `graphql:"list(first:10,query:$:String!,id:$id:Int!)"` // Supports parameter placeholders and types, $ represents anonymous placeholder, automatically generates variable names based on hierarchy: $list_query, type is specified via :Type
}


func main() {
	q, _ := graphql.Marshal(Query{})
	
	// Method 1: Get each part separately
	fmt.Println(strings.Repeat("-", 15), "Query Body", strings.Repeat("-", 15))
	fmt.Println(q.Body) // Print query body
	fmt.Println(strings.Repeat("-", 15), "Placeholder Variables", strings.Repeat("-", 15))
	for _, variable := range q.Variables {
		fmt.Println("Name:", variable.Name, ",Type:", variable.Type, ",Paths:", variable.Paths) // Placeholder variable list
	}
	fmt.Println(strings.Repeat("-", 15), "Deduplicated Fragments", strings.Repeat("-", 15))
	for _, fragment := range q.Fragments {
		fmt.Println(fragment.Body) // Deduplicated generated Fragments
	}
	
	// Method 2: Use Query method to assemble complete query
	query, _ := q.Query("GetData")
	fmt.Println(strings.Repeat("-", 15), "Complete Query", strings.Repeat("-", 15))
	fmt.Println(query)
--------------- Query Body ---------------
{
  field1
  list(first: 10, query: $list_query, id: $id){
    foo1{ ...MainFoo }
    foo2{ ...MainFoo }
    nodes{
      name
    }
  }
}
--------------- Placeholder Variables ---------------
Name: $list_query ,Type: String! ,Paths: [list]
Name: $id ,Type: Int! ,Paths: [list]
--------------- Deduplicated Fragments ---------------
fragment MainFoo on Foo{
  bar
}
--------------- Complete Query ---------------
fragment MainFoo on Foo{
  bar
}
query GetData(list_query: String!, id: Int!) {
  field1
  list(first: 10, query: $list_query, id: $id){
    foo1{ ...MainFoo }
    foo2{ ...MainFoo }
    nodes{
      name
    }
  }
}

```

### Generation Example (same as [test/main.go](./test/main.go))
Using `TestStruct` from the tests:

```go
type TestStruct struct {
	Field1     string          `graphql:"field1"`
	Fragment1  Fragment        `graphql:"fragment1"`
	Fragment2  Fragment        `graphql:"fragment2"`
	UnionField Union           `graphql:"unionField"`
	LineItem   LineItemConnect `graphql:"lineItem(first:10,query:$)"`
	Inline
	Inline2 Inline2 `graphql:"inline2,inline"`
	TreeField Tree  `graphql:"tree"`
}
```

### Typical Output:
#### Request Body
```text
{
  field1
  fragment1{ ...MainFragment }
  fragment2{ ...MainFragment }
  unionField{ ...MainUnion }
  lineItem(first: 10, query: $lineItem_query){ ...MainLineItemConnect }
  inlineField1
  inlineField2
  tree{
    tree1{
      union1Field1{ ...MainUnion }
      tree1Field1
      tree1Field2{
        tree2Field1
        lineItem(first: 10, query: $tree_tree1_tree1Field2_lineItem_query){ ...MainLineItemConnect }
      }
    }
  }
}
```
#### Variables
```text
- {Name:$lineItem_query Path:[tree]}
- {Name:$tree_tree1_tree1Field2_lineItem_query Path:[tree tree1 tree1Field2 lineItem]}
```
#### Fragments
- MainFragment:

```text
fragment MainFragment on Fragment{
  fragmentField1
}
```

- MainUnion:
```text
fragment MainUnion on Union{
  __typename
  ... on Union1 {
    union1Field1
  }
  ... on Union2 {
    union1Field1
  }
  ... on Fragment { ...MainFragment }
}
```

- MainLineItemConnect:
```text
fragment MainLineItemConnect on LineItemConnect{
  nodes{
    lineItemField1
  }
}
```

## Tag Rules (refer to [tagkit](https://github.com/lascyb/tagkit))
- `graphql:"fieldName"`: Specifies the field name; falls back to `json` tag if not provided, then to the field name.
- `graphql:"fieldName,inline"`: Inline expansion of anonymous or tagged fields.
- `graphql:"fieldName,alias=aliasName"`: Sets a GraphQL alias for the field, rendered as `aliasName: fieldName`. (Note: the json tag needs to specify the alias, such as `json:"aliasName"`)
- `graphql:"__typename,union"`: Marks union type branches, generates inline fragments.
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`: Supports parameters, `$` in values acts as a placeholder that automatically generates variable names, use `query:$custom` to specify a custom variable name.
- `graphql:"field(arg:$:Type1,arg2:$varName:Type2)"`: Supports specifying variable types, format is `$:Type` (anonymous placeholder) or `$varName:Type` (custom variable name), e.g., `query:$:String!`, `id:$id:Int!`.

## Output Structure
- `Graphql.Body`: Complete query body string.
- `Graphql.Variables`: Placeholder variable list (Name is `$xxx`, Path represents the hierarchical path, Type is the variable type such as `String!`, `Int!`).
- `Graphql.Fragments`: Deduplicated generated Fragment definitions.
- `Graphql.Query(name string)`: Assembles a complete GraphQL query string, including operation declaration, variable definitions, query body, and Fragments.

## Formatting
Default indentation is two spaces, can be overridden with `graphql.SetIndent("    ")`.

## GraphQL Feature Support
- [x] **Fields** - Query object fields with nested selection sets
- [x] **Arguments** - Field arguments with static values and variable placeholders
- [x] **Aliases** - Field aliases using `alias=aliasName` syntax
- [x] **Variables** - Variable placeholders supporting `$` anonymous placeholders and custom variable names
- [x] **Fragments** - Reusable fragments with automatic generation and deduplication
- [x] **Inline Fragments** - Union type support, automatically generated using `union` flag
- [x] **Meta fields** - Support for `__typename` field
- [ ] **Directives** - Directive support (e.g., `@include`, `@skip`, etc.)
- [x] **Operation type and name** - Support for generating complete operation declarations (e.g., `query GetUser { ... }`), via `Query(name)` method
- [x] **Variable types** - Support for specifying variable types (e.g., `$query:String!`, `$id:Int!`)
- [x] **Variable definitions** - Support for generating variable definitions (e.g., `($episode: Episode)`), automatically generated via `Query(name)` method
- [ ] **Default variables** - Support for variable default values (e.g., `$episode: Episode = JEDI`)
- [ ] **Mutations** - Support for generating mutation operations
- [ ] **Subscriptions** - Support for generating subscription operations
