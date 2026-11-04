package test_graphql

import (
	"fmt"
	"testing"

	"github.com/lascyb/struct-to-graphql/test/testutil"
)

// 用 graphql tag 生成的查询，其响应 JSON 的 key（alias 优先）必须能被 encoding/json 写回同一个结构体，
// 否则字段会被静默丢弃（例如 alias=headline 但 json:"title"）。
func TestResponseKeysMatchJSONTags(t *testing.T) {
	t.Parallel()
	types := []any{
		ScalarQuery{}, SliceQuery{}, PointerQuery{}, NestedQuery{},
		AnonymousQuery{}, AnonymousNestedQuery{}, AnonymousSliceQuery{}, AnonymousFragmentQuery{},
		AliasQuery{}, AnonymousAliasQuery{}, EmbedQuery{}, EmbeddedQuery{},
		LiteralQuery{}, EscapeQuery{}, AnonymousVariableQuery{}, VariableQuery{}, DefaultValueQuery{},
		UnionQuery{}, UnionReuseQuery{}, FragmentQuery{}, CombinedQuery{}, MutationStruct{},
		DeterministicQuery{}, CommentAddedSubscription{},
		DirectiveLiteralQuery{}, DirectiveVariableQuery{}, DirectiveCombinedQuery{}, DirectiveFragmentQuery{},
		DirectiveUnionQuery{}, DirectiveSharedVarQuery{}, DirectiveSubscription{}, DirectiveMutation{},
		DirectiveReadmeQuery{},
		TagFallbackQuery{}, ScalarLikeQuery{}, GenericQuery{}, FragmentCollisionQuery{}, FragmentNameClashQuery{},
		EmbedReuseQuery{}, EmbedAfterFragmentQuery{},
	}
	for _, v := range types {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			t.Parallel()
			testutil.AssertResponseKeys(t, v)
		})
	}
}
