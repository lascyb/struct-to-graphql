# go-struct-to-graphql

[English](README-EN.md) | 中文

一个将 Go 结构体自动转换为 GraphQL 查询字符串的包。通过类型安全的结构体定义，告别手动拼接 GraphQL 查询。

## 安装
```bash
go get github.com/lascyb/struct-to-graphql
```

## 快速上手

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
	User  User `graphql:"user"`
	List  struct {
		Nodes []struct {
			Name string `graphql:"name"`
		} `graphql:"nodes"`
	} `graphql:"list(first:10, query:$:String!, id:$id:Int!)"`
}

func main() {
	q, _ := graphql.Marshal(Query{})
	query, _ := q.Query("GetData")
	fmt.Println(query)            
}
```

输出（完整查询，含变量定义）：

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

更多示例（联合类型、Fragment 复用、匿名嵌入、Mutation、Subscription 等）见 [test](./test) 目录。

### 测试与示例（参考 [test](./test)）

| 场景 | 位置 |
|------|------|
| 可运行示例（`go doc` 中可见）与基准测试 | [example_test.go](./example_test.go) |
| 字段、参数、别名、匿名结构体、union、Fragment、边界情况等 | [test/test_graphql](./test/test_graphql)（按文件拆分，长输出位于 `testdata/*.golden`） |
| `union`、`type=xxx` 等 tag flag | [test/test_flag](./test/test_flag) |
| 常见错误用法 | [test/test_error/common_misuse_test.go](./test/test_error/common_misuse_test.go) |
| Query 列表/分页/变量默认值 | [test/test_query/discountNodes_test.go](./test/test_query/discountNodes_test.go) |
| Directives（指令） | [test/test_graphql/directive_test.go](./test/test_graphql/directive_test.go)、[test/test_error/directive_misuse_test.go](./test/test_error/directive_misuse_test.go) |
| Subscription | [test/test_graphql/subscription_test.go](./test/test_graphql/subscription_test.go) |
| Mutation | [test/test_mutation/productVariantsBulkUpdate_test.go](./test/test_mutation/productVariantsBulkUpdate_test.go) |

运行测试：

```bash
go test ./... -race                                           # 全部测试
go test ./test/test_graphql -update                           # 输出变化后重新生成 golden 文件
go test -run='^$' -fuzz=FuzzArgKeyOrder -fuzztime=20s ./core  # 模糊测试 tag 参数扫描
go test -run='^$' -bench=. .                                  # 基准测试
```

## 标签规则(参考[tagkit](https://github.com/lascyb/tagkit))
- `graphql:"fieldName"`：指定字段名；未提供 `graphql` 标签时回退到 `json` 标签（与 `encoding/json` 一致，只取逗号前的名字，如 `json:"full_name,omitempty"` → `full_name`），名字为空（如 `json:",omitempty"`）或没有标签时使用 Go 字段名。
- `graphql:"-"` 或（未写 `graphql` 标签时）`json:"-"`：跳过该字段，不出现在查询中；未导出字段同样被忽略。（**行为变更**：旧版本会忽略 `json:"-"`，以 Go 字段名查询该字段。）
- `graphql:"fieldName,alias=aliasName"`：为字段设置 GraphQL 别名，最终渲染为 `aliasName: fieldName`(要注意json标签需要指定别名，如`json:"aliasName"`)。
- `graphql:"__typename,union"`：在表示 union 的结构体中，在 `__typename` 上标记，用于生成 `... on 类型 { ... }` 与 `__typename` 选择。
- **联合类型（union）结构体约定**：
  - 除 `__typename` 外，**只接受「匿名嵌入的命名 struct」作为分支**（嵌入类型必须是已命名的 `struct`），以便与 `encoding/json` 反序列化结构一致；
  - 联合分支**不能**使用匿名 `struct`；
  - 若需让 GraphQL 的 `... on` 使用与 Go 类型名不同的名字，在**嵌入的那一行**加 `type=...`，例如：  

    ```go
    YourBranchType `graphql:",type=YourGraphQLTypeName"`
    ```
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`：支持参数，值中 `$` 作为占位符自动生成变量名（`<路径>_<参数名>`，驼峰转下划线），可用 `query:$custom` 指定变量名。参数按 tag 中的书写顺序输出，同一参数不能重复声明；字面量的写法见下方 [参数字面量](#参数字面量)。
- `graphql:"field(arg:$:Type1,arg2:$varName:Type2)"`：支持为变量指定类型，格式为 `$:Type`（匿名占位符）或 `$varName:Type`（自定义变量名），如 `query:$:String!`、`id:$id:Int!`；可追加默认值 `$first:Int=10`。
- `graphql:"field,directive='include(if:$show:Boolean!)'"`：为字段添加指令，详见下方 [指令（Directives）](#指令directives)。

> **字段平铺**：将嵌套结构体的字段平铺到父级，使用 **Go 匿名嵌入** 即可。当前实现中，**仅匿名字段**会作为内联展开；不再依赖单独的 `inline` 标记。匿名嵌入在查询生成与 `encoding/json` 反序列化中均为扁平结构，与常见用法一致。

### 参数字面量

字段参数、指令参数和变量默认值中的字面量按以下规则输出：

| tag 中的写法 | 输出 | 说明 |
|------|------|------|
| `'abc'` 或 `\"abc\"`（带引号） | `"abc"` | GraphQL 字符串；首尾空白会被去除，全空白报错 |
| `CREATED_AT`（裸标识符） | `CREATED_AT` | 原样输出，即 GraphQL 枚举值 |
| `10`、`-5`、`1.5` | 原样 | 数字（支持负数） |
| `true` / `false` / `null` | 原样 | 布尔值与 null |
| `[1,2]`、`[A,'b']` | `[1,2]`、`[A,"b"]` | 列表，元素按相同规则输出 |

- 字符串会被转义为合法的 GraphQL 字符串：`"`、`\`、换行、制表符等控制字符都会转义；tag 字符串内用 `\'`、`\"`、`\\` 表示引号和反斜杠（写在 Go struct tag 中还需要再转义一次，如 ``graphql:"search(path:'C:\\\\dir')"`` 输出 `path:"C:\\dir"`）。
- 变量默认值：`String` / `ID`（含列表、非空）类型的变量，裸标识符默认值按字符串输出（`$q:String=hello` → `$q:String="hello"`）；其他类型（如枚举）原样输出（`$sort:SortKey=CREATED_AT` → `$sort:SortKey=CREATED_AT`）。

```go
type Query struct {
	Orders []Order `graphql:"orders(first:10, query:'status:paid', sortKey:CREATED_AT, reverse:true, ids:[1,2])"`
}
// 输出：orders(first:10,query:"status:paid",sortKey:CREATED_AT,reverse:true,ids:[1,2]){ ... }
```

> **行为变更**：旧版本把所有非数字、非布尔的字面量都加上引号（`sortKey:CREATED_AT` → `sortKey:"CREATED_AT"`），这会被服务端当作字符串而拒绝。现在裸标识符按枚举原样输出。如果某个参数需要的是**字符串**，请给值加上引号：`name:'lascyb'`。

### 字段类型

- 结构体（含指针、切片、数组及其多层组合，如 `[]*T`、`[][]T`、`[3]T`）会展开为子选择集。
- `map`、`interface{}`/`any`、以及没有导出字段的结构体（如 `time.Time`）视为标量，只输出字段名。
- 循环引用（如 `type Node struct{ Children []*Node }`）会返回 `circular reference detected` 错误，请改用不含循环的结构体描述需要的层级。
- 根类型必须是结构体（或其指针 / 切片），且至少有一个可查询的字段，否则 `Marshal` 返回明确的错误。

### Fragment 命名

同一个命名结构体在查询中出现多次时会生成 Fragment：

- Fragment 名默认为 Go 类型名（首字母大写），**不再带包名前缀**，如 `UserInfo`；类型条件（`on Xxx`）为 Go 类型名，因此请让 Go 类型名与 GraphQL 类型名一致。
- 泛型类型的 Fragment 名为「类型名_类型参数短名」，如 `Page[Item]` → `Page_Item`、`Pair[string,int]` → `Pair_String_Int`，类型条件为去掉类型参数的 `Page`。
- 不同类型得到相同的名字时（如不同包下的同名类型），后完成构建的类型追加包名后缀（如 `UserInfo_models`），仍然冲突时继续追加 `_2`、`_3`……；分配顺序固定，结果稳定。
- 只通过匿名嵌入（字段平铺）复用的类型不会生成 Fragment；union 分支类型如果在别处也被使用，分支内输出为 `... on Type { ...Type }`。

> **行为变更**：旧版本的 Fragment 名为「包名 + 类型名」（如 `Test_graphqlUserInfo`），泛型类型会生成非法名称。Fragment 名只出现在查询内部，不影响响应 JSON 的结构。

### 指令（Directives）

在 tag 中追加 `directive='...'` 标记位为字段声明指令，引号内使用与字段调用**完全相同**的语法 `name(arg:value,...)`（前导 `@` 可写可不写）：

- **可重复**：同一字段可写多个 `directive='...'`，按书写顺序输出；每个 `directive` 只能包含一个指令。
- **参数值**：支持字面量（`true`、`60`、枚举 `PUBLIC`、字符串 `"..."`，规则同 [参数字面量](#参数字面量)）以及与字段参数一致的变量占位符 `$`、`$name`、`$:Type`、`$name:Type`、`$name:Type=默认值`，变量会自动加入操作头部的变量定义（匿名 `$` 的变量名为 `<路径>_<指令名>_<参数名>`）。
- **`@include` / `@skip`**：必须且只能包含 `if` 参数，取值为布尔字面量或变量；变量未声明类型时自动推断为 `Boolean!`；同一字段上不能重复。
- **自定义指令**：任意指令名与参数，如 `directive='cached(ttl:60)'`；无参指令可写成 `directive=live`。
- **适用位置**：普通字段、嵌套结构体字段、Fragment 内的字段，以及 union 分支（输出为 `... on Type @include(if:$x) { ... }`，写在嵌入行，如 ``graphql:",type=Image,directive='include(if:$withImages)'"``）。字段平铺用的匿名嵌入不支持指令。
- 引号内若需字符串字面量，使用双引号并在 struct tag 中转义，如 ``graphql:"summary,directive='truncate(suffix:\"...\")'"``。

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

`q.Query("GetUser")` 输出：

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

## 输出结构
- `Graphql.Body`：完整查询体字符串。
- `Graphql.Variables`：占位符变量列表（`graphql.Variable`：Name 为 `$xxx`，Paths 为出现位置的层级路径，Type 为变量类型如 `String!`、`Int!`，HasDefault/Default 为是否有默认值及其 GraphQL 写法），按首次出现的顺序排列。
- `Graphql.Fragments`：去重生成的 Fragment 定义（`graphql.Fragment`：Name、Type、Body），按首次使用的顺序排列。
- `Graphql.Query(name string)`：组装完整的 GraphQL 查询字符串，包含操作声明、变量定义、查询体和 Fragments。
- `Graphql.Mutation(name string)`：组装完整的 GraphQL 变更字符串，包含操作声明、变量定义、查询体和 Fragments。
- `Graphql.Subscription(name string)`：组装完整的 GraphQL 订阅字符串，包含操作声明、变量定义、查询体和 Fragments。

> **输出稳定**：字段参数按 tag 中的书写顺序输出，变量定义按首次出现的顺序输出，Fragment 按首次使用的顺序输出；同一结构体多次生成的结果完全一致，便于快照测试与缓存。

## 格式化
默认缩进为两个空格，可通过 `graphql.SetIndent("    ")` 覆盖。`SetIndent` 修改的是全局设置，并发安全；每次 `Marshal` 在开始时读取一次缩进，已经在执行的 `Marshal` 不受影响。

`Marshal` 本身没有共享状态，可以在多个 goroutine 中并发调用。

## GraphQL 功能支持
- [x] **Fields（字段）** - 查询对象字段，支持嵌套查询
- [x] **Arguments（参数）** - 字段参数支持，支持静态值和变量占位符
- [x] **Aliases（别名）** - 字段别名，使用 `alias=aliasName` 语法
- [x] **Variables（变量）** - 变量占位符，支持 `$` 匿名占位符和自定义变量名
- [x] **Fragments（片段）** - 可复用片段，自动生成和去重
- [x] **Inline Fragments（内联片段）** - 联合类型支持，使用 `__typename` 字段中的 `union` 标记自动生成
- [x] **Meta fields（元字段）** - 支持 `__typename` 字段
- [x] **Operation type and name（操作类型和名称）** - 支持生成完整的操作声明（如 `query GetUser { ... }`），通过 `Query(name)` 方法
- [x] **Variable types（变量类型）** - 支持为变量指定类型（如 `$query:String!`、`$id:Int!`）
- [x] **Variable definitions（变量定义）** - 支持生成变量定义部分（如 `($episode: Episode)`），通过 `Query(name)` 方法自动生成
- [x] **Mutations（变更）** - 支持生成 mutation 操作，通过 `Mutation(name)` 方法
- [x] **Default variables（默认变量值）** - 支持变量默认值（如 `$episode: Episode = JEDI`）
- [x] **Directives（指令）** - 字段与 union 分支指令，支持 `@include`、`@skip` 及自定义指令，使用 `directive='...'` 语法
- [x] **Subscriptions（订阅）** - 支持生成 subscription 操作，通过 `Subscription(name)` 方法
