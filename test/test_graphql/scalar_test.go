package test_graphql

import (
	"encoding/json"
	"strings"
	"testing"

	graphql "github.com/lascyb/struct-to-graphql"
)

// 测试标量类型支持
type ScalarQuery struct {
	StringField  string `json:"stringField" graphql:"stringField"`
	IntField     int    `json:"intField" graphql:"intField"`
	BoolField    bool   `json:"boolField" graphql:"boolField"`
	FloatField   float64 `json:"floatField" graphql:"floatField"`
}

func TestScalarTypes(t *testing.T) {
	exec, err := graphql.Marshal(ScalarQuery{})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	query, err := exec.Query("ScalarTest")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	
	t.Logf("Generated Query:\n%s", query)
	
	// 验证所有标量字段都存在
	if !strings.Contains(query, "stringField") {
		t.Error("Missing stringField")
	}
	if !strings.Contains(query, "intField") {
		t.Error("Missing intField")
	}
	if !strings.Contains(query, "boolField") {
		t.Error("Missing boolField")
	}
	if !strings.Contains(query, "floatField") {
		t.Error("Missing floatField")
	}
}

// ========== 标量类型：encoding/json 往返测试 ==========

func TestScalarUnmarshal(t *testing.T) {
	resp := `{"stringField":"hello","intField":42,"boolField":true,"floatField":3.14}`
	var got ScalarQuery
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	want := ScalarQuery{StringField: "hello", IntField: 42, BoolField: true, FloatField: 3.14}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
