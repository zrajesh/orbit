// Package tools defines the Tool contract, the registry, and built-in tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"orbit/internal/core"
)

// Tool is something the model can ask the agent to run.
type Tool interface {
	Definition() core.ToolDefinition
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Definition().Name] = t
}

// Definitions returns tool definitions sorted by name so requests are deterministic.
func (r *Registry) Definitions() []core.ToolDefinition {
	defs := make([]core.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	return defs
}

func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", name)
	}
	return t.Execute(ctx, input)
}
