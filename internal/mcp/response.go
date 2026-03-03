package mcp

import (
	"encoding/json"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/normalize"
)

// Response is the normalized MCP response envelope returned for every tool call.
type Response struct {
	Tool        string                 `json:"tool"`
	Command     string                 `json:"command"`
	CWD         string                 `json:"cwd"`
	Stderr      string                 `json:"stderr"`
	Argv        []string               `json:"argv"`
	Data        json.RawMessage        `json:"data"`
	Diagnostics []normalize.Diagnostic `json:"diagnostics"`
	ExitCode    int                    `json:"exit_code"`
	DurationMs  int64                  `json:"duration_ms"`
	Ok          bool                   `json:"ok"`
	TimedOut    bool                   `json:"timed_out"`
}
