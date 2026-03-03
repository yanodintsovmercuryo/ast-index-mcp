package runner

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// Result holds the output of a single ast-index command execution.
type Result struct {
	Stderr     string
	Stdout     []byte
	ExitCode   int
	DurationMs int64
	TimedOut   bool
}

// Runner executes ast-index commands as subprocesses.
type Runner struct{}

// New creates a new Runner.
func New() *Runner {
	return &Runner{}
}

// Run executes argv[0] with argv[1:] arguments in the given working directory.
// If timeout > 0, the command is killed after that duration and Result.TimedOut is set.
func (r *Runner) Run(ctx context.Context, argv []string, cwd string, timeout time.Duration) Result {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	//nolint:gosec // argv is controlled by the registry, not user input directly
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if cwd != "" {
		cmd.Dir = cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	result := Result{
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.String(),
		DurationMs: durationMs,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.ExitCode = -1
			return result
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result
}
