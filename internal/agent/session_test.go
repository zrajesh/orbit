package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"orbit/internal/core"
	"orbit/internal/tools"
)

// scripted replays canned responses and records every request it receives.
type scripted struct {
	responses []core.CompletionResponse
	requests  []core.CompletionRequest
}

func (s *scripted) Complete(ctx context.Context, req core.CompletionRequest) (core.CompletionResponse, error) {
	if err := ctx.Err(); err != nil {
		return core.CompletionResponse{}, err
	}
	s.requests = append(s.requests, req)
	if len(s.responses) == 0 {
		return core.CompletionResponse{}, errors.New("script exhausted")
	}
	r := s.responses[0]
	s.responses = s.responses[1:]
	return r, nil
}

func drain(ch <-chan Event) []Event {
	var evs []Event
	for e := range ch {
		evs = append(evs, e)
	}
	return evs
}

func newRegistry(t *testing.T, dir string) *tools.Registry {
	sb, err := tools.NewSandbox(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFile(sb))
	return reg
}

func TestLoopRunsToolThenEnds(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644)

	p := &scripted{responses: []core.CompletionResponse{
		{
			Message: core.Message{Role: core.RoleAssistant, Content: []core.ContentBlock{
				{Type: core.BlockToolUse, ToolUseID: "t1", Name: "read_file", Input: json.RawMessage(`{"path":"a.txt"}`)},
			}},
			StopReason: core.StopToolUse,
			Usage:      core.Usage{InputTokens: 10, OutputTokens: 5},
		},
		{
			Message:    core.TextMessage(core.RoleAssistant, "it says hello"),
			StopReason: core.StopEndTurn,
			Usage:      core.Usage{InputTokens: 20, OutputTokens: 4},
		},
	}}

	s := NewSession(Config{Provider: p, Tools: newRegistry(t, dir)})
	evs := drain(s.Run(context.Background(), "read a.txt"))

	last := evs[len(evs)-1]
	if last.Type != EventRunEnd || last.Usage.InputTokens != 30 || last.Usage.OutputTokens != 9 {
		t.Fatalf("unexpected final event: %+v", last)
	}

	// The second request must carry the tool result tied to the original tool_use id.
	second := p.requests[1].Messages
	result := second[len(second)-1].Content[0]
	if result.Type != core.BlockToolResult || result.ToolUseID != "t1" || result.Content != "hello" || result.IsError {
		t.Fatalf("bad tool result block: %+v", result)
	}
}

func TestToolErrorIsReturnedToModel(t *testing.T) {
	p := &scripted{responses: []core.CompletionResponse{
		{
			Message: core.Message{Role: core.RoleAssistant, Content: []core.ContentBlock{
				{Type: core.BlockToolUse, ToolUseID: "t1", Name: "read_file", Input: json.RawMessage(`{"path":"../../etc/passwd"}`)},
			}},
			StopReason: core.StopToolUse,
		},
		{Message: core.TextMessage(core.RoleAssistant, "denied"), StopReason: core.StopEndTurn},
	}}

	s := NewSession(Config{Provider: p, Tools: newRegistry(t, t.TempDir())})
	evs := drain(s.Run(context.Background(), "x"))

	if evs[len(evs)-1].Type != EventRunEnd {
		t.Fatalf("run should survive a failing tool: %+v", evs)
	}
	second := p.requests[1].Messages
	if r := second[len(second)-1].Content[0]; !r.IsError {
		t.Fatalf("expected is_error tool result, got %+v", r)
	}
}

func TestBudgetStopsRun(t *testing.T) {
	toolTurn := core.CompletionResponse{
		Message: core.Message{Role: core.RoleAssistant, Content: []core.ContentBlock{
			{Type: core.BlockToolUse, ToolUseID: "t", Name: "read_file", Input: json.RawMessage(`{"path":"."}`)},
		}},
		StopReason: core.StopToolUse,
	}
	p := &scripted{responses: []core.CompletionResponse{toolTurn, toolTurn, toolTurn}}

	s := NewSession(Config{Provider: p, Tools: newRegistry(t, t.TempDir()), Budget: Budget{MaxTurns: 2}})
	evs := drain(s.Run(context.Background(), "loop"))

	if evs[len(evs)-1].Type != EventError || len(p.requests) != 2 {
		t.Fatalf("want error after 2 turns, got %d requests, last=%+v", len(p.requests), evs[len(evs)-1])
	}
}

func TestCancelStopsRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewSession(Config{Provider: &scripted{}, Tools: newRegistry(t, t.TempDir())})
	evs := drain(s.Run(ctx, "x"))

	if len(evs) != 1 || evs[0].Type != EventError || !errors.Is(evs[0].Err, context.Canceled) {
		t.Fatalf("want a single canceled error event, got %+v", evs)
	}
}
