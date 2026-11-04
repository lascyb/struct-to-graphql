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
  list(first:10,query:$list_query,id:$id){
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
  list(first:10,query:$list_query,id:$id){
    nodes{
      name
    }
  }
}
```

For more examples (unions, fragment reuse, anonymous embedding, mutation, subscription, etc.) see the [test](./test) directory.

### Tests and examples (see [test](./test))

| Scenario | Location |
|----------|----------|
| Runnable examples (shown in `go doc`) and benchmarks | [example_test.go](./example_test.go) |
| Fields, arguments, aliases, anonymous structs, unions, fragments, edge cases (split by file) | [test/test_graphql](./test/test_graphql) (long outputs live in `testdata/*.golden`) |
| `union`, `type=...` and other tag flags | [test/test_flag](./test/test_flag) |
| Common misuses and expected errors | [test/test_error/common_misuse_test.go](./test/test_error/common_misuse_test.go) |
| Query list / pagination / variable defaults | [test/test_query/discountNodes_test.go](./test/test_query/discountNodes_test.go) |
| Directives | [test/test_graphql/directive_test.go](./test/test_graphql/directive_test.go), [test/test_error/directive_misuse_test.go](./test/test_error/directive_misuse_test.go) |
| Subscription | [test/test_graphql/subscription_test.go](./test/test_graphql/subscription_test.go) |
| Mutation | [test/test_mutation/productVariantsBulkUpdate_test.go](./test/test_mutation/productVariantsBulkUpdate_test.go) |

Running tests:

```bash
go test ./... -race                                           # all tests
go test ./test/test_graphql -update                           # regenerate golden files after an intended output change
go test -run='^$' -fuzz=FuzzArgKeyOrder -fuzztime=20s ./core  # fuzz the tag argument scanner
go test -run='^$' -bench=. .                                  # benchmarks
```

## Tag Rules (refer to [tagkit](https://github.com/lascyb/tagkit))
- `graphql:"fieldName"`: Specifies the field name. Without a `graphql` tag it falls back to the `json` tag the same way `encoding/json` does: only the part before the first comma is used (`json:"full_name,omitempty"` → `full_name`); an empty name (`json:",omitempty"`) or no tag at all means the Go field name.
- `graphql:"-"`, or `json:"-"` when there is no `graphql` tag: skips the field entirely; unexported fields are ignored as well. (**Behavior change**: older versions ignored `json:"-"` and queried the field by its Go name.)
- `graphql:"fieldName,alias=aliasName"`: Sets a GraphQL alias for the field, rendered as `aliasName: fieldName`. (Note: the json tag needs to specify the alias, such as `json:"aliasName"`)
- `graphql:"__typename,union"`: On the struct that represents a union, mark the `__typename` field; used to emit `__typename` and `... on Type { ... }` selections.
- **Union struct conventions**:
  - Other than `__typename`, every branch must be a **named struct type embedded with an anonymous field** (so JSON unmarshalling matches the response shape and each branch is a real Go type);
  - Anonymous `struct` branches are not allowed;
  - To use a different GraphQL type name than the Go type name, set `type=...` on the **embed line**, e.g.:

    ```go
    YourBranchType `graphql:",type=YourGraphQLTypeName"`
    ```
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`: Supports parameters, `$` in values acts as a placeholder that automatically generates variable names (`<path>_<argName>`, camelCase converted to snake_case), use `query:$custom` to specify a custom variable name. Arguments are emitted in the order written and an argument may not be declared twice; see [Argument literals](#argument-literals) for how literal values are rendered.
- `graphql:"field(arg:$:Type1,arg2:$varName:Type2)"`: Supports specifying variable types, format is `$:Type` (anonymous placeholder) or `$varName:Type` (custom variable name), e.g., `query:$:String!`, `id:$id:Int!`; a default value can be appended, e.g. `$first:Int=10`.
- `graphql:"field,directive='include(if:$show:Boolean!)'"`: Adds a directive to the field, see [Directives](#directives) below.

> **Field flattening**: To flatten nested struct fields to the parent level, use Go **anonymous embedding**. The builder treats **only anonymous fields** as inline expansion; a separate `inline` tag flag is not used. This matches common `encoding/json` behaviour.

### Argument literals

Literal values in field arguments, directive arguments and variable defaults are rendered as follows:

| Written in the tag | Output | Notes |
|------|------|------|
| `'abc'` or `\"abc\"` (quoted) | `"abc"` | GraphQL string; leading/trailing whitespace is trimmed, an all-whitespace value is an error |
| `CREATED_AT` (bare identifier) | `CREATED_AT` | emitted as is, i.e. a GraphQL enum value |
| `10`, `-5`, `1.5` | as is | numbers (negative numbers supported) |
| `true` / `false` / `null` | as is | booleans and null |
| `[1,2]`, `[A,'b']` | `[1,2]`, `[A,"b"]` | lists; elements follow the same rules |

- Strings are escaped into valid GraphQL strings: `"`, `\`, newlines, tabs and other control characters are escaped. Inside a tag string, use `\'`, `\"` and `\\` for quotes and backslashes (inside a Go struct tag they need escaping once more, e.g. ``graphql:"search(path:'C:\\\\dir')"`` renders `path:"C:\\dir"`).
- Variable defaults: for variables whose type is `String` or `ID` (including list / non-null forms) a bare identifier default is rendered as a string (`$q:String=hello` → `$q:String="hello"`); for any other type (e.g. an enum) it is emitted as is (`$sort:SortKey=CREATED_AT` → `$sort:SortKey=CREATED_AT`).

```go
type Query struct {
	Orders []Order `graphql:"orders(first:10, query:'status:paid', sortKey:CREATED_AT, reverse:true, ids:[1,2])"`
}
// renders: orders(first:10,query:"status:paid",sortKey:CREATED_AT,reverse:true,ids:[1,2]){ ... }
```

> **Behavior change**: older versions quoted every literal that was not a number or boolean (`sortKey:CREATED_AT` → `sortKey:"CREATED_AT"`), which servers reject for enum arguments. Bare identifiers are now emitted unquoted as enum values. If an argument expects a **string**, quote the value: `name:'lascyb'`.

### Field types

- Structs (including pointers, slices, arrays and nested combinations such as `[]*T`, `[][]T`, `[3]T`) are expanded into sub-selections.
- `map`, `interface{}`/`any` and structs without exported fields (such as `time.Time`) are treated as scalars: only the field name is emitted.
- Circular references (e.g. `type Node struct{ Children []*Node }`) return a `circular reference detected` error; describe the depth you need with non-recursive structs instead.
- The root type must be a struct (or a pointer / slice of one) with at least one queryable field; otherwise `Marshal` returns a descriptive error.

### Fragment naming

A named struct that appears more than once in a query is emitted as a fragment:

- The fragment name is the Go type name (first letter upper-cased) **without the package prefix**, e.g. `UserInfo`. The type condition (`on Xxx`) is the Go type name, so keep Go type names aligned with GraphQL type names.
- Generic types are named `TypeName_TypeArgShortNames`, e.g. `Page[Item]` → `Page_Item`, `Pair[string,int]` → `Pair_String_Int`; the type condition is the name without type parameters (`Page`).
- When two different types end up with the same name (e.g. same-named types from different packages), the type that finishes building later gets the package name appended (`UserInfo_models`), then `_2`, `_3`, ... if still taken. The assignment order is fixed, so output stays deterministic.
- Types reused only through anonymous embedding (field flattening) never become fragments; a union branch type that is also used elsewhere is rendered as `... on Type { ...Type }`.

> **Behavior change**: fragments used to be named `<package><Type>` (e.g. `Test_graphqlUserInfo`), and generic types produced invalid names. Fragment names only appear inside the query and do not affect the shape of the JSON response.

### Directives

Append a `directive='...'` flag to the tag to declare a directive on a field. Inside the quotes, use **exactly the same** syntax as a field call, `name(arg:value,...)` (a leading `@` is optional):

- **Repeatable**: a field may have several `directive='...'` flags; they are emitted in the order written. Each `directive` holds exactly one directive.
- **Argument values**: literals (`true`, `60`, enum `PUBLIC`, string `"..."`, following [Argument literals](#argument-literals)) and the same variable placeholders as field arguments: `$`, `$name`, `$:Type`, `$name:Type`, `$name:Type=default`. Variables are added to the operation's variable definitions automatically (an anonymous `$` is named `<path>_<directiveName>_<argName>`).
- **`@include` / `@skip`**: must have exactly one `if` argument, either a boolean literal or a variable; a variable without a declared type is inferred as `Boolean!`; they cannot be repeated on the same field.
- **Custom directives**: any directive name and arguments, e.g. `directive='cached(ttl:60)'`; a directive without arguments can be written as `directive=live`.
- **Where**: regular fields, nested struct fields, fields inside fragments, and union branches (rendered as `... on Type @include(if:$x) { ... }`, declared on the embed line, e.g. ``graphql:",type=Image,directive='include(if:$withImages)'"``). Anonymous embeds used for field flattening do not support directives.
- For string literals inside the quotes, use double quotes escaped in the struct tag, e.g. ``graphql:"summary,directive='truncate(suffix:\"...\")'"``.

```go
type Query struct {
	User struct {
		ID      string `graphql:"id"`
		Email   string `graphql:"email,directive='include(if:$withEmail:Boolean!)'"`
		Friends []struct {
			Name string `graphql:"name"`
		} `json:"topFriends" graphql:"friends(first:10),alias=topFriends,directive='skip(if:$hideFriends)',directive='cached(ttl:60)'"`
	} `graphql:"user(id:$id:ID!),directive='include(if:$withUser:Boolean=true)'"`
}
```

`q.Query("GetUser")` output:

```text
query GetUser($id:ID!,$withUser:Boolean=true,$withEmail:Boolean!,$hideFriends:Boolean!) {
  user(id:$id) @include(if:$withUser){
    id
    email @include(if:$withEmail)
    topFriends:friends(first:10) @skip(if:$hideFriends) @cached(ttl:60){
      name
    }
  }
}
```

## Output Structure
- `Graphql.Body`: Complete query body string.
- `Graphql.Variables`: Placeholder variable list (`graphql.Variable`: Name is `$xxx`, Paths lists the hierarchical paths where it is used, Type is the variable type such as `String!`, `Int!`, HasDefault/Default tell whether it has a default value and its GraphQL literal), in order of first appearance.
- `Graphql.Fragments`: Deduplicated generated Fragment definitions (`graphql.Fragment`: Name, Type, Body), in order of first use.
- `Graphql.Query(name string)`: Assembles a complete GraphQL query string, including operation declaration, variable definitions, query body, and Fragments.
- `Graphql.Mutation(name string)`: Assembles a complete GraphQL mutation string, including operation declaration, variable definitions, query body, and Fragments.
- `Graphql.Subscription(name string)`: Assembles a complete GraphQL subscription string, including operation declaration, variable definitions, query body, and Fragments.

> **Deterministic output**: field arguments are emitted in the order they are written in the tag, variable definitions in order of first appearance, and fragments in order of first use; generating the same struct repeatedly always yields identical output, which makes snapshot testing and caching easy.

## Formatting
Default indentation is two spaces, can be overridden with `graphql.SetIndent("    ")`. `SetIndent` changes a global setting and is safe for concurrent use; each `Marshal` call reads the indentation once when it starts, so calls already in progress are not affected.

`Marshal` itself has no shared state and may be called from multiple goroutines concurrently.

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
- [x] **Directives** - Directives on fields and union branches, supporting `@include`, `@skip` and custom directives via the `directive='...'` syntax
- [x] **Subscriptions** - Support for generating subscription operations, via `Subscription(name)` method
