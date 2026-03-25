package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/commands"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/config"
	internalmcp "github.com/yanodintsovmercuryo/ast-index-mcp/internal/mcp"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/normalize"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/runner"
	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/security"
)

const (
	serverName    = "ast-index-mcp"
	serverVersion = "0.1.0"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ast-index-mcp: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := buildLogger(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	guard, err := security.NewPathGuard(cfg.CWD)
	if err != nil {
		return fmt.Errorf("init path guard: %w", err)
	}

	registry := commands.New(cfg.Tools)
	r := runner.New()
	n := normalize.New()

	handler := internalmcp.NewToolHandler(
		cfg.Bin,
		cfg.CWD,
		cfg.TimeoutSec,
		registry,
		guard,
		r,
		n,
	)

	mcpServer := server.NewMCPServer(serverName, serverVersion)

	for _, def := range registry.All() {
		def := def // capture loop var
		tool := commands.ToMCPTool(def)
		mcpServer.AddTool(tool, func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
			args := req.GetArguments()

			resp := handler.Handle(ctx, def.ToolName, args)

			b, marshalErr := json.Marshal(resp)
			if marshalErr != nil {
				return nil, fmt.Errorf("marshal response: %w", marshalErr)
			}

			return &mcpgo.CallToolResult{
				Content: []mcpgo.Content{
					mcpgo.NewTextContent(string(b)),
				},
				IsError: !resp.Ok,
			}, nil
		})
	}

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

	stdioServer := server.NewStdioServer(mcpServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}

func buildLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stderr"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return cfg.Build()
}
