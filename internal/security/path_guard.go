package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathGuard restricts filesystem access to paths within an allowed root directory.
// When allowedRoot is empty the guard operates in open mode — any absolute path is permitted.
type PathGuard struct {
	allowedRoot string
}

// NewPathGuard creates a PathGuard that permits only paths within allowedRoot.
// If allowedRoot is empty, the guard operates in open mode — any absolute path is allowed.
// allowedRoot is resolved to its real path (symlinks evaluated) at construction time.
func NewPathGuard(allowedRoot string) (*PathGuard, error) {
	if allowedRoot == "" {
		return &PathGuard{}, nil
	}
	real, err := filepath.EvalSymlinks(allowedRoot)
	if err != nil {
		if os.IsNotExist(err) {
			// If root doesn't exist yet, use cleaned absolute path.
			abs, absErr := filepath.Abs(allowedRoot)
			if absErr != nil {
				return nil, fmt.Errorf("path guard: resolve allowed root: %w", absErr)
			}
			real = abs
		} else {
			return nil, fmt.Errorf("path guard: eval symlinks for root: %w", err)
		}
	}
	return &PathGuard{allowedRoot: real}, nil
}

// Validate checks that path is inside the allowed root.
// In open mode (allowedRoot empty), any non-empty path is accepted.
// Symlinks in path are resolved before the check.
// Returns PathOutsideRootError if the path escapes the allowed root.
func (g *PathGuard) Validate(path string) error {
	if path == "" {
		return errors.New("path guard: path must not be empty")
	}
	if g.IsOpen() {
		return nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("path guard: make path absolute: %w", err)
	}

	real, err := evalSymlinksPartial(abs)
	if err != nil {
		return fmt.Errorf("path guard: eval symlinks: %w", err)
	}

	if !isSubPath(g.allowedRoot, real) {
		return &PathOutsideRootError{Path: path, Root: g.allowedRoot}
	}
	return nil
}

// AllowedRoot returns the resolved allowed root directory, or empty string in open mode.
func (g *PathGuard) AllowedRoot() string {
	return g.allowedRoot
}

// IsOpen reports whether the guard is in open mode (no root restriction).
func (g *PathGuard) IsOpen() bool {
	return g.allowedRoot == ""
}

// isSubPath reports whether child is allowedRoot or a descendant of it.
func isSubPath(root, child string) bool {
	if root == child {
		return true
	}
	return strings.HasPrefix(child, root+string(filepath.Separator))
}

// evalSymlinksPartial resolves symlinks on the longest existing prefix of path,
// then appends the remaining non-existent suffix. This handles macOS /var → /private/var
// and similar symlink patterns even when the full path doesn't exist yet.
func evalSymlinksPartial(path string) (string, error) {
	real, err := filepath.EvalSymlinks(path)
	if err == nil {
		return real, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	// Walk up to the nearest existing ancestor.
	dir := path
	var suffix []string
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root with no existing component.
			return filepath.Clean(path), nil
		}
		suffix = append([]string{filepath.Base(dir)}, suffix...)
		dir = parent

		resolved, resolveErr := filepath.EvalSymlinks(dir)
		if resolveErr == nil {
			return filepath.Join(append([]string{resolved}, suffix...)...), nil
		}
		if !os.IsNotExist(resolveErr) {
			return "", resolveErr
		}
	}
}
