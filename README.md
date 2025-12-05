# go-struct-to-graphql
一个将 Go 结构体自动转换为 GraphQL 查询字符串的包。通过类型安全的结构体定义，告别手动拼接 GraphQL 查询。

## 安装
```bash
go get github.com/lascyb/struct-to-graphql
```

## 快速上手
```go
type Query struct {
	Field1 string `graphql:"field1"`
	// 支持参数占位符，$ 表示匿名占位符，自动生成变量名
	List struct {
		Nodes []struct {
			Name string `graphql:"name"`
		} `graphql:"nodes"`
	} `graphql:"list(first:10,query:$,id:$id)"`
}

func main() {
	q, _ := graphql.Marshal(Query{})
	fmt.Println(q.Body)       // 打印查询体
	fmt.Println(q.Variables)  // 占位符变量列表
	fmt.Println(q.Fragments)  // 去重生成的 Fragment
}
```

### 生成示例（同 [test/main.go](./test/main.go)）
使用测试里的 `TestStruct`：

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

### 典型输出：
#### 请求主体
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

## 标签规则(参考[tagkit](https://github.com/lascyb/tagkit))
- `graphql:"fieldName"`：指定字段名；未提供时回退到 `json` 标签，再回退到字段名。
- `graphql:"fieldName,inline"`：内联展开匿名或标记字段。
- `graphql:"__typename,union"`：标记联合类型分支，生成 inline fragment。
- `graphql:"field(arg1:1,arg2:$,arg3:$value3,...)"`：支持参数，值中 `$` 作为占位符自动生成变量名，可用 `query:$custom` 指定变量名。

## 输出结构
- `Graphql.Body`：完整查询体字符串。
- `Graphql.Variables`：占位符变量列表（Name 为 `$xxx`，Path 表示层级路径）。
- `Graphql.Fragments`：去重生成的 Fragment 定义。

## 格式化
默认缩进为两个空格，可通过 `graphql.SetIndent("    ")` 覆盖。
