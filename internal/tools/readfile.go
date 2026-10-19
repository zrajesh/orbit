package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"orbit/internal/core"
)

const maxReadBytes = 256 * 1024

type ReadFile struct{ sb *Sandbox }

func NewReadFile(sb *Sandbox) *ReadFile { return &ReadFile{sb: sb} }

func (*ReadFile) Definition() core.ToolDefinition {
	return core.ToolDefinition{
		Name:        "read_file",
		Description: "Read a UTF-8 text file inside the workspace and return its contents.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path, relative to the workspace root"}},"required":["path"]}`),
	}
}

func (t *ReadFile) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &in); err != nil || in.Path == "" {
		return "", fmt.Errorf("invalid input: expected {\"path\": string}")
	}
	path, err := t.sb.Resolve(in.Path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxReadBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxReadBytes {
		return string(data[:maxReadBytes]) + "\n[truncated]", nil
	}
	return string(data), nil
}
