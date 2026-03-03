package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/config"
)

// Note: t.Setenv modifies process-level env vars, so subtests using it cannot be parallel.

func TestLoad(t *testing.T) {
	t.Run("defaults when no env set", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "")
		t.Setenv("AST_INDEX_CWD", "")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
		t.Setenv("AST_INDEX_LOG_LEVEL", "")

		cfg, err := config.Load()
		require.NoError(t, err)
		require.Equal(t, "ast-index", cfg.Bin)
		require.Equal(t, 60, cfg.TimeoutSec)
		require.Equal(t, "info", cfg.LogLevel)
		// CWD is empty when AST_INDEX_CWD is not set — open mode.
		require.Empty(t, cfg.CWD)
	})

	t.Run("custom bin", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "/usr/local/bin/ast-index")
		t.Setenv("AST_INDEX_CWD", "")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
		t.Setenv("AST_INDEX_LOG_LEVEL", "")

		cfg, err := config.Load()
		require.NoError(t, err)
		require.Equal(t, "/usr/local/bin/ast-index", cfg.Bin)
	})

	t.Run("custom cwd", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "")
		t.Setenv("AST_INDEX_CWD", "/tmp")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
		t.Setenv("AST_INDEX_LOG_LEVEL", "")

		cfg, err := config.Load()
		require.NoError(t, err)
		require.Equal(t, "/tmp", cfg.CWD)
	})

	t.Run("custom timeout", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "")
		t.Setenv("AST_INDEX_CWD", "")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "120")
		t.Setenv("AST_INDEX_LOG_LEVEL", "")

		cfg, err := config.Load()
		require.NoError(t, err)
		require.Equal(t, 120, cfg.TimeoutSec)
	})

	t.Run("invalid timeout non-numeric", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "")
		t.Setenv("AST_INDEX_CWD", "")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "abc")
		t.Setenv("AST_INDEX_LOG_LEVEL", "")

		_, err := config.Load()
		require.Error(t, err)
	})

	t.Run("invalid timeout zero", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "")
		t.Setenv("AST_INDEX_CWD", "")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "0")
		t.Setenv("AST_INDEX_LOG_LEVEL", "")

		_, err := config.Load()
		require.Error(t, err)
	})

	t.Run("custom log level", func(t *testing.T) {
		t.Setenv("AST_INDEX_BIN", "")
		t.Setenv("AST_INDEX_CWD", "")
		t.Setenv("AST_INDEX_TIMEOUT_SEC", "")
		t.Setenv("AST_INDEX_LOG_LEVEL", "debug")

		cfg, err := config.Load()
		require.NoError(t, err)
		require.Equal(t, "debug", cfg.LogLevel)
	})
}
