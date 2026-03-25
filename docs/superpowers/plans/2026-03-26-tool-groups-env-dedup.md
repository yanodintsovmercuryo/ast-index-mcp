# Tool Groups via ENV, Deduplication, Description Trimming — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cut MCP context cost from ~12k tokens to ~5k by making language-specific tools opt-in via `AST_INDEX_TOOLS` env var, merging duplicate tools, and trimming verbose descriptions.

**Architecture:** Add `Groups []string` to `CommandDef`; `Registry.New(enabledGroups []string)` filters out tools whose groups don't intersect with enabled groups (empty groups = universal, always included). Config parses `AST_INDEX_TOOLS` comma-separated string into `[]string` and passes it to `commands.New()`. Duplicate tool pairs (`ast_resource_*`, `ast_asset_*`) are merged into single tools with an `unused bool` arg.

**Tech Stack:** Go, `github.com/stretchr/testify/require`, existing internal packages.

---

## File Map

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `Tools []string`, parse `AST_INDEX_TOOLS` |
| `internal/config/config_test.go` | Add tests for `AST_INDEX_TOOLS` parsing |
| `internal/commands/registry.go` | Add `Groups []string` to `CommandDef`; change `New()` → `New(enabledGroups []string)`; filtering logic; group tags on language-specific tools; merge ast_resource/ast_asset; trim descriptions |
| `internal/commands/registry_test.go` | Update `New()` → `New(nil)`; add group-filtering tests; update count assertion |
| `internal/mcp/tools.go` | Update `buildArgv` switch: handle `ast_resource` / `ast_asset` with `unused` bool |
| `internal/mcp/tools_test.go` | Update `setUp` → `commands.New(nil)`; add tests for merged tools |
| `main.go` | Pass `cfg.Tools` to `commands.New()` |

---

## Task 1: Config — parse AST_INDEX_TOOLS

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/config/config_test.go` inside `TestLoad`:

```go
t.Run("AST_INDEX_TOOLS comma-separated", func(t *testing.T) {
    t.Setenv("AST_INDEX_TOOLS", "kotlin,android")
    t.Setenv("AST_INDEX_BIN", "")
    t.Setenv("AST_INDEX_CWD", "")
    t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
    t.Setenv("AST_INDEX_LOG_LEVEL", "")

    cfg, err := config.Load()
    require.NoError(t, err)
    require.Equal(t, []string{"kotlin", "android"}, cfg.Tools)
})

t.Run("AST_INDEX_TOOLS not set returns nil", func(t *testing.T) {
    t.Setenv("AST_INDEX_TOOLS", "")
    t.Setenv("AST_INDEX_BIN", "")
    t.Setenv("AST_INDEX_CWD", "")
    t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
    t.Setenv("AST_INDEX_LOG_LEVEL", "")

    cfg, err := config.Load()
    require.NoError(t, err)
    require.Nil(t, cfg.Tools)
})

t.Run("AST_INDEX_TOOLS trims spaces", func(t *testing.T) {
    t.Setenv("AST_INDEX_TOOLS", " kotlin , android ")
    t.Setenv("AST_INDEX_BIN", "")
    t.Setenv("AST_INDEX_CWD", "")
    t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
    t.Setenv("AST_INDEX_LOG_LEVEL", "")

    cfg, err := config.Load()
    require.NoError(t, err)
    require.Equal(t, []string{"kotlin", "android"}, cfg.Tools)
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/... -run TestLoad -v
```

Expected: FAIL — `cfg.Tools` field does not exist yet.

- [ ] **Step 3: Add Tools field and parsing to config.go**

Add `"strings"` to imports. Add `Tools []string` field to `Config`. Add parsing block after the `logLevel` block:

```go
// Tools is the list of opt-in tool groups. Env: AST_INDEX_TOOLS.
Tools []string
```

Parsing block (add after `logLevel` assignment):

```go
var tools []string
if v := os.Getenv("AST_INDEX_TOOLS"); v != "" {
    for _, g := range strings.Split(v, ",") {
        g = strings.TrimSpace(g)
        if g != "" {
            tools = append(tools, g)
        }
    }
}
```

Set in return:

```go
return &Config{
    Bin:        bin,
    CWD:        cwd,
    TimeoutSec: timeoutSec,
    LogLevel:   logLevel,
    Tools:      tools,
}, nil
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add Tools field, parse AST_INDEX_TOOLS env"
```

---

## Task 2: CommandDef.Groups field + Registry.New(enabledGroups) filtering

**Files:**
- Modify: `internal/commands/registry.go`
- Modify: `internal/commands/registry_test.go`
- Modify: `internal/mcp/tools_test.go`

- [ ] **Step 1: Write failing test for group filtering**

Add to `internal/commands/registry_test.go`:

```go
func TestRegistry_GroupFiltering(t *testing.T) {
    t.Parallel()

    t.Run("New(nil) returns all tools when no groups tagged", func(t *testing.T) {
        t.Parallel()
        // Before groups are tagged on tools, nil == all tools included.
        r := commands.New(nil)
        require.NotEmpty(t, r.All())
    })

    t.Run("New with unknown group returns same as nil before groups are tagged", func(t *testing.T) {
        t.Parallel()
        r := commands.New([]string{"unknowngroup"})
        // All tools have empty Groups, so all are universal → all included.
        require.NotEmpty(t, r.All())
    })
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/commands/... -run TestRegistry_GroupFiltering -v
```

Expected: FAIL — `commands.New` does not accept arguments.

- [ ] **Step 3: Add Groups to CommandDef and update New()**

In `internal/commands/registry.go`, add `Groups` to `CommandDef`:

```go
// Groups lists opt-in group names that activate this tool (e.g. "kotlin", "swift").
// Empty means universal — always included regardless of enabled groups.
Groups []string
```

Change `New()` signature and add filtering:

```go
// New builds and validates the command registry.
// enabledGroups is the list of opt-in group names from config (e.g. ["kotlin", "android"]).
// Tools with no Groups are always included. Tools with Groups are included only if at
// least one of their groups is in enabledGroups.
// Panics if any tool name is duplicated.
func New(enabledGroups []string) *Registry {
    defs := allCommands()

    m := make(map[string]CommandDef, len(defs))
    for _, d := range defs {
        if !isEnabled(d, enabledGroups) {
            continue
        }
        if _, exists := m[d.ToolName]; exists {
            panic(fmt.Sprintf("commands: duplicate tool name %q", d.ToolName))
        }
        m[d.ToolName] = d
    }
    return &Registry{commands: m}
}

// isEnabled reports whether a command should be included given the enabled groups.
func isEnabled(def CommandDef, enabledGroups []string) bool {
    if len(def.Groups) == 0 {
        return true
    }
    for _, eg := range enabledGroups {
        for _, dg := range def.Groups {
            if eg == dg {
                return true
            }
        }
    }
    return false
}
```

- [ ] **Step 4: Update all existing callers of New() in tests**

In `internal/commands/registry_test.go`, replace every `commands.New()` with `commands.New(nil)`.

Lines to update (all occurrences in TestRegistry_New and TestRegistry_Get):
```go
// before:
commands.New()
r := commands.New()
// after:
commands.New(nil)
r := commands.New(nil)
```

In `internal/mcp/tools_test.go`, update both `setUp` and `setUpWithBin`:

```go
func setUp(t *testing.T, root string) *internalmcp.ToolHandler {
    t.Helper()
    guard, err := security.NewPathGuard(root)
    require.NoError(t, err)

    return internalmcp.NewToolHandler(
        "echo",
        root,
        5,
        commands.New(nil),
        guard,
        runner.New(),
        normalize.New(),
    )
}

func setUpWithBin(t *testing.T, root, bin string) *internalmcp.ToolHandler {
    t.Helper()
    guard, err := security.NewPathGuard(root)
    require.NoError(t, err)
    return internalmcp.NewToolHandler(bin, root, 5, commands.New(nil), guard, runner.New(), normalize.New())
}
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: all PASS. (At this step, no tools have Groups set, so nil still returns all tools.)

- [ ] **Step 6: Commit**

```bash
git add internal/commands/registry.go internal/commands/registry_test.go internal/mcp/tools_test.go
git commit -m "feat(commands): add Groups field and enabledGroups filtering to Registry.New"
```

---

## Task 3: Tag language-specific tools with Groups

**Files:**
- Modify: `internal/commands/registry.go` (allCommands only)
- Modify: `internal/commands/registry_test.go`

- [ ] **Step 1: Write failing tests for group-based filtering**

Add to `internal/commands/registry_test.go` in `TestRegistry_GroupFiltering`:

```go
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

t.Run("New(nil) universal tool count", func(t *testing.T) {
    t.Parallel()
    r := commands.New(nil)
    // 41 universal tools (61 total - 20 language-specific)
    require.Equal(t, 41, len(r.All()))
})
```

Also update the existing `"at least 46 commands registered"` test to reflect the new default:

```go
t.Run("New(nil) has expected universal tool count", func(t *testing.T) {
    t.Parallel()
    r := commands.New(nil)
    require.Equal(t, 41, len(r.All()))
})
```

(Delete the old `"at least 46 commands registered"` subtest.)

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/commands/... -run TestRegistry_GroupFiltering -v
```

Expected: FAIL — language-specific tools are not tagged yet.

- [ ] **Step 3: Add Groups tags to language-specific tools in allCommands()**

In `internal/commands/registry.go`, add `Groups: []string{...}` to the following entries:

**Kotlin group** (4 tools):
```go
// ast_suspend
Groups: []string{"kotlin"},

// ast_composables
Groups: []string{"kotlin"},

// ast_flows
Groups: []string{"kotlin"},

// ast_previews
Groups: []string{"kotlin"},
```

**Kotlin + Swift shared** (1 tool):
```go
// ast_async_funcs
Groups: []string{"kotlin", "swift"},
```

**Android group** (6 tools — ast_resource_usages, ast_resource_unused, ast_asset_usages, ast_asset_unused, ast_xml_usages, ast_deeplinks):
```go
// ast_xml_usages
Groups: []string{"android"},

// ast_resource_usages
Groups: []string{"android"},

// ast_resource_unused
Groups: []string{"android"},

// ast_storyboard_usages — NOTE: this is swift, not android
// ast_asset_usages
Groups: []string{"android"},

// ast_asset_unused
Groups: []string{"android"},

// ast_deeplinks
Groups: []string{"android"},
```

**Swift group** (4 tools — ast_swiftui, ast_publishers, ast_main_actor, ast_storyboard_usages):
```go
// ast_swiftui
Groups: []string{"swift"},

// ast_publishers
Groups: []string{"swift"},

// ast_main_actor
Groups: []string{"swift"},

// ast_storyboard_usages
Groups: []string{"swift"},
```

**Perl group** (5 tools):
```go
// ast_perl_exports
Groups: []string{"perl"},

// ast_perl_subs
Groups: []string{"perl"},

// ast_perl_pod
Groups: []string{"perl"},

// ast_perl_tests
Groups: []string{"perl"},

// ast_perl_imports
Groups: []string{"perl"},
```

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/commands/registry.go internal/commands/registry_test.go
git commit -m "feat(commands): tag language-specific tools with opt-in groups"
```

---

## Task 4: Merge ast_resource + ast_asset, update buildArgv

**Files:**
- Modify: `internal/commands/registry.go` (allCommands)
- Modify: `internal/mcp/tools.go` (buildArgv)
- Modify: `internal/mcp/tools_test.go`
- Modify: `internal/commands/registry_test.go`

- [ ] **Step 1: Write failing tests for merged tools**

Add to `internal/mcp/tools_test.go`:

```go
func TestToolHandler_MergedTools(t *testing.T) {
    t.Parallel()

    root := t.TempDir()
    guard, err := security.NewPathGuard(root)
    require.NoError(t, err)
    handler := internalmcp.NewToolHandler("echo", root, 5,
        commands.New([]string{"android"}), guard, runner.New(), normalize.New())

    t.Run("ast_resource unused=false passes resource as positional", func(t *testing.T) {
        t.Parallel()
        resp := handler.Handle(context.Background(), "ast_resource", map[string]any{
            "resource": "R.string.title",
        })
        require.Equal(t, "ast_resource", resp.Tool)
        require.Contains(t, resp.Argv, "R.string.title")
        require.NotContains(t, resp.Argv, "--unused")
    })

    t.Run("ast_resource unused=true passes --unused --module", func(t *testing.T) {
        t.Parallel()
        resp := handler.Handle(context.Background(), "ast_resource", map[string]any{
            "unused": true,
            "module": ":feature-home",
        })
        require.Equal(t, "ast_resource", resp.Tool)
        require.Contains(t, resp.Argv, "--unused")
        require.Contains(t, resp.Argv, "--module")
        require.Contains(t, resp.Argv, ":feature-home")
    })

    t.Run("ast_resource unused=true without module returns error", func(t *testing.T) {
        t.Parallel()
        resp := handler.Handle(context.Background(), "ast_resource", map[string]any{
            "unused": true,
        })
        require.False(t, resp.Ok)
        require.NotEmpty(t, resp.Diagnostics)
    })

    t.Run("ast_asset unused=false with no asset name passes nothing extra", func(t *testing.T) {
        t.Parallel()
        resp := handler.Handle(context.Background(), "ast_asset", map[string]any{})
        require.Equal(t, "ast_asset", resp.Tool)
        require.NotContains(t, resp.Argv, "--unused")
    })

    t.Run("ast_asset unused=true passes --unused --module", func(t *testing.T) {
        t.Parallel()
        resp := handler.Handle(context.Background(), "ast_asset", map[string]any{
            "unused": true,
            "module": ":feature-home",
        })
        require.Contains(t, resp.Argv, "--unused")
        require.Contains(t, resp.Argv, "--module")
        require.Contains(t, resp.Argv, ":feature-home")
    })
}
```

Also add to `TestRegistry_GroupFiltering` in `registry_test.go`:

```go
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
```

Update the `"New(nil) universal tool count"` test: with merges, android loses 2 tools (ast_resource_usages, ast_resource_unused → ast_resource; ast_asset_usages, ast_asset_unused → ast_asset) but universal count stays 41 (merged tools are in android group, not universal). Update `New([]string{"android"})` expected count from 6 android tools to 4:

```go
t.Run("New(android) count after merge", func(t *testing.T) {
    t.Parallel()
    r := commands.New([]string{"android"})
    // 41 universal + 4 android (ast_deeplinks, ast_xml_usages, ast_resource, ast_asset)
    require.Equal(t, 45, len(r.All()))
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/commands/... ./internal/mcp/... -v 2>&1 | grep -E "FAIL|PASS" | head -20
```

Expected: FAIL — ast_resource/ast_asset don't exist yet.

- [ ] **Step 3: Replace ast_resource_usages + ast_resource_unused with ast_resource in allCommands()**

Remove the two entries for `ast_resource_usages` and `ast_resource_unused`. Add in their place:

```go
{
    ToolName:       "ast_resource",
    CLISubcommand:  "resource-usages",
    Description:    "Resource usages (R.* / string / drawable); set unused=true to list unused in a module",
    DataType:       "resource_usages",
    UsesFormatJSON: true,
    Groups:         []string{"android"},
    Args: []ArgDef{
        {Name: "resource", Kind: ArgKindString, Description: "Resource identifier (omit when unused=true)"},
        {Name: "module", Kind: ArgKindString, Description: "Module name (required when unused=true)", Flag: "module"},
        {Name: "unused", Kind: ArgKindBoolean, Description: "List unused resources instead of searching usages"},
    },
},
```

Replace `ast_asset_usages` and `ast_asset_unused` with:

```go
{
    ToolName:       "ast_asset",
    CLISubcommand:  "asset-usages",
    Description:    "Asset usages; set unused=true to list unused assets in a module",
    DataType:       "asset_usages",
    UsesFormatJSON: true,
    Groups:         []string{"android"},
    Args: []ArgDef{
        {Name: "asset", Kind: ArgKindString, Description: "Asset name (omit when unused=true)"},
        {Name: "module", Kind: ArgKindString, Description: "Module name (required when unused=true)", Flag: "module"},
        {Name: "unused", Kind: ArgKindBoolean, Description: "List unused assets instead of searching usages"},
    },
},
```

Also update `"New(nil) universal tool count"` test — after the merge total drops to 59 but universal count stays 41. Confirm it's still 41:

The android group now has 4 tools instead of 6 (2 pairs merged into 1 each). Universal count is unaffected. The `"New(nil) universal tool count"` test expectation of 41 remains correct.

- [ ] **Step 4: Update buildArgv switch in tools.go**

Replace the existing `case "ast_resource_unused":` and `case "ast_asset_unused":` cases with:

```go
case "ast_resource", "ast_asset":
    unused, _ := boolArg(args, "unused")
    if unused {
        module, ok := stringArg(args, "module")
        if !ok || module == "" {
            return nil, fmt.Errorf("missing required argument: module (required when unused=true)")
        }
        argv = append(argv, "--unused", "--module", module)
    } else {
        argName := "resource"
        if def.ToolName == "ast_asset" {
            argName = "asset"
        }
        if v, ok := stringArg(args, argName); ok && v != "" {
            argv = append(argv, v)
        }
    }
    return argv, nil
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/registry.go internal/mcp/tools.go internal/commands/registry_test.go internal/mcp/tools_test.go
git commit -m "feat(commands): merge ast_resource and ast_asset tool pairs, add unused=bool arg"
```

---

## Task 5: Trim descriptions

**Files:**
- Modify: `internal/commands/registry.go` (allCommands descriptions only)

No new tests needed — descriptions are cosmetic.

- [ ] **Step 1: Shorten all descriptions in allCommands()**

Apply these replacements throughout `registry.go`:

```go
// ast_search
"Full-text and semantic search across the index"

// ast_symbol
"Symbol lookup by name or prefix"

// ast_class
"Class, interface, struct or trait declarations"

// ast_file
"Files by name pattern"

// ast_usages
"All usages of a symbol"

// ast_refs
"Cross-references for a symbol: definitions, usages, imports"

// ast_callers
"All call sites of a function"

// ast_call_tree
"Call tree rooted at a function"

// ast_implementations
"Implementations of an interface or abstract class"

// ast_hierarchy
"Class/type inheritance hierarchy"

// ast_outline
"Symbols defined in a file"

// ast_imports
"Imports in a file"

// ast_todo
"TODO/FIXME/HACK comments"

// ast_provides
"Dependency-injection providers for a type"

// ast_suspend
"Kotlin suspend / async functions"

// ast_composables
"Jetpack Compose @Composable functions"

// ast_deprecated
"Deprecated symbols"

// ast_suppress
"Suppressed warnings / lint annotations"

// ast_inject
"Injection points for a type"

// ast_annotations
"Symbols annotated with a specific annotation"

// ast_deeplinks
"Deep-link URI patterns"

// ast_extensions
"Extension functions for a type"

// ast_flows
"Kotlin Flow / reactive stream usages"

// ast_previews
"Compose @Preview functions"

// ast_agrep
"Structural pattern search (requires ast-grep)"

// ast_module
"Modules by name pattern"

// ast_deps
"Dependencies of a module"

// ast_dependents
"Modules that depend on a given module"

// ast_unused_deps
"Unused dependencies of a module"

// ast_api
"Public API surface of a module"

// ast_map
"High-level project structure map"

// ast_conventions
"Project architecture and coding conventions"

// ast_xml_usages
"XML layout usages of a class"

// ast_storyboard_usages
"Storyboard/xib usages of a class"

// ast_swiftui
"SwiftUI views and modifiers"

// ast_async_funcs
"Async/await functions (Swift/Kotlin)"

// ast_publishers
"Combine Publisher declarations"

// ast_main_actor
"@MainActor annotated symbols"

// ast_perl_exports
"Perl module exports"

// ast_perl_subs
"Perl subroutines"

// ast_perl_pod
"Perl POD documentation blocks"

// ast_perl_tests
"Perl test assertions"

// ast_perl_imports
"Perl use/require statements"

// ast_changed
"Symbols changed since a git base branch"

// ast_init
"Initialize the ast-index database"

// ast_rebuild
"Rebuild the index from scratch"

// ast_update
"Incrementally update the index"

// ast_watch
"File-watcher for automatic index updates"

// ast_stats
"Index statistics"

// ast_version
"ast-index version information"

// ast_unused_symbols
"Unused symbols in the codebase"

// ast_add_root
"Add a root directory to the multi-root index"

// ast_list_roots
"All indexed root directories"

// ast_remove_root
"Remove a root directory from the multi-root index"

// ast_query
"Read-only SQL query against the ast-index database"

// ast_db_path
"Path to the ast-index SQLite database file"

// ast_schema
"Database schema (tables, indexes, columns)"
```

- [ ] **Step 2: Run tests**

```bash
go test ./... 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/commands/registry.go
git commit -m "chore(commands): trim verbose tool descriptions"
```

---

## Task 6: Wire main.go — pass cfg.Tools to commands.New()

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update commands.New() call in run()**

In `main.go`, change:

```go
registry := commands.New()
```

to:

```go
registry := commands.New(cfg.Tools)
```

- [ ] **Step 2: Update log message to include active groups**

Replace the `logger.Info` calls to include active groups info:

```go
if guard.IsOpen() {
    logger.Info("starting ast-index-mcp in open mode (no root restriction — pass cwd per call)",
        zap.String("bin", cfg.Bin),
        zap.Int("tools", len(registry.All())),
        zap.Strings("groups", cfg.Tools),
    )
} else {
    logger.Info("starting ast-index-mcp",
        zap.String("bin", cfg.Bin),
        zap.String("cwd", cfg.CWD),
        zap.Int("tools", len(registry.All())),
        zap.Strings("groups", cfg.Tools),
    )
}
```

- [ ] **Step 3: Build to verify no compile errors**

```bash
go build ./...
```

Expected: exits 0, no output.

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: wire cfg.Tools into commands.New, log active groups"
```

---

## Self-Review

**Spec coverage:**
- ✅ `AST_INDEX_TOOLS` parsed in Config (Task 1)
- ✅ `Groups []string` on `CommandDef` (Task 2)
- ✅ `Registry.New(enabledGroups)` filtering (Task 2)
- ✅ kotlin/android/swift/perl group tags (Task 3)
- ✅ `ast_async_funcs` in both kotlin+swift (Task 3)
- ✅ `ast_resource` merges ast_resource_usages + ast_resource_unused (Task 4)
- ✅ `ast_asset` merges ast_asset_usages + ast_asset_unused (Task 4)
- ✅ buildArgv handles merged tools with unused bool (Task 4)
- ✅ Descriptions trimmed (Task 5)
- ✅ main.go wired (Task 6)

**Type consistency:** `commands.New(nil)` used consistently in all test helpers after Task 2. `commands.New([]string{"android"})` used in MergedTools test. No drift.

**Placeholder scan:** No TBDs.
