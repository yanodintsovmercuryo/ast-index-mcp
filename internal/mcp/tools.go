package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/commands"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/normalize"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/runner"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/security"
)

// sqlDenyPattern matches SQL write operations that are forbidden in ast_query.
var sqlDenyPattern = regexp.MustCompile(
	`(?i)\b(INSERT|UPDATE|DELETE|DROP|ALTER|ATTACH|DETACH|CREATE|REPLACE|TRUNCATE)\b|PRAGMA\s+writable_schema`,
)

// ToolHandler executes MCP tool calls through the unified pipeline.
type ToolHandler struct {
	registry   *commands.Registry
	guard      *security.PathGuard
	runner     *runner.Runner
	normalizer *normalize.Normalizer
	bin        string
	defaultCWD string
	defaultTTL time.Duration
}

// NewToolHandler creates a ToolHandler wired to the provided dependencies.
func NewToolHandler(
	bin string,
	defaultCWD string,
	defaultTimeoutSec int,
	registry *commands.Registry,
	guard *security.PathGuard,
	r *runner.Runner,
	n *normalize.Normalizer,
) *ToolHandler {
	return &ToolHandler{
		bin:        bin,
		defaultCWD: defaultCWD,
		defaultTTL: time.Duration(defaultTimeoutSec) * time.Second,
		registry:   registry,
		guard:      guard,
		runner:     r,
		normalizer: n,
	}
}

// Handle executes the named tool with the provided arguments map and returns a normalized Response.
func (h *ToolHandler) Handle(ctx context.Context, toolName string, args map[string]any) Response {
	def, ok := h.registry.Get(toolName)
	if !ok {
		return errorResponse(toolName, "", nil, "", fmt.Sprintf("unknown tool: %s", toolName))
	}

	// Resolve working directory.
	cwd := h.defaultCWD
	if v, ok := stringArg(args, "cwd"); ok && v != "" {
		if err := h.guard.Validate(v); err != nil {
			return errorResponse(toolName, def.CLISubcommand, nil, cwd, err.Error())
		}
		cwd = v
	}
	// In open mode (AST_INDEX_CWD not set), cwd must be supplied per-call.
	if cwd == "" {
		return errorResponse(toolName, def.CLISubcommand, nil, "",
			"cwd is required: set AST_INDEX_CWD env or pass cwd argument")
	}

	// Resolve timeout.
	timeout := h.defaultTTL
	if v, ok := numberArg(args, "timeout_sec"); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}

	// Build argv.
	argv, buildErr := h.buildArgv(def, args, cwd)
	if buildErr != nil {
		return errorResponse(toolName, def.CLISubcommand, nil, cwd, buildErr.Error())
	}

	// Execute.
	result := h.runner.Run(ctx, argv, cwd, timeout)

	diags := make([]normalize.Diagnostic, 0)

	if result.TimedOut {
		diags = append(diags, normalize.Diagnostic{
			Code:    normalize.DiagnosticCodeTimeout,
			Message: fmt.Sprintf("command %q timed out after %.0fs", toolName, timeout.Seconds()),
		})
		return Response{
			Ok:          false,
			Tool:        toolName,
			Command:     def.CLISubcommand,
			Argv:        argv,
			CWD:         cwd,
			ExitCode:    result.ExitCode,
			DurationMs:  result.DurationMs,
			TimedOut:    true,
			Data:        json.RawMessage(`{"type":"timeout"}`),
			Stderr:      result.Stderr,
			Diagnostics: diags,
		}
	}

	data, normDiags := h.normalizer.Normalize(def.DataType, result.Stdout)
	diags = append(diags, normDiags...)

	dataJSON, _ := json.Marshal(data)

	return Response{
		Ok:          result.ExitCode == 0,
		Tool:        toolName,
		Command:     def.CLISubcommand,
		Argv:        argv,
		CWD:         cwd,
		ExitCode:    result.ExitCode,
		DurationMs:  result.DurationMs,
		TimedOut:    false,
		Data:        dataJSON,
		Stderr:      result.Stderr,
		Diagnostics: diags,
	}
}

// buildArgv constructs the full CLI argument list for a command definition.
func (h *ToolHandler) buildArgv(def commands.CommandDef, args map[string]any, cwd string) ([]string, error) {
	argv := []string{h.bin}

	if def.UsesFormatJSON {
		argv = append(argv, "--format", "json")
	}

	argv = append(argv, def.CLISubcommand)

	// Special handling for ast_resource_unused and ast_asset_unused (need --unused --module flags).
	switch def.ToolName {
	case "ast_resource_unused":
		if module, ok := stringArg(args, "module"); ok && module != "" {
			argv = append(argv, "--unused", "--module", module)
		}
		return argv, nil
	case "ast_asset_unused":
		if module, ok := stringArg(args, "module"); ok && module != "" {
			argv = append(argv, "--unused", "--module", module)
		}
		return argv, nil
	}

	for _, argDef := range def.Args {
		switch argDef.Kind {
		case commands.ArgKindString:
			v, ok := stringArg(args, argDef.Name)
			if !ok || v == "" {
				if argDef.Required {
					return nil, fmt.Errorf("missing required argument: %s", argDef.Name)
				}
				continue
			}
			// Path args need guard validation.
			if argDef.Name == "file" || argDef.Name == "path" {
				if err := h.guard.Validate(v); err != nil {
					return nil, err
				}
			}
			// SQL deny-list.
			if argDef.Name == "sql" {
				if err := validateSQL(v); err != nil {
					return nil, err
				}
			}
			argv = append(argv, v)

		case commands.ArgKindBoolean:
			if v, ok := boolArg(args, argDef.Name); ok && v {
				argv = append(argv, "--"+argDef.Name)
			}

		case commands.ArgKindNumber:
			if v, ok := numberArg(args, argDef.Name); ok && v > 0 {
				argv = append(argv, fmt.Sprintf("--%s=%d", argDef.Name, int64(v)))
			}
		}
	}

	// Append raw_args if the command allows it.
	if def.AllowRawArgs {
		if raw, ok := stringArg(args, "raw_args"); ok && raw != "" {
			argv = append(argv, strings.Fields(raw)...)
		}
	}

	return argv, nil
}

// validateSQL rejects SQL that contains write operations.
func validateSQL(sql string) error {
	if sqlDenyPattern.MatchString(sql) {
		return fmt.Errorf("sql deny-list: only SELECT statements are permitted")
	}
	return nil
}

// errorResponse builds an error Response without running a command.
func errorResponse(toolName, command string, argv []string, cwd, message string) Response {
	if argv == nil {
		argv = []string{}
	}
	return Response{
		Ok:      false,
		Tool:    toolName,
		Command: command,
		Argv:    argv,
		CWD:     cwd,
		Data:    json.RawMessage(`{"type":"error"}`),
		Diagnostics: []normalize.Diagnostic{
			{Code: "ERROR", Message: message},
		},
	}
}

// stringArg extracts a string value from the args map.
func stringArg(args map[string]any, name string) (string, bool) {
	v, ok := args[name]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// boolArg extracts a boolean value from the args map.
func boolArg(args map[string]any, name string) (bool, bool) {
	v, ok := args[name]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// numberArg extracts a numeric value from the args map (JSON numbers are float64).
func numberArg(args map[string]any, name string) (float64, bool) {
	v, ok := args[name]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
