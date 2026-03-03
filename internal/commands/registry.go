package commands

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ArgKind describes the type of a command argument.
type ArgKind string

const (
	ArgKindString  ArgKind = "string"
	ArgKindBoolean ArgKind = "boolean"
	ArgKindNumber  ArgKind = "number"
	ArgKindArray   ArgKind = "array"
)

// ArgDef describes a single input argument for a command.
type ArgDef struct {
	Name        string
	Kind        ArgKind
	Description string
	Required    bool
}

// CommandDef describes a single ast-index command exposed as an MCP tool.
type CommandDef struct {
	// ToolName is the MCP tool name, e.g. "ast_search".
	ToolName string
	// CLISubcommand is the ast-index subcommand, e.g. "search".
	CLISubcommand string
	// Description is the human-readable tool description.
	Description string
	// DataType is the value of data.type in the normalized response.
	DataType string
	// Args is the ordered list of input argument definitions.
	Args []ArgDef
	// UsesFormatJSON indicates whether --format json should be added to the CLI call.
	UsesFormatJSON bool
	// AllowRawArgs permits callers to append raw_args to the CLI invocation.
	AllowRawArgs bool
}

// Registry is the static registry of all supported ast-index commands.
type Registry struct {
	commands map[string]CommandDef
}

// New builds and validates the command registry.
// Panics if any tool name is duplicated.
func New() *Registry {
	defs := allCommands()

	m := make(map[string]CommandDef, len(defs))
	for _, d := range defs {
		if _, exists := m[d.ToolName]; exists {
			panic(fmt.Sprintf("commands: duplicate tool name %q", d.ToolName))
		}
		m[d.ToolName] = d
	}
	return &Registry{commands: m}
}

// Get returns the CommandDef for the given MCP tool name.
func (r *Registry) Get(toolName string) (CommandDef, bool) {
	d, ok := r.commands[toolName]
	return d, ok
}

// All returns all registered CommandDefs.
func (r *Registry) All() []CommandDef {
	out := make([]CommandDef, 0, len(r.commands))
	for _, d := range r.commands {
		out = append(out, d)
	}
	return out
}

// ToMCPTool converts a CommandDef to a mcp.Tool for registration with the MCP server.
func ToMCPTool(d CommandDef) mcp.Tool {
	opts := []mcp.ToolOption{mcp.WithDescription(d.Description)}
	for _, arg := range d.Args {
		switch arg.Kind {
		case ArgKindString:
			if arg.Required {
				opts = append(opts, mcp.WithString(arg.Name, mcp.Required(), mcp.Description(arg.Description)))
			} else {
				opts = append(opts, mcp.WithString(arg.Name, mcp.Description(arg.Description)))
			}
		case ArgKindBoolean:
			opts = append(opts, mcp.WithBoolean(arg.Name, mcp.Description(arg.Description)))
		case ArgKindNumber:
			if arg.Required {
				opts = append(opts, mcp.WithNumber(arg.Name, mcp.Required(), mcp.Description(arg.Description)))
			} else {
				opts = append(opts, mcp.WithNumber(arg.Name, mcp.Description(arg.Description)))
			}
		}
	}

	// Common fields for all tools. cwd is optional when AST_INDEX_CWD env is set (server uses
	// the env value as default), and required per-call only in open mode (AST_INDEX_CWD unset).
	opts = append(opts,
		mcp.WithString("cwd", mcp.Description("Absolute path to the project root. Required when AST_INDEX_CWD env is not set; optional otherwise.")),
		mcp.WithNumber("timeout_sec", mcp.Description("Override command timeout in seconds")),
	)
	if d.AllowRawArgs {
		opts = append(opts, mcp.WithString("raw_args", mcp.Description("Extra CLI flags as space-separated string")))
	}

	return mcp.NewTool(d.ToolName, opts...)
}

// allCommands returns the full list of command definitions.
func allCommands() []CommandDef {
	return []CommandDef{
		// ── 6.1 Search & Symbols ──────────────────────────────────────────────
		{
			ToolName:       "ast_search",
			CLISubcommand:  "search",
			Description:    "Full-text and semantic search across the indexed codebase",
			DataType:       "search_hits",
			UsesFormatJSON: true,
			AllowRawArgs:   true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Required: true, Description: "Search query"},
				{Name: "limit", Kind: ArgKindNumber, Description: "Maximum number of results"},
				{Name: "kind", Kind: ArgKindString, Description: "Filter by symbol kind (class, func, etc.)"},
			},
		},
		{
			ToolName:       "ast_symbol",
			CLISubcommand:  "symbol",
			Description:    "Find symbols by name",
			DataType:       "symbols",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "name", Kind: ArgKindString, Required: true, Description: "Symbol name or prefix"},
				{Name: "kind", Kind: ArgKindString, Description: "Filter by symbol kind"},
			},
		},
		{
			ToolName:       "ast_class",
			CLISubcommand:  "class",
			Description:    "Find class, interface, struct or trait declarations",
			DataType:       "symbols",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "name", Kind: ArgKindString, Required: true, Description: "Class/interface/struct name"},
			},
		},
		{
			ToolName:       "ast_file",
			CLISubcommand:  "file",
			Description:    "Find files by name pattern",
			DataType:       "files",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "pattern", Kind: ArgKindString, Required: true, Description: "File name pattern"},
				{Name: "glob", Kind: ArgKindString, Description: "Additional glob filter"},
			},
		},
		{
			ToolName:       "ast_usages",
			CLISubcommand:  "usages",
			Description:    "Find all usages of a symbol",
			DataType:       "references",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "symbol", Kind: ArgKindString, Required: true, Description: "Symbol name"},
				{Name: "scope", Kind: ArgKindString, Description: "Limit search to a module or directory scope"},
			},
		},
		{
			ToolName:       "ast_refs",
			CLISubcommand:  "refs",
			Description:    "Get cross-references for a symbol: definitions, usages, imports",
			DataType:       "cross_references",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "symbol", Kind: ArgKindString, Required: true, Description: "Symbol name"},
			},
		},
		{
			ToolName:       "ast_callers",
			CLISubcommand:  "callers",
			Description:    "Find all call sites of a function",
			DataType:       "call_sites",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "function", Kind: ArgKindString, Required: true, Description: "Function name"},
			},
		},
		{
			ToolName:       "ast_call_tree",
			CLISubcommand:  "call-tree",
			Description:    "Build call tree rooted at a function",
			DataType:       "call_tree",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "function", Kind: ArgKindString, Required: true, Description: "Root function name"},
				{Name: "depth", Kind: ArgKindNumber, Description: "Maximum tree depth"},
			},
		},
		{
			ToolName:       "ast_implementations",
			CLISubcommand:  "implementations",
			Description:    "Find all implementations of an interface or abstract class",
			DataType:       "implementations",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "parent", Kind: ArgKindString, Required: true, Description: "Interface or base class name"},
			},
		},
		{
			ToolName:       "ast_hierarchy",
			CLISubcommand:  "hierarchy",
			Description:    "Show class/type inheritance hierarchy",
			DataType:       "hierarchy",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "class_name", Kind: ArgKindString, Required: true, Description: "Class name"},
				{Name: "depth", Kind: ArgKindNumber, Description: "Maximum depth"},
			},
		},
		{
			ToolName:       "ast_outline",
			CLISubcommand:  "outline",
			Description:    "Show all symbols defined in a file",
			DataType:       "file_outline",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "file", Kind: ArgKindString, Required: true, Description: "File path (relative or absolute)"},
			},
		},
		{
			ToolName:       "ast_imports",
			CLISubcommand:  "imports",
			Description:    "List all imports in a file",
			DataType:       "imports",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "file", Kind: ArgKindString, Required: true, Description: "File path (relative or absolute)"},
			},
		},

		// ── 6.2 Grep / Pattern commands ───────────────────────────────────────
		{
			ToolName:       "ast_todo",
			CLISubcommand:  "todo",
			Description:    "Find TODO/FIXME/HACK comments",
			DataType:       "todo_items",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "pattern", Kind: ArgKindString, Description: "Filter pattern"},
			},
		},
		{
			ToolName:       "ast_provides",
			CLISubcommand:  "provides",
			Description:    "Find dependency-injection providers for a type",
			DataType:       "di_providers",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "type_name", Kind: ArgKindString, Required: true, Description: "Type name to search providers for"},
			},
		},
		{
			ToolName:       "ast_suspend",
			CLISubcommand:  "suspend",
			Description:    "Find Kotlin suspend / async functions",
			DataType:       "symbols",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional name filter"},
			},
		},
		{
			ToolName:       "ast_composables",
			CLISubcommand:  "composables",
			Description:    "Find Jetpack Compose @Composable functions",
			DataType:       "symbols",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional name filter"},
			},
		},
		{
			ToolName:       "ast_deprecated",
			CLISubcommand:  "deprecated",
			Description:    "Find deprecated symbols",
			DataType:       "deprecated_items",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional name filter"},
			},
		},
		{
			ToolName:       "ast_suppress",
			CLISubcommand:  "suppress",
			Description:    "Find suppressed warnings / lint annotations",
			DataType:       "annotations",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_inject",
			CLISubcommand:  "inject",
			Description:    "Find injection points for a type",
			DataType:       "di_injections",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "type_name", Kind: ArgKindString, Required: true, Description: "Type name to find injections for"},
			},
		},
		{
			ToolName:       "ast_annotations",
			CLISubcommand:  "annotations",
			Description:    "Find symbols annotated with a specific annotation",
			DataType:       "annotations",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "annotation", Kind: ArgKindString, Required: true, Description: "Annotation name (e.g. @Inject)"},
			},
		},
		{
			ToolName:       "ast_deeplinks",
			CLISubcommand:  "deeplinks",
			Description:    "Find deep-link URI patterns",
			DataType:       "deeplinks",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional URI pattern filter"},
			},
		},
		{
			ToolName:       "ast_extensions",
			CLISubcommand:  "extensions",
			Description:    "Find extension functions for a type",
			DataType:       "extension_functions",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "type_name", Kind: ArgKindString, Required: true, Description: "Receiver type name"},
			},
		},
		{
			ToolName:       "ast_flows",
			CLISubcommand:  "flows",
			Description:    "Find Kotlin Flow / reactive stream usages",
			DataType:       "reactive_streams",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_previews",
			CLISubcommand:  "previews",
			Description:    "Find Compose @Preview functions",
			DataType:       "preview_items",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_agrep",
			CLISubcommand:  "agrep",
			Description:    "AST-aware structural pattern search",
			DataType:       "pattern_matches",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "pattern", Kind: ArgKindString, Required: true, Description: "Structural pattern"},
				{Name: "lang", Kind: ArgKindString, Description: "Language filter"},
			},
		},

		// ── 6.3 Modules & Dependencies ────────────────────────────────────────
		{
			ToolName:       "ast_module",
			CLISubcommand:  "module",
			Description:    "Find modules by name pattern",
			DataType:       "modules",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "pattern", Kind: ArgKindString, Required: true, Description: "Module name pattern"},
			},
		},
		{
			ToolName:       "ast_deps",
			CLISubcommand:  "deps",
			Description:    "List dependencies of a module",
			DataType:       "dependencies",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Required: true, Description: "Module name"},
				{Name: "transitive", Kind: ArgKindBoolean, Description: "Include transitive dependencies"},
			},
		},
		{
			ToolName:       "ast_dependents",
			CLISubcommand:  "dependents",
			Description:    "Find modules that depend on a given module",
			DataType:       "dependents",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Required: true, Description: "Module name"},
				{Name: "transitive", Kind: ArgKindBoolean, Description: "Include transitive dependents"},
			},
		},
		{
			ToolName:       "ast_unused_deps",
			CLISubcommand:  "unused-deps",
			Description:    "Find unused dependencies of a module",
			DataType:       "unused_dependencies",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Required: true, Description: "Module name"},
			},
		},
		{
			ToolName:       "ast_api",
			CLISubcommand:  "api",
			Description:    "Show public API surface of a module",
			DataType:       "public_api",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Required: true, Description: "Module name"},
			},
		},
		{
			ToolName:       "ast_map",
			CLISubcommand:  "map",
			Description:    "Show high-level project structure map",
			DataType:       "project_map",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "max_depth", Kind: ArgKindNumber, Description: "Maximum directory depth"},
			},
		},
		{
			ToolName:       "ast_conventions",
			CLISubcommand:  "conventions",
			Description:    "Detect project architecture and coding conventions",
			DataType:       "conventions",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "focus", Kind: ArgKindString, Description: "Comma-separated focus areas (architecture, frameworks, naming)"},
			},
		},

		// ── 6.4 Resources / XML / iOS ─────────────────────────────────────────
		{
			ToolName:       "ast_xml_usages",
			CLISubcommand:  "xml-usages",
			Description:    "Find XML layout usages of a class",
			DataType:       "xml_usages",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "class_name", Kind: ArgKindString, Required: true, Description: "Class name"},
			},
		},
		{
			ToolName:       "ast_resource_usages",
			CLISubcommand:  "resource-usages",
			Description:    "Find usages of a resource (R.* / string / drawable)",
			DataType:       "resource_usages",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "resource", Kind: ArgKindString, Required: true, Description: "Resource identifier"},
			},
		},
		{
			ToolName:       "ast_resource_unused",
			CLISubcommand:  "resource-usages",
			Description:    "Find unused resources in a module",
			DataType:       "unused_resources",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Required: true, Description: "Module name"},
			},
		},
		{
			ToolName:       "ast_storyboard_usages",
			CLISubcommand:  "storyboard-usages",
			Description:    "Find storyboard/xib usages of a class",
			DataType:       "storyboard_usages",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "class_name", Kind: ArgKindString, Required: true, Description: "Class name"},
			},
		},
		{
			ToolName:       "ast_asset_usages",
			CLISubcommand:  "asset-usages",
			Description:    "Find usages of an asset",
			DataType:       "asset_usages",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "asset", Kind: ArgKindString, Description: "Asset name (optional)"},
			},
		},
		{
			ToolName:       "ast_asset_unused",
			CLISubcommand:  "asset-usages",
			Description:    "Find unused assets in a module",
			DataType:       "unused_assets",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Required: true, Description: "Module name"},
			},
		},
		{
			ToolName:       "ast_swiftui",
			CLISubcommand:  "swiftui",
			Description:    "Find SwiftUI views and modifiers",
			DataType:       "swiftui_items",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_async_funcs",
			CLISubcommand:  "async-funcs",
			Description:    "Find async/await functions (Swift/Kotlin)",
			DataType:       "async_functions",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_publishers",
			CLISubcommand:  "publishers",
			Description:    "Find Combine Publisher declarations",
			DataType:       "combine_publishers",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_main_actor",
			CLISubcommand:  "main-actor",
			Description:    "Find @MainActor annotated symbols",
			DataType:       "main_actor_items",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},

		// ── 6.5 Perl ──────────────────────────────────────────────────────────
		{
			ToolName:       "ast_perl_exports",
			CLISubcommand:  "perl-exports",
			Description:    "Find Perl module exports",
			DataType:       "perl_exports",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_perl_subs",
			CLISubcommand:  "perl-subs",
			Description:    "Find Perl subroutines",
			DataType:       "perl_subroutines",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_perl_pod",
			CLISubcommand:  "perl-pod",
			Description:    "Find Perl POD documentation blocks",
			DataType:       "perl_pod_docs",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_perl_tests",
			CLISubcommand:  "perl-tests",
			Description:    "Find Perl test assertions",
			DataType:       "perl_test_assertions",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},
		{
			ToolName:       "ast_perl_imports",
			CLISubcommand:  "perl-imports",
			Description:    "Find Perl use/require statements",
			DataType:       "perl_imports",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "query", Kind: ArgKindString, Description: "Optional filter"},
			},
		},

		// ── 6.6 Changes & Index ───────────────────────────────────────────────
		{
			ToolName:       "ast_changed",
			CLISubcommand:  "changed",
			Description:    "Show symbols changed since a git base branch",
			DataType:       "changed_symbols",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "base", Kind: ArgKindString, Description: "Base branch or commit (default: main)"},
			},
		},
		{
			ToolName:       "ast_init",
			CLISubcommand:  "init",
			Description:    "Initialize the ast-index database",
			DataType:       "index_operation",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "force", Kind: ArgKindBoolean, Description: "Force re-initialization"},
			},
		},
		{
			ToolName:       "ast_rebuild",
			CLISubcommand:  "rebuild",
			Description:    "Rebuild the index from scratch",
			DataType:       "index_operation",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "type", Kind: ArgKindString, Description: "Index type filter"},
				{Name: "project_type", Kind: ArgKindString, Description: "Project type hint"},
			},
		},
		{
			ToolName:       "ast_update",
			CLISubcommand:  "update",
			Description:    "Incrementally update the index",
			DataType:       "index_operation",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "since", Kind: ArgKindString, Description: "Update since this git ref"},
			},
		},
		{
			ToolName:       "ast_watch",
			CLISubcommand:  "watch",
			Description:    "Start file-watcher for automatic index updates",
			DataType:       "watch_status",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "debounce_ms", Kind: ArgKindNumber, Description: "Debounce interval in milliseconds"},
			},
		},
		{
			ToolName:       "ast_stats",
			CLISubcommand:  "stats",
			Description:    "Show index statistics",
			DataType:       "index_stats",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "details", Kind: ArgKindBoolean, Description: "Include detailed breakdown"},
			},
		},
		{
			ToolName:      "ast_version",
			CLISubcommand: "version",
			Description:   "Show ast-index version information",
			DataType:      "version_info",
			Args:          []ArgDef{},
		},
		{
			ToolName:       "ast_unused_symbols",
			CLISubcommand:  "unused-symbols",
			Description:    "Find unused symbols in the codebase",
			DataType:       "unused_symbols",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "module", Kind: ArgKindString, Description: "Limit to module"},
				{Name: "visibility", Kind: ArgKindString, Description: "Filter by visibility (public, internal, private)"},
			},
		},

		// ── 6.7 Multi-root ────────────────────────────────────────────────────
		{
			ToolName:       "ast_add_root",
			CLISubcommand:  "add-root",
			Description:    "Add a new root directory to the multi-root index",
			DataType:       "roots_operation",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "path", Kind: ArgKindString, Required: true, Description: "Absolute path to add"},
			},
		},
		{
			ToolName:       "ast_list_roots",
			CLISubcommand:  "list-roots",
			Description:    "List all indexed root directories",
			DataType:       "roots",
			UsesFormatJSON: true,
			Args:           []ArgDef{},
		},
		{
			ToolName:       "ast_remove_root",
			CLISubcommand:  "remove-root",
			Description:    "Remove a root directory from the multi-root index",
			DataType:       "roots_operation",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "path", Kind: ArgKindString, Required: true, Description: "Root path to remove"},
			},
		},

		// ── 6.8 SQL / DB introspection ────────────────────────────────────────
		{
			ToolName:       "ast_query",
			CLISubcommand:  "query",
			Description:    "Execute a read-only SQL query against the ast-index database",
			DataType:       "sql_result",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "sql", Kind: ArgKindString, Required: true, Description: "SELECT statement to execute"},
				{Name: "limit", Kind: ArgKindNumber, Description: "Maximum number of rows"},
			},
		},
		{
			ToolName:       "ast_db_path",
			CLISubcommand:  "db-path",
			Description:    "Show the path to the ast-index SQLite database file",
			DataType:       "db_info",
			UsesFormatJSON: true,
			Args:           []ArgDef{},
		},
		{
			ToolName:       "ast_schema",
			CLISubcommand:  "schema",
			Description:    "Show the database schema (tables, indexes, columns)",
			DataType:       "db_schema",
			UsesFormatJSON: true,
			Args: []ArgDef{
				{Name: "table", Kind: ArgKindString, Description: "Filter by table name"},
			},
		},
	}
}
