package normalize_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/normalize"
)

func TestNormalizer_Normalize(t *testing.T) {
	t.Parallel()

	n := normalize.New()

	t.Run("valid JSON stdout is passed through", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"items":[{"name":"Foo","kind":"class"}]}`)
		data, diags := n.Normalize("search_hits", stdout)
		require.Empty(t, diags)
		require.Equal(t, "search_hits", data.Type)
		require.True(t, json.Valid(data.Payload))
	})

	t.Run("empty stdout returns empty payload", func(t *testing.T) {
		t.Parallel()
		data, diags := n.Normalize("symbols", nil)
		require.Empty(t, diags)
		require.Equal(t, "symbols", data.Type)
		require.Equal(t, json.RawMessage(`{}`), data.Payload)
	})

	t.Run("invalid JSON triggers fallback diagnostic", func(t *testing.T) {
		t.Parallel()
		stdout := []byte("UserRepository   src/user.go:42\nOrderService   src/order.go:10\n")
		data, diags := n.Normalize("symbols", stdout)
		require.Len(t, diags, 1)
		require.Equal(t, normalize.DiagnosticCodeNormalizationFallback, diags[0].Code)
		require.Equal(t, "symbols", data.Type)

		// Payload must be valid JSON containing raw field.
		var payload map[string]any
		require.NoError(t, json.Unmarshal(data.Payload, &payload))
		require.Contains(t, payload, "raw")
	})

	t.Run("search golden", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"type":"search_hits","items":[{"name":"UserRepo","score":0.9}]}`)
		data, diags := n.Normalize("search_hits", stdout)
		require.Empty(t, diags)
		require.Equal(t, "search_hits", data.Type)
	})

	t.Run("stats golden", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"files":100,"symbols":2000,"db_size_bytes":4096}`)
		data, diags := n.Normalize("index_stats", stdout)
		require.Empty(t, diags)
		require.Equal(t, "index_stats", data.Type)
		require.True(t, json.Valid(data.Payload))
	})

	t.Run("query golden", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"columns":["name","file"],"rows":[["Foo","src/foo.go"]],"row_count":1}`)
		data, diags := n.Normalize("sql_result", stdout)
		require.Empty(t, diags)
		require.Equal(t, "sql_result", data.Type)
	})
}
