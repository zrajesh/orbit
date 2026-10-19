package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"orbit/internal/core"
)

type ListDir struct{ sb *Sandbox }

func NewListDir(sb *Sandbox) *ListDir { return &ListDir{sb: sb} }

func (*ListDir) Definition() core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "list_dir",
		Description: "List the entries of a directory inside the workspace. Directories end with '/'.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"Directory path, relative to the workspace root; '.' for the root"}},"required":["path"]}`),
	}
}

func (t *ListDir) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: expected {\"path\": string}")
	}
	if in.Path == "" {
		in.Path = "."
	}
	path, err := t.sb.Resolve(in.Path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.Name())
		if e.IsDir() {
			b.WriteByte('/')
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}
