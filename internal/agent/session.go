// Package agent implements the agentic loop.
package agent

import (
	"context"
	"fmt"
	"time"

	"orbit/internal/core"
	"orbit/internal/tools"
)

type Config struct {
	Provider core.Provider
	Tools    *tools.Registry
	System   string
	Budget   Budget
}

// Session owns one conversation: its history, usage and loop.
type Session struct {
	cfg      Config
	messages []core.Message
	usage    core.Usage
}

func NewSession(cfg Config) *Session {
	return &Session{cfg: cfg}
}

// Run appends the prompt, drives the loop in a goroutine, and streams events.
// The returned channel is closed when the run ends. Cancelling ctx aborts any
// in-flight provider call and stops the loop.
func (s *Session) Run(ctx context.Context, prompt string) <-chan Event {
	events := make(chan Event)
	s.messages = append(s.messages, core.TextMessage(core.RoleUser, prompt))

	go func() {
		defer close(events)
		s.loop(ctx, events)
	}()
	return events
}

func (s *Session) emit(ctx context.Context, events chan<- Event, ev Event) bool {
	select {
	case events <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Session) fail(ctx context.Context, events chan<- Event, err error) {
	// Errors are delivered even after cancellation so the caller sees partial usage.
	select {
	case events <- Event{Type: EventError, Err: err, Usage: s.usage}:
	case <-time.After(time.Second):
	}
}

func (s *Session) loop(ctx context.Context, events chan<- Event) {
	start := time.Now()

	for turn := 0; ; turn++ {
		if err := ctx.Err(); err != nil {
			s.fail(ctx, events, err)
			return
		}
		if err := s.cfg.Budget.Check(turn, s.usage, time.Since(start)); err != nil {
			s.fail(ctx, events, err)
			return
		}

		resp, err := s.cfg.Provider.Complete(ctx, core.CompletionRequest{
			System:   s.cfg.System,
			Messages: s.messages,
			Tools:    s.cfg.Tools.Definitions(),
		})
		if err != nil {
			s.fail(ctx, events, err)
			return
		}
		s.usage.Add(resp.Usage)
		s.messages = append(s.messages, resp.Message)

		if text := resp.Message.Text(); text != "" {
			if !s.emit(ctx, events, Event{Type: EventText, Text: text}) {
				s.fail(ctx, events, ctx.Err())
				return
			}
		}

		switch resp.StopReason {
		case core.StopEndTurn:
			s.emit(ctx, events, Event{Type: EventRunEnd, StopReason: resp.StopReason, Usage: s.usage})
			return
		case core.StopToolUse:
			if !s.runTools(ctx, events, resp.Message) {
				s.fail(ctx, events, ctx.Err())
				return
			}
		case core.StopMaxTokens:
			s.fail(ctx, events, fmt.Errorf("model hit max_tokens before finishing"))
			return
		default:
			s.fail(ctx, events, fmt.Errorf("unexpected stop reason %q", resp.StopReason))
			return
		}
	}
}

// runTools executes every tool_use block and appends one user message holding all results.
func (s *Session) runTools(ctx context.Context, events chan<- Event, msg core.Message) bool {
	var results []core.ContentBlock

	for _, use := range msg.ToolUses() {
		if !s.emit(ctx, events, Event{Type: EventToolCall, ToolName: use.Name, ToolInput: string(use.Input)}) {
			return false
		}

		out, err := s.cfg.Tools.Execute(ctx, use.Name, use.Input)
		block := core.ContentBlock{Type: core.BlockToolResult, ToolUseID: use.ToolUseID, Content: out}
		if err != nil {
			block.Content, block.IsError = err.Error(), true
		}
		results = append(results, block)

		if !s.emit(ctx, events, Event{Type: EventToolResult, ToolName: use.Name, Text: block.Content, ToolError: block.IsError}) {
			return false
		}
	}

	s.messages = append(s.messages, core.Message{Role: core.RoleUser, Content: results})
	return true
}
