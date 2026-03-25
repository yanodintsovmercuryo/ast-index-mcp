package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultBin        = "ast-index"
	defaultTimeoutSec = 60
	defaultLogLevel   = "info"
)

// Config holds all configuration for the MCP server loaded from environment variables.
type Config struct {
	// Bin is the path to the ast-index binary. Env: AST_INDEX_BIN.
	Bin string
	// CWD is the working directory for ast-index commands. Env: AST_INDEX_CWD.
	CWD string
	// LogLevel controls logging verbosity. Env: AST_INDEX_LOG_LEVEL.
	LogLevel string
	// TimeoutSec is the default command timeout in seconds. Env: AST_INDEX_TIMEOUT_SEC.
	TimeoutSec int
	// Tools is the list of opt-in tool groups. Env: AST_INDEX_TOOLS.
	Tools []string
}

// Load reads configuration from environment variables, applying defaults where not set.
// When AST_INDEX_CWD is not set, CWD is empty — the server runs in open mode where any
// path is accepted and the caller must supply cwd on each tool invocation.
func Load() (*Config, error) {
	cwd := os.Getenv("AST_INDEX_CWD")

	timeoutSec := defaultTimeoutSec
	if v := os.Getenv("AST_INDEX_TIMEOUT_SEC"); v != "" {
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil {
			return nil, &InvalidEnvError{Name: "AST_INDEX_TIMEOUT_SEC", Value: v, Err: parseErr}
		}
		if parsed <= 0 {
			return nil, &InvalidEnvError{Name: "AST_INDEX_TIMEOUT_SEC", Value: v, Err: errNonPositiveTimeout}
		}
		timeoutSec = parsed
	}

	bin := defaultBin
	if v := os.Getenv("AST_INDEX_BIN"); v != "" {
		bin = v
	}

	logLevel := defaultLogLevel
	if v := os.Getenv("AST_INDEX_LOG_LEVEL"); v != "" {
		logLevel = v
	}

	var tools []string
	if v := os.Getenv("AST_INDEX_TOOLS"); v != "" {
		for _, g := range strings.Split(v, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				tools = append(tools, g)
			}
		}
	}

	return &Config{
		Bin:        bin,
		CWD:        cwd,
		TimeoutSec: timeoutSec,
		LogLevel:   logLevel,
		Tools:      tools,
	}, nil
}
