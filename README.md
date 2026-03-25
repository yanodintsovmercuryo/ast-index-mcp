# ast-index-mcp

Go MCP (Model Context Protocol) stdio server that exposes [`ast-index`](https://github.com/defendend/Claude-ast-index-search) as 46+ structured MCP tools.

## Prerequisites

### Install ast-index

```bash
brew install defendend/tap/ast-index
```

Or build from source — see the [ast-index repository](https://github.com/defendend/Claude-ast-index-search).

### Optional: install ast-grep (for specific commands)

```bash
brew install ast-grep
```

Required only for `ast_agrep` (CLI subcommand `agrep`). Other commands do not require `ast-grep`.



### Install ast-index-mcp

```bash
go install github.com/yanodintsovmercuryo/ast-index-mcp@latest
```

Or build locally:

```bash
git clone https://github.com/yanodintsovmercuryo/ast-index-mcp
cd ast-index-mcp
go build -o ast-index-mcp ./
```

## MCP Client Configuration

### Recommended: open mode (one server — multiple projects)

Do **not** set `AST_INDEX_CWD`. Every tool call must include a `cwd` argument pointing to the project root. This lets a single server instance serve all your projects.

```json
{
  "mcpServers": {
    "ast-index": {
      "command": "ast-index-mcp",
      "args": [],
      "env": {
        "AST_INDEX_BIN": "ast-index"
      }
    }
  }
}
```

> If `go install` placed the binary somewhere not on your `PATH`, use the full path instead: `"command": "/path/to/ast-index-mcp"`.
> Run `which ast-index-mcp` to find it, or make sure `$(go env GOPATH)/bin` is in your `PATH`.

Each tool call then includes `cwd`:
```json
{ "tool": "ast_search", "arguments": { "query": "UserRepository", "cwd": "/projects/my-app" } }
```

### Alternative: restricted mode (single project)

Set `AST_INDEX_CWD` to lock the server to one project root. Path arguments are validated to stay within that root.

```json
{
  "mcpServers": {
    "ast-index": {
      "command": "ast-index-mcp",
      "args": [],
      "env": {
        "AST_INDEX_BIN": "ast-index",
        "AST_INDEX_CWD": "/path/to/your/project"
      }
    }
  }
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AST_INDEX_BIN` | `ast-index` | Path to the ast-index binary |
| `AST_INDEX_CWD` | *(empty)* | If set: restrict all paths to this root. If empty: open mode — `cwd` required per call |
| `AST_INDEX_TIMEOUT_SEC` | `60` | Default command timeout in seconds |
| `AST_INDEX_LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |

## Available Tools (46+)

### Search & Symbols
| Tool | Description |
|------|-------------|
| `ast_search` | Full-text and semantic search |
| `ast_symbol` | Find symbols by name |
| `ast_class` | Find class/interface/struct declarations |
| `ast_file` | Find files by name pattern |
| `ast_usages` | Find all usages of a symbol |
| `ast_refs` | Cross-references: definitions, usages, imports |
| `ast_callers` | Find call sites of a function |
| `ast_call_tree` | Build call tree |
| `ast_implementations` | Find implementations of an interface |
| `ast_hierarchy` | Type inheritance hierarchy |
| `ast_outline` | Symbols defined in a file |
| `ast_imports` | Imports in a file |

### Pattern / Grep
`ast_todo`, `ast_provides`, `ast_suspend`, `ast_composables`, `ast_deprecated`,
`ast_suppress`, `ast_inject`, `ast_annotations`, `ast_deeplinks`, `ast_extensions`,
`ast_flows`, `ast_previews`, `ast_agrep`

### Modules & Dependencies
`ast_module`, `ast_deps`, `ast_dependents`, `ast_unused_deps`, `ast_api`, `ast_map`, `ast_conventions`

### Resources / XML / iOS
`ast_xml_usages`, `ast_resource_usages`, `ast_resource_unused`, `ast_storyboard_usages`,
`ast_asset_usages`, `ast_asset_unused`, `ast_swiftui`, `ast_async_funcs`, `ast_publishers`, `ast_main_actor`

### Perl
`ast_perl_exports`, `ast_perl_subs`, `ast_perl_pod`, `ast_perl_tests`, `ast_perl_imports`

### Index Management
`ast_changed`, `ast_init`, `ast_rebuild`, `ast_update`, `ast_watch`, `ast_stats`, `ast_version`, `ast_unused_symbols`

### Multi-root
`ast_add_root`, `ast_list_roots`, `ast_remove_root`

### SQL / DB
`ast_query`, `ast_db_path`, `ast_schema`

## Response Envelope

Every tool returns a JSON object with the same structure:

```json
{
  "ok": true,
  "tool": "ast_search",
  "command": "search",
  "argv": ["ast-index", "--format", "json", "search", "UserRepository"],
  "cwd": "/repo",
  "exit_code": 0,
  "duration_ms": 42,
  "timed_out": false,
  "data": {
    "type": "search_hits",
    "payload": { "..." : "..." }
  },
  "stderr": "",
  "diagnostics": []
}
```

## Common Arguments (all tools)

| Argument | Type | Description |
|----------|------|-------------|
| `cwd` | string | Absolute path to the project root. **Required** when `AST_INDEX_CWD` is not set; optional otherwise (defaults to `AST_INDEX_CWD`). |
| `timeout_sec` | number | Override timeout for this call |

## Example Tool Calls

**Search for a class:**
```json
{ "tool": "ast_search", "arguments": { "query": "class UserRepository" } }
```

**Get index statistics:**
```json
{ "tool": "ast_stats", "arguments": {} }
```

**Run a read-only SQL query:**
```json
{ "tool": "ast_query", "arguments": { "sql": "SELECT name, file FROM symbols LIMIT 10" } }
```

**Show file outline:**
```json
{ "tool": "ast_outline", "arguments": { "file": "src/user/repository.go" } }
```

## Security

- All path arguments are validated to be inside `AST_INDEX_CWD`.
- Symlinks are resolved before the check to prevent escape.
- `ast_query` only permits `SELECT` statements — write operations (`INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `ATTACH`, `DETACH`, `PRAGMA writable_schema`) are blocked.

## Troubleshooting

**`ast-index not found`**
Set `AST_INDEX_BIN` to the full path: `which ast-index`.

**`path outside allowed root`**
The `file`/`path`/`cwd` argument points outside `AST_INDEX_CWD`. Check that `AST_INDEX_CWD` is set to your project root.

**Timeout**
Increase `AST_INDEX_TIMEOUT_SEC` or pass `timeout_sec` per-call. Long operations like `ast_rebuild` on large codebases may need 120–300 seconds.

**Normalization fallback diagnostic**
The `ast-index` version installed does not support `--format json` for this command. The raw text output is still returned in `data.payload.raw`. Update `ast-index` to the latest version.

**Registry out of sync**
If `ast-index --help` lists commands not exposed here, open an issue or update `internal/commands/registry.go`.
