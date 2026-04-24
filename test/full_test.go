package main

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

// --- 联合类型：内容可为文章或视频（GraphQL Union / ... on Type）---

// ArticleContent 文章内容，联合类型分支之一
type ArticleContent struct {
	Title string `json:"title" graphql:"title"`
	Body  string `json:"body" graphql:"body"`
}

// VideoContent 视频内容，联合类型分支之一
type VideoContent struct {
	Title string `json:"title" graphql:"title"`
	URL   string `json:"url" graphql:"url"`
}

// ContentUnion 内容联合类型；标记 __typename,union 后生成 ... on ArticleContent / ... on VideoContent
type ContentUnion struct {
	Typename string `json:"__typename" graphql:"__typename,union"`
	ArticleContent
	VideoContent // 与 ArticleContent 同含 json:"title"，联合解析时按 __typename 仅填充其一
}

// --- Fragment 复用：同一结构体在多处引用时生成 fragment，避免重复字段列表 ---

// UserBasic 用户基础信息；在多处引用时会生成为 fragment UserBasic on UserBasic { ... }
type UserBasic struct {
	ID    string `json:"id" graphql:"id"`
	Name  string `json:"name" graphql:"name"`
	Email string `json:"email" graphql:"email"`
}

// --- 列表/连接与参数：分页、筛选等 ---

// ProductItem 商品项
type ProductItem struct {
	ID    string `json:"id" graphql:"id"`
	Title string `json:"title" graphql:"title"`
}

// ProductConnection 商品连接；nodes 为列表字段
type ProductConnection struct {
	Nodes []ProductItem `json:"nodes" graphql:"nodes"`
}

// --- 嵌入类型：Go 匿名字段，未指定 graphql 名时类型名作为选择集，字段平铺到父级 ---

// MetaInfo 元信息；仅作嵌入类型示例，匿名嵌入后 createdAt/updatedAt 直接出现在父级
type MetaInfo struct {
	CreatedAt string `json:"createdAt" graphql:"createdAt"`
	UpdatedAt string `json:"updatedAt" graphql:"updatedAt"`
}

// --- 匿名嵌入类型：字段平铺到父级，不增加层级 ---

// AddressEmbed 地址类型；匿名嵌入时字段直接出现在父级
type AddressEmbed struct {
	City    string `json:"city" graphql:"city"`
	Street  string `json:"street" graphql:"street"`
	ZipCode string `json:"zipCode" graphql:"zipCode"`
}

// ContactEmbed 联系方式；匿名嵌入时字段平铺到父级
type ContactEmbed struct {
	Phone string `json:"phone" graphql:"phone"`
	Fax   string `json:"faxAlias" graphql:"fax,alias=faxAlias"`
}

// --- 嵌套树与联合/匿名嵌入组合 ---

// TreeNodeLeaf 树叶子节点；避免 TreeNode 递归导致循环引用
type TreeNodeLeaf struct {
	Label   string       `json:"label" graphql:"label"`
	Content ContentUnion `json:"content" graphql:"content"`
}

// TreeNodeConnection 树子节点连接
type TreeNodeConnection struct {
	Nodes []TreeNodeLeaf `json:"nodes" graphql:"nodes"`
}

// TreeNode 树节点；内含联合类型、带参数子列表（必填/可选变量）、匿名嵌入地址
type TreeNode struct {
	Content  ContentUnion       `json:"content" graphql:"content"`
	Label    string             `json:"label" graphql:"label"`
	Children TreeNodeConnection `json:"children" graphql:"children(first:$first:Int!, after:$:String)"`
	AddressEmbed // 匿名嵌入：字段平铺到父级，与 encoding/json 完全兼容
}

// TreeRoot 查询根下的树；仅一层入口
type TreeRoot struct {
	Node TreeNode `json:"node" graphql:"node"`
}

// --- 主测结构体：覆盖参数、参数类型、默认值、联合、Fragment、内联、匿名块、别名 ---

// Profile 用于验证 struct-to-graphql 所有支持的 GraphQL 场景
type Profile struct {
	// 标量字段
	QueryID string `json:"queryId" graphql:"queryId"`

	// 参数：字面量 first:10；必填变量 query:$:String!、id:$id:Int!；可选变量 filter:$filter:String；默认值 page:$page:Int=1
	ProductList ProductConnection `json:"productList" graphql:"productList(first:10, query:$:String!, id:$id:Int!, filter:$filter:String, page:$page:Int=1)"`

	// Fragment 复用：UserBasic 在多处出现，会生成 fragment UserBasic on UserBasic { id name email }
	Author   UserBasic `json:"author" graphql:"author"`
	Reviewer UserBasic `json:"reviewer" graphql:"reviewer"`

	// 联合类型：根据 __typename 展开为 ... on ArticleContent / ... on VideoContent
	Content ContentUnion `json:"content" graphql:"content"`

	VideoContent VideoContent `json:"videoContent" graphql:"videoContent"`

	// 自动生成字段名(不转换大小写)
	AddressEmbedField AddressEmbed

	// 匿名嵌入：ContactEmbed 字段平铺到父级，与 encoding/json 完全兼容
	ContactEmbed

	// 匿名结构体：仅此处使用的形状，带别名
	AnonymousBlock struct {
		DisplayName string `json:"displayName" graphql:"name,alias=displayName"`
	} `json:"anonymousBlock" graphql:"anonymousBlock"`

	// 嵌套树：内含联合类型、带参数列表、匿名嵌入地址，自定义字段名
	Tree TreeRoot `json:"tree1" graphql:"tree1"`

	// Fragment 复用 + 匿名嵌入：同一 UserBasic 以匿名嵌入展开到本层
	UserBasic

	// 嵌入类型示例：匿名嵌入 MetaInfo，其字段 createdAt/updatedAt 平铺到本层
	MetaInfo

	Pointers  []*MetaInfo   `json:"pointers" graphql:"pointers"`
	Pointers2 [][]*MetaInfo `json:"pointers2" graphql:"pointers2"`
}
type QueryFull struct {
	Profile  Profile        `json:"profile" graphql:"profile(author:$author:String!)"`
	Contents []ContentUnion `json:"contents" graphql:"contents(author:$author:String!)"`
}

func TestFull(t *testing.T) {
	exec, err := graphql.Marshal(QueryFull{})
	if err != nil {
		panic(err)
	}
	query, err := exec.Query("GetData")
	if err != nil {
		panic(err)
	}
	fmt.Println(query)
}
