# Design: Tool Groups via ENV, Deduplication, Description Trimming

**Date:** 2026-03-26
**Status:** Approved

## Problem

The MCP server exposes 61 tools unconditionally. Each tool definition (name + description + parameter schemas) is sent to the client on every connection, consuming ~12k tokens in the context window — even when most language-specific tools are irrelevant to the project.

## Goals

1. Language-specific tools opt-in via `AST_INDEX_TOOLS` env var.
2. Default: only universal tools loaded (~38 tools instead of 61).
3. Merge duplicate tool pairs that share a CLI subcommand.
4. Shorten tool descriptions to reduce per-tool token cost.

## Architecture

### Approach: Filter in Registry (chosen)

`Registry.New()` receives `enabledGroups []string` from config and filters at construction time. `main.go` and `ToolHandler` are unaware of groups.

```
Config.Load() → Config.Tools []string
     ↓
commands.New(enabledGroups)
     ↓
Registry filters: include if def.Groups == nil OR def.Groups ∩ enabledGroups ≠ ∅
     ↓
main.go iterates registry.All() (unchanged loop)
```

### CommandDef change

Add one field:

```go
// Groups lists the opt-in groups that activate this tool.
// Empty means universal — always included.
Groups []string
```

### Registry.New signature

```go
func New(enabledGroups []string) *Registry
```

Filtering rule:
- `len(def.Groups) == 0` → always include
- otherwise → include if any element of `def.Groups` is in `enabledGroups`

## Tool Groups

| Group | Tools |
|-------|-------|
| *(universal)* | ast_search, ast_symbol, ast_class, ast_file, ast_usages, ast_refs, ast_callers, ast_call_tree, ast_implementations, ast_hierarchy, ast_outline, ast_imports, ast_todo, ast_provides, ast_deprecated, ast_suppress, ast_inject, ast_annotations, ast_agrep, ast_extensions, ast_module, ast_deps, ast_dependents, ast_unused_deps, ast_api, ast_map, ast_conventions, ast_changed, ast_init, ast_rebuild, ast_update, ast_watch, ast_stats, ast_version, ast_unused_symbols, ast_add_root, ast_list_roots, ast_remove_root, ast_query, ast_db_path, ast_schema |
| `kotlin` | ast_suspend, ast_composables, ast_flows, ast_previews, ast_async_funcs |
| `android` | ast_deeplinks, ast_xml_usages, ast_resource (merged), ast_asset (merged) |
| `swift` | ast_swiftui, ast_publishers, ast_main_actor, ast_storyboard_usages, ast_async_funcs |
| `perl` | ast_perl_exports, ast_perl_subs, ast_perl_pod, ast_perl_tests, ast_perl_imports |

`ast_async_funcs` has `Groups: []string{"kotlin", "swift"}` — enabled by either.

## Deduplication

### ast_resource (replaces ast_resource_usages + ast_resource_unused)

CLI subcommand: `resource-usages`

| unused | required args | CLI flags added |
|--------|---------------|-----------------|
| false (default) | `resource` string | positional `resource` value |
| true | `module` string | `--unused --module <module>` |

Description: `"Resource usages (R.* / string / drawable); set unused=true to list unused resources in a module"`

### ast_asset (replaces ast_asset_usages + ast_asset_unused)

CLI subcommand: `asset-usages`

| unused | required args | CLI flags added |
|--------|---------------|-----------------|
| false (default) | `asset` string (optional) | positional `asset` value if provided |
| true | `module` string | `--unused --module <module>` |

Description: `"Asset usages; set unused=true to list unused assets in a module"`

The existing special-case switch in `buildArgv` for `ast_resource_unused` / `ast_asset_unused` is replaced by handling the `unused` bool on the merged tools.

## Config

### New field

```go
// Tools is the list of opt-in tool groups to enable. Env: AST_INDEX_TOOLS.
// Empty means only universal tools are loaded.
// Valid values: kotlin, android, swift, perl (comma-separated).
Tools []string
```

### Parsing

```go
if v := os.Getenv("AST_INDEX_TOOLS"); v != "" {
    for _, g := range strings.Split(v, ",") {
        g = strings.TrimSpace(g)
        if g != "" {
            cfg.Tools = append(cfg.Tools, g)
        }
    }
}
```

No validation of group names — unknown groups are silently ignored (no tool matches them).

## Description Trimming

Remove filler prefixes: "Find all", "Show all", "List all", "Get", "Build".
Each description trimmed to ≤ 10 words where possible.

Examples:
- `"Full-text and semantic search across the indexed codebase"` → `"Full-text and semantic search across the index"`
- `"Find all usages of a symbol"` → `"Usages of a symbol"`
- `"Show class/type inheritance hierarchy"` → `"Class/type inheritance hierarchy"`
- `"Find Kotlin suspend / async functions"` → `"Kotlin suspend / async functions"`

## Files Changed

| File | Change |
|------|--------|
| `internal/commands/registry.go` | Add `Groups []string` to `CommandDef`; change `New()` signature; add group tags to all language-specific tools; merge ast_resource_unused→ast_resource, ast_asset_unused→ast_asset; trim descriptions |
| `internal/mcp/tools.go` | Update `buildArgv` to handle merged tools `ast_resource` / `ast_asset` with `unused` bool; remove old switch cases for `ast_resource_unused` / `ast_asset_unused` |
| `internal/config/config.go` | Add `Tools []string` field; parse `AST_INDEX_TOOLS` |
| `main.go` | Pass `cfg.Tools` to `commands.New()` |
| `internal/commands/registry_test.go` | Update tests for new `New(groups)` signature |
| `internal/mcp/tools_test.go` | Update `setUp` helper to pass groups; add test for merged ast_resource / ast_asset |
| `internal/config/config_test.go` | Add test for `AST_INDEX_TOOLS` parsing |

## Testing

- `Registry.New(nil)` → only universal tools, no language-specific ones.
- `Registry.New([]string{"kotlin"})` → universal + kotlin tools, no swift/android/perl.
- `Registry.New([]string{"swift"})` → includes `ast_async_funcs` (shared group).
- `Registry.New([]string{"kotlin"})` → includes `ast_async_funcs` (shared group).
- `ast_resource` with `unused=true` → argv contains `--unused --module <module>`.
- `ast_resource` with `unused=false` → argv contains resource as positional.
- `ast_asset` same pattern.
- Config: `AST_INDEX_TOOLS=kotlin,android` → `cfg.Tools == ["kotlin","android"]`.
- Config: `AST_INDEX_TOOLS` unset → `cfg.Tools == nil`.
