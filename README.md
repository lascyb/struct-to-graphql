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
  list(query:$list_query,id:$id,first:10){
    nodes{
      name
    }
  }
}
```

更多示例（联合类型、Fragment 复用、匿名嵌入、Mutation 等）见 [test](./test) 目录。

### 测试与示例（参考 [test](./test)）

| 场景 | 位置 |
|------|------|
| 综合场景、GraphQL 各能力用例 | [test/test_graphql](./test/test_graphql)（按文件拆分） |
| `union`、`type=xxx` 等 tag flag | [test/test_flag](./test/test_flag) |
| 常见错误用法 | [test/test_error/common_misuse_test.go](./test/test_error/common_misuse_test.go) |
| Query 列表/分页/变量默认值 | [test/test_query/discountNodes_test.go](./test/test_query/discountNodes_test.go) |
| Mutation | [test/test_mutation/productVariantsBulkUpdate_test.go](./test/test_mutation/productVariantsBulkUpdate_test.go) |

运行测试：`go test ./test/...`，或在对应 `_test.go` 中查看完整结构体定义与生成结果。

## 标签规则(参考[tagkit](https://github.com/lascyb/tagkit))
- `graphql:"fieldName"`：指定字段名；未提供时回退到 `json` 标签，再回退到字段名。
- `graphql:"fieldName,alias=aliasName"`：为字段设置 GraphQL 别名，最终渲染为 `aliasName: fieldName`(要注意json标签需要指定别名，如`json:"aliasName"`)。
- `graphql:"__typename,union"`：在表示 union 的结构体中，在 `__typename` 上标记，用于生成 `... on 类型 { ... }` 与 `__typename` 选择。
- **联合类型（union）结构体约定**：
  - 除 `__typename` 外，**只接受「匿名嵌入的命名 struct」作为分支**（嵌入类型必须是已命名的 `struct`），以便与 `encoding/json` 反序列化结构一致；
  - 联合分支**不能**使用匿名 `struct`；
  - 若需让 GraphQL 的 `... on` 使用与 Go 类型名不同的名字，在**嵌入的那一行**加 `type=...`，例如：  

    ```go
    YourBranchType `graphql:",type=YourGraphQLTypeName"`
    ```
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`：支持参数，值中 `$` 作为占位符自动生成变量名，可用 `query:$custom` 指定变量名。
- `graphql:"field(arg:$:Type1,arg2:$varName:Type2)"`：支持为变量指定类型，格式为 `$:Type`（匿名占位符）或 `$varName:Type`（自定义变量名），如 `query:$:String!`、`id:$id:Int!`。

> **字段平铺**：将嵌套结构体的字段平铺到父级，使用 **Go 匿名嵌入** 即可。当前实现中，**仅匿名字段**会作为内联展开；不再依赖单独的 `inline` 标记。匿名嵌入在查询生成与 `encoding/json` 反序列化中均为扁平结构，与常见用法一致。

## 输出结构
- `Graphql.Body`：完整查询体字符串。
- `Graphql.Variables`：占位符变量列表（Name 为 `$xxx`，Path 表示层级路径，Type 为变量类型如 `String!`、`Int!`）。
- `Graphql.Fragments`：去重生成的 Fragment 定义。
- `Graphql.Query(name string)`：组装完整的 GraphQL 查询字符串，包含操作声明、变量定义、查询体和 Fragments。
- `Graphql.Mutation(name string)`：组装完整的 GraphQL 变更字符串，包含操作声明、变量定义、查询体和 Fragments。

## 格式化
默认缩进为两个空格，可通过 `graphql.SetIndent("    ")` 覆盖。

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
- [ ] **Directives（指令）** - 指令功能支持（如 `@include`、`@skip` 等）
- [ ] **Subscriptions（订阅）** - 支持生成 subscription 操作
