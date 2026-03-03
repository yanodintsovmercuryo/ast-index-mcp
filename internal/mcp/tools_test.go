package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/commands"
	internalmcp "github.com/yanodintsovmercuryo/ast-index-mcp/internal/mcp"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/normalize"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/runner"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/security"
)

func setUp(t *testing.T, root string) *internalmcp.ToolHandler {
	t.Helper()
	guard, err := security.NewPathGuard(root)
	require.NoError(t, err)

	return internalmcp.NewToolHandler(
		"echo", // use echo as fake ast-index binary
		root,
		5,
		commands.New(),
		guard,
		runner.New(),
		normalize.New(),
	)
}

func TestToolHandler_Handle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	handler := setUp(t, root)

	t.Run("unknown tool returns error response", func(t *testing.T) {
		t.Parallel()
		resp := handler.Handle(context.Background(), "ast_nonexistent", nil)
		require.False(t, resp.Ok)
		require.NotEmpty(t, resp.Diagnostics)
	})

	t.Run("response envelope has all required fields", func(t *testing.T) {
		t.Parallel()
		resp := handler.Handle(context.Background(), "ast_version", map[string]any{})
		require.Equal(t, "ast_version", resp.Tool)
		require.Equal(t, "version", resp.Command)
		require.NotNil(t, resp.Argv)
		require.NotNil(t, resp.Data)
		require.GreaterOrEqual(t, resp.DurationMs, int64(0))
	})

	t.Run("path outside root rejected", func(t *testing.T) {
		t.Parallel()
		resp := handler.Handle(context.Background(), "ast_outline", map[string]any{
			"file": "/etc/passwd",
		})
		require.False(t, resp.Ok)
		require.NotEmpty(t, resp.Diagnostics)
	})

	t.Run("cwd outside root rejected", func(t *testing.T) {
		t.Parallel()
		resp := handler.Handle(context.Background(), "ast_search", map[string]any{
			"query": "Foo",
			"cwd":   "/etc",
		})
		require.False(t, resp.Ok)
		require.NotEmpty(t, resp.Diagnostics)
	})
}

func TestToolHandler_Handle_SQLDenyList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	handler := setUp(t, root)

	forbidden := []string{
		"INSERT INTO symbols VALUES (1,'x')",
		"UPDATE symbols SET name='y'",
		"DELETE FROM symbols",
		"DROP TABLE symbols",
		"ALTER TABLE symbols ADD COLUMN foo TEXT",
		"ATTACH DATABASE '/tmp/evil' AS evil",
		"PRAGMA writable_schema = ON",
	}

	for _, sql := range forbidden {
		sql := sql
		t.Run(sql, func(t *testing.T) {
			t.Parallel()
			resp := handler.Handle(context.Background(), "ast_query", map[string]any{
				"sql": sql,
			})
			require.False(t, resp.Ok)
			require.NotEmpty(t, resp.Diagnostics)
		})
	}

	t.Run("SELECT is allowed", func(t *testing.T) {
		t.Parallel()
		// echo will just output the args, so exit code will be 0.
		resp := handler.Handle(context.Background(), "ast_query", map[string]any{
			"sql": "SELECT * FROM symbols LIMIT 10",
		})
		// Command may fail (echo is not ast-index) but no deny-list error.
		require.Equal(t, "ast_query", resp.Tool)
		noDenyDiag := true
		for _, d := range resp.Diagnostics {
			if d.Code == "ERROR" && len(d.Message) > 0 && d.Message == "sql deny-list: only SELECT statements are permitted" {
				noDenyDiag = false
			}
		}
		require.True(t, noDenyDiag)
	})
}

func TestToolHandler_ResponseEnvelope_Contract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	handler := setUp(t, root)

	resp := handler.Handle(context.Background(), "ast_stats", map[string]any{})

	// Serialize and verify all envelope fields are present.
	b, err := json.Marshal(resp)
	require.NoError(t, err)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(b, &envelope))

	for _, field := range []string{"ok", "tool", "command", "argv", "cwd", "exit_code", "duration_ms", "timed_out", "data", "stderr", "diagnostics"} {
		_, exists := envelope[field]
		require.True(t, exists, "envelope missing field: %s", field)
	}
}
