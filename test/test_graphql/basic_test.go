package test_graphql

import "testing"

// ========== 标量、切片、指针、嵌套结构体 ==========

type ScalarQuery struct {
	StringField string  `json:"stringField" graphql:"stringField"`
	IntField    int     `json:"intField" graphql:"intField"`
	BoolField   bool    `json:"boolField" graphql:"boolField"`
	FloatField  float64 `json:"floatField" graphql:"floatField"`
}

type Tag struct {
	Name string `json:"name" graphql:"name"`
}

type SliceQuery struct {
	Tags    []Tag    `json:"tags" graphql:"tags"`
	Numbers []int    `json:"numbers" graphql:"numbers"`
	Nested  [][]Tag  `json:"nested" graphql:"nested"`
	Fixed   [2]*Tag  `json:"fixed" graphql:"fixed"`
	Labels  []string `json:"labels" graphql:"labels"`
}

type ProfileInfo struct {
	Bio    string `json:"bio" graphql:"bio"`
	Avatar string `json:"avatar" graphql:"avatar"`
}

type PointerQuery struct {
	ID      string        `json:"id" graphql:"id"`
	Profile *ProfileInfo  `json:"profile" graphql:"profile"`
	Double  **ProfileInfo `json:"double" graphql:"double"`
	Tags    []*struct {
		Name string `json:"name" graphql:"name"`
	} `json:"tags" graphql:"tags"`
}

type Address struct {
	City   string `json:"city" graphql:"city"`
	Street string `json:"street" graphql:"street"`
}

type Company struct {
	Name    string  `json:"name" graphql:"name"`
	Address Address `json:"address" graphql:"address"`
}

type NestedQuery struct {
	User    string  `json:"user" graphql:"user"`
	Company Company `json:"company" graphql:"company"`
}

func TestBasicOutput(t *testing.T) {
	t.Parallel()
	runOutputCases(t, []outputCase{
		{name: "Scalar", v: ScalarQuery{}, opName: "ScalarTest", want: `query ScalarTest {
  stringField
  intField
  boolField
  floatField
}`},
		// 切片、二维切片、数组均解开到元素类型；Tag 被复用 3 次因此生成 Fragment
		{name: "Slice", v: SliceQuery{}, opName: "SliceTest", want: `fragment Tag on Tag{
  name
}
query SliceTest {
  tags{ ...Tag }
  numbers
  nested{ ...Tag }
  fixed{ ...Tag }
  labels
}`},
		// 指针（含多级指针）与指针切片
		{name: "Pointer", v: &PointerQuery{}, opName: "PointerTest", want: `fragment ProfileInfo on ProfileInfo{
  bio
  avatar
}
query PointerTest {
  id
  profile{ ...ProfileInfo }
  double{ ...ProfileInfo }
  tags{
    name
  }
}`},
		{name: "Nested", v: NestedQuery{}, opName: "NestedTest", want: `query NestedTest {
  user
  company{
    name
    address{
      city
      street
    }
  }
}`},
	})
}
