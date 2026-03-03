package security_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yanodintsovmercuryo/ast-index-mcp/internal/security"
)

func TestPathGuard_Validate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	guard, err := security.NewPathGuard(root)
	require.NoError(t, err)

	t.Run("allows root itself", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, guard.Validate(root))
	})

	t.Run("allows subdirectory", func(t *testing.T) {
		t.Parallel()
		sub := filepath.Join(root, "sub", "dir")
		require.NoError(t, guard.Validate(sub))
	})

	t.Run("allows file inside root", func(t *testing.T) {
		t.Parallel()
		f := filepath.Join(root, "file.txt")
		require.NoError(t, guard.Validate(f))
	})

	t.Run("denies parent directory", func(t *testing.T) {
		t.Parallel()
		err := guard.Validate(filepath.Dir(root))
		var pathErr *security.PathOutsideRootError
		require.ErrorAs(t, err, &pathErr)
	})

	t.Run("denies dotdot traversal", func(t *testing.T) {
		t.Parallel()
		escape := filepath.Join(root, "..", "escape")
		err := guard.Validate(escape)
		var pathErr *security.PathOutsideRootError
		require.ErrorAs(t, err, &pathErr)
	})

	t.Run("denies absolute path outside root", func(t *testing.T) {
		t.Parallel()
		err := guard.Validate("/etc/passwd")
		var pathErr *security.PathOutsideRootError
		require.ErrorAs(t, err, &pathErr)
	})

	t.Run("denies empty path", func(t *testing.T) {
		t.Parallel()
		require.Error(t, guard.Validate(""))
	})

	t.Run("denies symlink escaping root", func(t *testing.T) {
		t.Parallel()

		// Create a real outside dir.
		outside := t.TempDir()

		// Create a symlink inside root pointing outside.
		symlink := filepath.Join(root, "escape_link")
		if err := os.Symlink(outside, symlink); err != nil {
			t.Skip("cannot create symlink:", err)
		}

		err := guard.Validate(symlink)
		var pathErr *security.PathOutsideRootError
		require.ErrorAs(t, err, &pathErr)
	})
}

func TestNewPathGuard(t *testing.T) {
	t.Parallel()

	t.Run("empty root creates open guard", func(t *testing.T) {
		t.Parallel()
		g, err := security.NewPathGuard("")
		require.NoError(t, err)
		require.True(t, g.IsOpen())
		require.NoError(t, g.Validate("/etc/passwd")) // open mode — everything passes
		require.Error(t, g.Validate(""))              // empty path still rejected
	})

	t.Run("non-existent root is accepted", func(t *testing.T) {
		t.Parallel()
		_, err := security.NewPathGuard("/tmp/definitely-does-not-exist-xyz123")
		require.NoError(t, err)
	})
}
