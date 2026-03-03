package security

import "fmt"

// PathOutsideRootError is returned when a path escapes the allowed root directory.
type PathOutsideRootError struct {
	Path string
	Root string
}

func (e *PathOutsideRootError) Error() string {
	return fmt.Sprintf("path guard: path %q is outside allowed root %q", e.Path, e.Root)
}
