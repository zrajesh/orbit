package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxResolve(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	os.WriteFile(filepath.Join(root, "ok.txt"), []byte("hi"), 0o644)
	os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("no"), 0o644)
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	sb, err := NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"inside relative", "ok.txt", false},
		{"dot root", ".", false},
		{"parent traversal", "../" + filepath.Base(outside) + "/secret.txt", true},
		{"absolute outside", filepath.Join(outside, "secret.txt"), true},
		{"symlink escape", "link/secret.txt", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := sb.Resolve(c.path)
			if (err != nil) != c.wantErr {
				t.Fatalf("Resolve(%q) err=%v, wantErr=%v", c.path, err, c.wantErr)
			}
		})
	}
}
