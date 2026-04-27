# go-struct-to-graphql

English | [中文](README.md)

A package that automatically converts Go structs to GraphQL query strings. Say goodbye to manually concatenating GraphQL queries with type-safe struct definitions.

## Installation
```bash
go get github.com/lascyb/struct-to-graphql
```

## Quick Start

```go
package main

import (
	"fmt"
	graphql "github.com/lascyb/struct-to-graphql"
)

type User struct {
	ID   string `graphql:"id"`
	Name string `graphql:"name"`
}

type Query struct {
	User User `graphql:"user"`
	List struct {
		Nodes []struct {
			Name string `graphql:"name"`
		} `graphql:"nodes"`
	} `graphql:"list(first:10, query:$:String!, id:$id:Int!)"`
}

func main() {
	q, _ := graphql.Marshal(Query{})
	fmt.Println(q.Body)           // query body
	query, _ := q.Query("GetData")
	fmt.Println(query)            // full query (variable definitions + body + fragments)
}
```

**Run result:**

`q.Body` (query body):

```text
{
  user{
    id
    name
  }
  list(query:$list_query,id:$id,first:10){
    nodes{
      name
    }
  }
}
```

`q.Query("GetData")` (full query with variable definitions):

```text
query GetData($list_query:String!,$id:Int!) {
  user{
    id
    name
  }
  list(query:$list_query,id:$id,first:10){
    nodes{
      name
    }
  }
}
```

For more examples (unions, fragment reuse, anonymous embedding, mutation, etc.) see the [test](./test) directory.

### Tests and examples (see [test](./test))

| Scenario | Location |
|----------|----------|
| General GraphQL feature tests (split by file) | [test/test_graphql](./test/test_graphql) |
| `union`, `type=...` and other tag flags | [test/test_flag](./test/test_flag) |
| Common misuses and expected errors | [test/test_error/common_misuse_test.go](./test/test_error/common_misuse_test.go) |
| Query list / pagination / variable defaults | [test/test_query/discountNodes_test.go](./test/test_query/discountNodes_test.go) |
| Mutation | [test/test_mutation/productVariantsBulkUpdate_test.go](./test/test_mutation/productVariantsBulkUpdate_test.go) |

Run tests: `go test ./test/...`, or open the corresponding `_test.go` files for full struct definitions and generated output.

## Tag Rules (refer to [tagkit](https://github.com/lascyb/tagkit))
- `graphql:"fieldName"`: Specifies the field name; falls back to `json` tag if not provided, then to the field name.
- `graphql:"fieldName,alias=aliasName"`: Sets a GraphQL alias for the field, rendered as `aliasName: fieldName`. (Note: the json tag needs to specify the alias, such as `json:"aliasName"`)
- `graphql:"__typename,union"`: On the struct that represents a union, mark the `__typename` field; used to emit `__typename` and `... on Type { ... }` selections.
- **Union struct conventions**:
  - Other than `__typename`, every branch must be a **named struct type embedded with an anonymous field** (so JSON unmarshalling matches the response shape and each branch is a real Go type);
  - Anonymous `struct` branches are not allowed;
  - To use a different GraphQL type name than the Go type name, set `type=...` on the **embed line**, e.g.:

    ```go
    YourBranchType `graphql:",type=YourGraphQLTypeName"`
    ```
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`: Supports parameters, `$` in values acts as a placeholder that automatically generates variable names, use `query:$custom` to specify a custom variable name.
- `graphql:"field(arg:$:Type1,arg2:$varName:Type2)"`: Supports specifying variable types, format is `$:Type` (anonymous placeholder) or `$varName:Type` (custom variable name), e.g., `query:$:String!`, `id:$id:Int!`.

> **Field flattening**: To flatten nested struct fields to the parent level, use Go **anonymous embedding**. The builder treats **only anonymous fields** as inline expansion; a separate `inline` tag flag is not used. This matches common `encoding/json` behaviour.

## Output Structure
- `Graphql.Body`: Complete query body string.
- `Graphql.Variables`: Placeholder variable list (Name is `$xxx`, Path represents the hierarchical path, Type is the variable type such as `String!`, `Int!`).
- `Graphql.Fragments`: Deduplicated generated Fragment definitions.
- `Graphql.Query(name string)`: Assembles a complete GraphQL query string, including operation declaration, variable definitions, query body, and Fragments.
- `Graphql.Mutation(name string)`: Assembles a complete GraphQL mutation string, including operation declaration, variable definitions, query body, and Fragments.

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
- [x] **Operation type and name** - Support for generating complete operation declarations (e.g., `query GetUser { ... }`), via `Query(name)` method
- [x] **Variable types** - Support for specifying variable types (e.g., `$query:String!`, `$id:Int!`)
- [x] **Variable definitions** - Support for generating variable definitions (e.g., `($episode: Episode)`), automatically generated via `Query(name)` method
- [x] **Mutations** - Support for generating mutation operations, via `Mutation(name)` method
- [x] **Default variables** - Support for variable default values (e.g., `$episode: Episode = JEDI`)
- [ ] **Directives** - Directive support (e.g., `@include`, `@skip`, etc.)
- [ ] **Subscriptions** - Support for generating subscription operations
