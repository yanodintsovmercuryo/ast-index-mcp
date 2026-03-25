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
		require.NotPanics(t, func() { commands.New(nil) })
	})

	t.Run("all tool names are unique", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		seen := make(map[string]struct{})
		for _, d := range r.All() {
			_, exists := seen[d.ToolName]
			require.False(t, exists, "duplicate tool name: %s", d.ToolName)
			seen[d.ToolName] = struct{}{}
		}
	})

	t.Run("all tool names have ast_ prefix", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		for _, d := range r.All() {
			require.True(t, len(d.ToolName) > 4 && d.ToolName[:4] == "ast_",
				"tool name %q does not start with ast_", d.ToolName)
		}
	})

	t.Run("New(nil) registers universal tools only", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		require.Equal(t, 41, len(r.All()))
	})

	t.Run("all commands have non-empty DataType", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		for _, d := range r.All() {
			require.NotEmpty(t, d.DataType, "tool %s has empty DataType", d.ToolName)
		}
	})

	t.Run("all commands have non-empty CLISubcommand", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		for _, d := range r.All() {
			require.NotEmpty(t, d.CLISubcommand, "tool %s has empty CLISubcommand", d.ToolName)
		}
	})
}

func TestRegistry_GroupFiltering(t *testing.T) {
	t.Parallel()

	t.Run("New(nil) returns all tools when no groups tagged", func(t *testing.T) {
		t.Parallel()
		// Before groups are tagged on tools, nil == all tools included.
		r := commands.New(nil)
		require.NotEmpty(t, r.All())
	})

	t.Run("New with unknown group only returns universal tools", func(t *testing.T) {
		t.Parallel()
		rUnknown := commands.New([]string{"unknowngroup"})
		rNil := commands.New(nil)
		require.Equal(t, len(rNil.All()), len(rUnknown.All()))
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	r := commands.New(nil)

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

func TestRegistry_GroupFiltering_WithTags(t *testing.T) {
	t.Parallel()

	t.Run("New(nil) excludes language-specific tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		langSpecific := []string{
			"ast_suspend", "ast_composables", "ast_flows", "ast_previews", "ast_async_funcs",
			"ast_deeplinks", "ast_xml_usages", "ast_resource_usages", "ast_resource_unused",
			"ast_asset_usages", "ast_asset_unused",
			"ast_swiftui", "ast_publishers", "ast_main_actor", "ast_storyboard_usages",
			"ast_perl_exports", "ast_perl_subs", "ast_perl_pod", "ast_perl_tests", "ast_perl_imports",
		}
		for _, name := range langSpecific {
			_, ok := r.Get(name)
			require.False(t, ok, "tool %s should not be in universal set", name)
		}
	})

	t.Run("New(kotlin) includes kotlin tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"kotlin"})
		for _, name := range []string{"ast_suspend", "ast_composables", "ast_flows", "ast_previews", "ast_async_funcs"} {
			_, ok := r.Get(name)
			require.True(t, ok, "tool %s should be included with kotlin group", name)
		}
	})

	t.Run("New(swift) includes swift tools including ast_async_funcs", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"swift"})
		for _, name := range []string{"ast_swiftui", "ast_publishers", "ast_main_actor", "ast_storyboard_usages", "ast_async_funcs"} {
			_, ok := r.Get(name)
			require.True(t, ok, "tool %s should be included with swift group", name)
		}
	})

	t.Run("New(android) includes android tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"android"})
		for _, name := range []string{"ast_deeplinks", "ast_xml_usages", "ast_resource_usages", "ast_resource_unused", "ast_asset_usages", "ast_asset_unused"} {
			_, ok := r.Get(name)
			require.True(t, ok, "tool %s should be included with android group", name)
		}
	})

	t.Run("New(perl) includes perl tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"perl"})
		for _, name := range []string{"ast_perl_exports", "ast_perl_subs", "ast_perl_pod", "ast_perl_tests", "ast_perl_imports"} {
			_, ok := r.Get(name)
			require.True(t, ok, "tool %s should be included with perl group", name)
		}
	})

	t.Run("New(nil) universal tool count is 41", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		require.Equal(t, 41, len(r.All()))
	})
}
