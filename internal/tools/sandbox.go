package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Sandbox confines file tools to a root directory.
type Sandbox struct {
	root string
}

func NewSandbox(root string) (*Sandbox, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	return &Sandbox{root: resolved}, nil
}

// Resolve maps a user-supplied path to an absolute path inside the sandbox,
// following symlinks so a link cannot point outside the root.
func (s *Sandbox) Resolve(p string) (string, error) {
	if !filepath.IsAbs(p) {
		p = filepath.Join(s.root, p)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(p))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the sandbox", p)
	}
	return resolved, nil
}
