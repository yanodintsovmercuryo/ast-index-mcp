package runner_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/runner"
)

func TestRunner_Run(t *testing.T) {
	t.Parallel()

	r := runner.New()

	t.Run("successful command with output", func(t *testing.T) {
		t.Parallel()
		result := r.Run(context.Background(), []string{"echo", "hello"}, "", 5*time.Second)
		require.Equal(t, 0, result.ExitCode)
		require.Equal(t, "hello\n", string(result.Stdout))
		require.False(t, result.TimedOut)
		require.GreaterOrEqual(t, result.DurationMs, int64(0))
	})

	t.Run("non-zero exit code", func(t *testing.T) {
		t.Parallel()
		result := r.Run(context.Background(), []string{"false"}, "", 5*time.Second)
		require.Equal(t, 1, result.ExitCode)
		require.False(t, result.TimedOut)
	})

	t.Run("stderr captured", func(t *testing.T) {
		t.Parallel()
		result := r.Run(context.Background(), []string{"sh", "-c", "echo err >&2"}, "", 5*time.Second)
		require.Equal(t, 0, result.ExitCode)
		require.Equal(t, "err\n", result.Stderr)
	})

	t.Run("timeout triggers", func(t *testing.T) {
		t.Parallel()
		result := r.Run(context.Background(), []string{"sleep", "10"}, "", 100*time.Millisecond)
		require.True(t, result.TimedOut)
		require.Equal(t, -1, result.ExitCode)
	})

	t.Run("no timeout when zero", func(t *testing.T) {
		t.Parallel()
		result := r.Run(context.Background(), []string{"echo", "ok"}, "", 0)
		require.Equal(t, 0, result.ExitCode)
		require.False(t, result.TimedOut)
	})

	t.Run("respects cwd", func(t *testing.T) {
		t.Parallel()
		result := r.Run(context.Background(), []string{"pwd"}, "/tmp", 5*time.Second)
		require.Equal(t, 0, result.ExitCode)
		require.Contains(t, string(result.Stdout), "tmp")
	})

	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := r.Run(ctx, []string{"sleep", "10"}, "", 5*time.Second)
		require.NotEqual(t, 0, result.ExitCode)
	})
}
