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
		require.Equal(t, 21, len(r.All()))
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

	t.Run("New(nil) returns non-empty result", func(t *testing.T) {
		t.Parallel()
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
		rSQL := commands.New([]string{"sql"})
		d, ok := rSQL.Get("ast_query")
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
		rExt := commands.New([]string{"extended"})
		d, ok := rExt.Get("ast_version")
		require.True(t, ok)
		for _, arg := range d.Args {
			require.False(t, arg.Required, "ast_version should have no required args")
		}
	})
}

func TestRegistry_GroupFiltering_WithTags(t *testing.T) {
	t.Parallel()

	t.Run("New(nil) excludes opt-in group tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New(nil)
		optIn := []string{
			// kotlin
			"ast_suspend", "ast_composables", "ast_flows", "ast_previews", "ast_async_funcs",
			// android
			"ast_deeplinks", "ast_xml_usages", "ast_resource", "ast_asset",
			// swift
			"ast_swiftui", "ast_publishers", "ast_main_actor", "ast_storyboard_usages",
			// perl
			"ast_perl_exports", "ast_perl_subs", "ast_perl_pod", "ast_perl_tests", "ast_perl_imports",
			// extended
			"ast_call_tree", "ast_hierarchy", "ast_provides", "ast_deprecated", "ast_suppress",
			"ast_inject", "ast_agrep", "ast_unused_deps", "ast_api", "ast_conventions",
			"ast_watch", "ast_stats", "ast_version", "ast_unused_symbols",
			"ast_add_root", "ast_list_roots", "ast_remove_root",
			// sql
			"ast_query", "ast_db_path", "ast_schema",
		}
		for _, name := range optIn {
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
		for _, name := range []string{"ast_deeplinks", "ast_xml_usages", "ast_resource", "ast_asset"} {
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

	t.Run("ast_resource replaces ast_resource_usages and ast_resource_unused", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"android"})
		_, hasOldUsages := r.Get("ast_resource_usages")
		_, hasOldUnused := r.Get("ast_resource_unused")
		_, hasMerged := r.Get("ast_resource")
		require.False(t, hasOldUsages)
		require.False(t, hasOldUnused)
		require.True(t, hasMerged)
	})

	t.Run("ast_asset replaces ast_asset_usages and ast_asset_unused", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"android"})
		_, hasOldUsages := r.Get("ast_asset_usages")
		_, hasOldUnused := r.Get("ast_asset_unused")
		_, hasMerged := r.Get("ast_asset")
		require.False(t, hasOldUsages)
		require.False(t, hasOldUnused)
		require.True(t, hasMerged)
	})

	t.Run("New(android) count after merge is 25", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"android"})
		// 21 universal + 4 android (ast_deeplinks, ast_xml_usages, ast_resource, ast_asset)
		require.Equal(t, 25, len(r.All()))
	})

	t.Run("New(extended) includes extended tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"extended"})
		for _, name := range []string{
			"ast_call_tree", "ast_hierarchy", "ast_provides", "ast_deprecated", "ast_suppress",
			"ast_inject", "ast_agrep", "ast_unused_deps", "ast_api", "ast_conventions",
			"ast_watch", "ast_stats", "ast_version", "ast_unused_symbols",
			"ast_add_root", "ast_list_roots", "ast_remove_root",
		} {
			_, ok := r.Get(name)
			require.True(t, ok, "tool %s should be included with extended group", name)
		}
	})

	t.Run("New(sql) includes sql tools", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"sql"})
		for _, name := range []string{"ast_query", "ast_db_path", "ast_schema"} {
			_, ok := r.Get(name)
			require.True(t, ok, "tool %s should be included with sql group", name)
		}
	})

	t.Run("New(sql) count is 24", func(t *testing.T) {
		t.Parallel()
		r := commands.New([]string{"sql"})
		// 21 universal + 3 sql
		require.Equal(t, 24, len(r.All()))
	})

}
