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
| `union`, `type=...`, `fragment=...` and other tag flags, `GraphQLType()` / `GraphQLFragment()` | [test/test_graphql/naming_test.go](./test/test_graphql/naming_test.go), [test/test_flag](./test/test_flag) |
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
  - Alternatively, make the branch struct implement `GraphQLType() string` (see [Custom fragment names and type conditions](#custom-fragment-names-and-type-conditions)), so no union needs `type=`; a `type=` on the embed line takes precedence.
  - For a generic struct used as a branch, `... on` defaults to the type name without type parameters (`Box[int]` → `Box`).
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`: Supports parameters, `$` in values acts as a placeholder that automatically generates variable names (`<path>_<argName>`, camelCase converted to snake_case), use `query:$custom` to specify a custom variable name. Arguments are emitted in the order written and an argument may not be declared twice; see [Argument literals](#argument-literals) for how literal values are rendered.
- `graphql:"field(arg:$:Type1,arg2:$varName:Type2)"`: Supports specifying variable types, format is `$:Type` (anonymous placeholder) or `$varName:Type` (custom variable name), e.g., `query:$:String!`, `id:$id:Int!`; a default value can be appended, e.g. `$first:Int=10`.
- `graphql:"field,type=User,fragment=UserFields"`: sets the fragment type condition and name for a reused struct, mainly for third-party types you cannot add methods to; see [Custom fragment names and type conditions](#custom-fragment-names-and-type-conditions). Values of `type=` and `fragment=` must be valid GraphQL names (`[_A-Za-z][_0-9A-Za-z]*`, and a fragment name cannot be `on`), otherwise `Marshal` returns an error.
- `graphql:"field,directive='include(if:$show:Boolean!)'"`: Adds a directive to the field, see [Directives](#directives) below.

> **Field flattening**: To flatten nested struct fields to the parent level, use Go **anonymous embedding**. The builder treats **only anonymous fields** as inline expansion; a separate `inline` tag flag is not used. This matches common `encoding/json` behaviour.

### Argument literals

Literal values in field arguments, directive arguments and variable defaults are rendered as follows:

| Written in the tag | Output | Notes |
|------|------|------|
| `'abc'` or `\"abc\"` (quoted) | `"abc"` | GraphQL string; leading/trailing whitespace is trimmed, an all-whitespace value is an error |
| `CREATED_AT` (bare identifier) | `CREATED_AT` | emitted as is, i.e. a GraphQL enum value |
| `10`, `-5`, `1.5`, `1e3` | as is | numbers (negative numbers and exponents, GraphQL rules) |
| `true` / `false` / `null` | as is | booleans and null |
| `[1,2]`, `[A,'b']` | `[1,2]`, `[A,"b"]` | lists; elements follow the same rules and must share depth / kind |

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

A named struct that appears more than once in a query is emitted as a fragment (disable with `graphql.WithFragment(false)`, see [Marshal options](#marshal-options)):

- The fragment name is the Go type name (first letter upper-cased) **without the package prefix**, e.g. `UserInfo`. The type condition (`on Xxx`) is the Go type name by default; when they differ, implement `GraphQLType()` or set `type=X` on the field (see [Custom fragment names and type conditions](#custom-fragment-names-and-type-conditions)), or disable fragments with `WithFragment(false)`.
- Generic types are named `TypeName_TypeArgShortNames`, e.g. `Page[Item]` → `Page_Item`, `Pair[string,int]` → `Pair_String_Int`; the type condition is the name without type parameters (`Page`).
- When two different types end up with the same name (e.g. same-named types from different packages), the type that finishes building later gets the package name appended (`UserInfo_models`), then `_2`, `_3`, ... if still taken. The assignment order is fixed, so output stays deterministic.
- Types reused only through anonymous embedding (field flattening) never become fragments; a union branch type that is also used elsewhere is rendered as `... on Type { ...Type }`.

### Custom fragment names and type conditions

When the Go type name differs from the schema type name (e.g. `UserDTO` in Go, `User` in the schema), have the struct implement these interfaces (value or pointer receiver both work):

| Mechanism | Sets | Used in |
| --- | --- | --- |
| `GraphQLType() string` (`graphql.TypeNamer`) | schema type name | `fragment X on <Type>`, union branch `... on <Type>` |
| `GraphQLFragment() string` (`graphql.FragmentNamer`) | fragment name | `fragment <Name> on ...`, `...<Name>` |
| field tag `type=X` | same as `GraphQLType()`, for third-party types you cannot add methods to | same |
| field tag `fragment=X` | same as `GraphQLFragment()`, for third-party types | same |

```go
type UserDTO struct {
	ID   string `json:"id" graphql:"id"`
	Name string `json:"name" graphql:"name"`
}

func (UserDTO) GraphQLType() string { return "User" }           // fragment UserDTO on User
func (UserDTO) GraphQLFragment() string { return "UserFields" } // optional: fragment UserFields on User

// Third-party type: put the tags on any referencing field
Author pkg.Account `graphql:"author,type=User,fragment=UserFields"`
```

Rules:

- Returned values are trimmed, and an empty string means "not set". Invalid names (e.g. `bad-name`, `123`, or `on` as a fragment name) make `Marshal` return an error.
- **Union branch `... on <Type>`**: `type=X` on the embed line > the struct's `GraphQLType()` > the Go type name.
- **Fragment type condition `on <Type>`**:
  - If the struct implements `GraphQLType()`, that is used. A `type=X` on a regular (non-union) field must then match it, or `Marshal` returns an error. A `type=X` on a union branch only sets that branch's `... on` and may differ, e.g. `GraphQLType()` returns the interface `Node` while the branches are `... on BranchA { ...SharedNode }` and `... on BranchB { ...SharedNode }`.
  - Otherwise the `type=X` of the fields referencing the struct is used. Setting it on one field, or the same value on several, is fine. Different values are an error because only one fragment is generated; implement `GraphQLType()` or use `WithFragment(false)` instead.
  - With neither, the Go type name is used.
- **Fragment name**: `GraphQLFragment()` > a consistent `fragment=X` on the referencing fields (an error if they differ) > auto-generated.
  - With only `GraphQLType()`, the fragment name stays the Go type name, so `UserCardDTO` and `UserDetailDTO` can both map to `User` without clashing: `fragment UserCardDTO on User`, `fragment UserDetailDTO on User`.
  - Explicit names are kept exactly, and two types declaring the same explicit name is an error. When an auto-generated name equals an explicit one, the auto-generated side gets the package suffix, regardless of field order.
- With `WithFragment(false)` no fragments are generated, so these conflict checks are skipped; name validation still applies.

> **Behavior change**: a `type=` value that is not a valid GraphQL name (e.g. `type='bad-type'`, `type=123`) used to be emitted as-is or ignored and is now an error. When a union branch renamed with `type=X` is also reused elsewhere, its fragment used to be `on <Go type name>` and is now `on X`.

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

## Dependency upgrade: tagkit v1.1.0 (behavior changes)

Tag parsing is done by [tagkit](https://github.com/lascyb/tagkit), now upgraded to v1.1.1 (the behavior changes came in v1.1.0; v1.1.1 is a docs-only fix). Argument order, literal kinds (string / enum / number / boolean / null / list), list arguments and string escapes now come straight from tagkit; this library no longer re-scans the raw tag. **Output for valid tags is unchanged.** The differences are limited to invalid input that used to be accepted silently:

| Tag | Before | Now |
|------|------|------|
| `f(n:.5)`, `f(a:007)` | rendered `5`, `007` | error: invalid number |
| `f(n:1.5e3)` | error | exponents supported, rendered as `1.5e3` |
| `f(a:$a:Int b:2)` (missing comma) | treated as two arguments | error |
| `f(ids:[1,[2,3]])`, `f(a:[A,1])` | rendered as is | error: list elements must share depth / kind |
| `$a:Int=x`, `$a:String=1`, `$a:String!=null`, `$a:[Int]=1`, `$a:Int=99999999999` | rendered as is | error: default does not match the declared type |
| `$a:[Int!!]` | rendered `[Int!]` | error: invalid type |
| `f(é:1)` (non-ASCII identifier) | invalid UTF-8 output | error (quote non-ASCII text) |

- **Error messages**: tag syntax errors are reported by tagkit as `field [<field>]: <message> at position <byte offset>`; use `errors.As(err, &se)` with `se *tagkit.SyntaxError` to get the position. The former `invalid tag "..."` and `参数 x 重复声明` errors became `unterminated string at position N` and `duplicate argument "x" at position N`.
- **`Arg.Value`** (the embedded `tagkit.ArgValue` of `core.Arg`): for list literal arguments `Value` changed from the rendered string (e.g. `"[1,2]"`) to `[]interface{}`, and for `null` from the string `"null"` to `nil`; use `Arg.Literal` for the GraphQL form. `Arg.ArgValue` gains `Kind`, `Literal` and `Default`.

## Marshal options

`Marshal(v any, opts ...graphql.Option)` accepts optional arguments; without them it behaves exactly as before. If the same option is passed more than once, the last one wins; `nil` options are ignored.

| Option | Default | Effect |
| --- | --- | --- |
| `graphql.WithFragment(enable bool)` | `true` | Whether named structs referenced more than once are extracted into top-level `fragment`s |
| `graphql.WithIndent(val string)` | the global indentation set by `SetIndent` (two spaces by default) | Per-call indentation unit |

```go
q, err := graphql.Marshal(Query{}, graphql.WithFragment(false), graphql.WithIndent("    "))
```

**`WithFragment(false)`**: reused structs are expanded inline as `{ ... }` at every occurrence, `Graphql.Fragments` is an empty slice, and the document contains no `fragment` definitions.

- Reused structs embedded anonymously are flattened into the parent as usual instead of emitting `...TypeName`.
- Union branches still render as `... on Type { ... }`, with the fields expanded directly inside instead of `... on Type { ...Type }`.
- Anonymous variables (`$:Type`) inside a reused struct are named after each occurrence's actual path, e.g. `$author_avatar_size` and `$others_avatar_size`; named variables (`$id:ID!`) are still deduplicated, and `Paths` records every occurrence.
- Since no `fragment ... on Type` is emitted, the Go type name of a reused struct (e.g. `UserDTO`) does not need to match the GraphQL type name in the server schema.
- The trade-off is a longer query text, especially with heavy reuse.

**`WithIndent(val)`**: overrides the indentation for this call (fragment bodies included) without touching the global setting, so concurrent and later `Marshal` calls are unaffected. `WithIndent("")` disables indentation.

## Output Structure
- `Graphql.Body`: Complete query body string.
- `Graphql.Variables`: Placeholder variable list (`graphql.Variable`: Name is `$xxx`, Paths lists the hierarchical paths where it is used, Type is the variable type such as `String!`, `Int!`, HasDefault/Default tell whether it has a default value and its GraphQL literal), in order of first appearance.
- `Graphql.Fragments`: Deduplicated generated Fragment definitions (`graphql.Fragment`: Name, Type, Body), in order of first use; an empty slice with `WithFragment(false)`.
- `Graphql.Query(name string)`: Assembles a complete GraphQL query string, including operation declaration, variable definitions, query body, and Fragments.
- `Graphql.Mutation(name string)`: Assembles a complete GraphQL mutation string, including operation declaration, variable definitions, query body, and Fragments.
- `Graphql.Subscription(name string)`: Assembles a complete GraphQL subscription string, including operation declaration, variable definitions, query body, and Fragments.

> **Deterministic output**: field arguments are emitted in the order they are written in the tag, variable definitions in order of first appearance, and fragments in order of first use; generating the same struct repeatedly always yields identical output, which makes snapshot testing and caching easy.

## Formatting
Default indentation is two spaces. There are two ways to change it:

- `graphql.Marshal(v, graphql.WithIndent("    "))`: applies to this call only and takes precedence; recommended.
- `graphql.SetIndent("    ")`: changes the global default for every later `Marshal` call that does not pass `WithIndent`. `SetIndent` is safe for concurrent use; each `Marshal` call reads the indentation once when it starts, so calls already in progress are not affected.

`Marshal` itself has no shared state and may be called from multiple goroutines concurrently.

## GraphQL Feature Support
- [x] **Fields** - Query object fields with nested selection sets
- [x] **Arguments** - Field arguments with static values and variable placeholders
- [x] **Aliases** - Field aliases using `alias=aliasName` syntax
- [x] **Variables** - Variable placeholders supporting `$` anonymous placeholders and custom variable names
- [x] **Fragments** - Reusable fragments with automatic generation and deduplication; can be disabled with `WithFragment(false)`
- [x] **Inline Fragments** - Union type support, automatically generated using `union` flag
- [x] **Meta fields** - Support for `__typename` field
- [x] **Operation type and name** - Support for generating complete operation declarations (e.g., `query GetUser { ... }`), via `Query(name)` method
- [x] **Variable types** - Support for specifying variable types (e.g., `$query:String!`, `$id:Int!`)
- [x] **Variable definitions** - Support for generating variable definitions (e.g., `($episode: Episode)`), automatically generated via `Query(name)` method
- [x] **Mutations** - Support for generating mutation operations, via `Mutation(name)` method
- [x] **Default variables** - Support for variable default values (e.g., `$episode: Episode = JEDI`)
- [x] **Directives** - Directives on fields and union branches, supporting `@include`, `@skip` and custom directives via the `directive='...'` syntax
- [x] **Subscriptions** - Support for generating subscription operations, via `Subscription(name)` method
