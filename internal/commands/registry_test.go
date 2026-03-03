package commands_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/commands"
)

func TestRegistry_New(t *testing.T) {
	t.Parallel()

	t.Run("creates without panic", func(t *testing.T) {
		t.Parallel()
		require.NotPanics(t, func() { commands.New() })
	})

	t.Run("all tool names are unique", func(t *testing.T) {
		t.Parallel()
		r := commands.New()
		seen := make(map[string]struct{})
		for _, d := range r.All() {
			_, exists := seen[d.ToolName]
			require.False(t, exists, "duplicate tool name: %s", d.ToolName)
			seen[d.ToolName] = struct{}{}
		}
	})

	t.Run("all tool names have ast_ prefix", func(t *testing.T) {
		t.Parallel()
		r := commands.New()
		for _, d := range r.All() {
			require.True(t, len(d.ToolName) > 4 && d.ToolName[:4] == "ast_",
				"tool name %q does not start with ast_", d.ToolName)
		}
	})

	t.Run("at least 46 commands registered", func(t *testing.T) {
		t.Parallel()
		r := commands.New()
		require.GreaterOrEqual(t, len(r.All()), 46)
	})

	t.Run("all commands have non-empty DataType", func(t *testing.T) {
		t.Parallel()
		r := commands.New()
		for _, d := range r.All() {
			require.NotEmpty(t, d.DataType, "tool %s has empty DataType", d.ToolName)
		}
	})

	t.Run("all commands have non-empty CLISubcommand", func(t *testing.T) {
		t.Parallel()
		r := commands.New()
		for _, d := range r.All() {
			require.NotEmpty(t, d.CLISubcommand, "tool %s has empty CLISubcommand", d.ToolName)
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	r := commands.New()

	t.Run("returns known command", func(t *testing.T) {
		t.Parallel()
		d, ok := r.Get("ast_search")
		require.True(t, ok)
		require.Equal(t, "ast_search", d.ToolName)
		require.Equal(t, "search", d.CLISubcommand)
		require.Equal(t, "search_hits", d.DataType)
	})

	t.Run("returns false for unknown command", func(t *testing.T) {
		t.Parallel()
		_, ok := r.Get("ast_nonexistent")
		require.False(t, ok)
	})

	t.Run("ast_query has required sql arg", func(t *testing.T) {
		t.Parallel()
		d, ok := r.Get("ast_query")
		require.True(t, ok)
		found := false
		for _, arg := range d.Args {
			if arg.Name == "sql" && arg.Required {
				found = true
				break
			}
		}
		require.True(t, found, "ast_query must have required 'sql' argument")
	})

	t.Run("ast_version has no required args", func(t *testing.T) {
		t.Parallel()
		d, ok := r.Get("ast_version")
		require.True(t, ok)
		for _, arg := range d.Args {
			require.False(t, arg.Required, "ast_version should have no required args")
		}
	})
}
